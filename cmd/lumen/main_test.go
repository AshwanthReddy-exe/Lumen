package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

func secureTestDir(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	d, err := os.MkdirTemp(root, ".lumen-cli-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(d) })
	return d
}

func TestHardenedCombinedWriteConfigReceivesTLSEnvironment(t *testing.T) {
	root := secureTestDir(t)
	d := filepath.Join(root, "state")
	ca, cert, key := filepath.Join(root, "ca.pem"), filepath.Join(root, "client.crt"), filepath.Join(root, "client.key")
	for path, body := range map[string]string{ca: "ca", cert: "cert", key: "key"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("LUMEN_HERMES_CA_FILE", ca)
	t.Setenv("LUMEN_HERMES_CLIENT_CERT_FILE", cert)
	t.Setenv("LUMEN_HERMES_CLIENT_KEY_FILE", key)
	t.Setenv("LUMEN_HERMES_SERVER_CERT_PIN", strings.Repeat("a", 64))
	t.Setenv("LUMEN_HERMES_BASE_URL", "https://hermes.example")
	s := &setupState{dataDir: d, profile: setup.Hardened, topology: setup.TopologyCombined}
	if err := s.writeConfig(); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{filepath.Join(d, "hermes", "ca.pem"): "ca", filepath.Join(d, "hermes", "client.crt"): "cert", filepath.Join(d, "hermes", "client.key"): "key"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("TLS file %s = %q, %v", path, got, err)
		}
	}
}

func TestServiceDefinitionsMatchSupervisorFormat(t *testing.T) {
	root := filepath.Join(secureTestDir(t), "services")
	for _, tc := range []struct {
		manager setup.Supervisor
		source  string
		ext     string
	}{
		{setup.SupervisorSystemd, "deploy/systemd/lumen-host.service", ".service"},
		{setup.SupervisorLaunchd, "deploy/launchd/dev.lumen.host.plist", ".plist"},
		{setup.SupervisorRunit, "deploy/termux/run", "/run"},
	} {
		defs := serviceDefinitions(tc.manager, setup.TopologyExternal, root)
		if len(defs) != 1 || defs[0].Source != tc.source || !strings.HasSuffix(defs[0].Destination, tc.ext) {
			t.Fatalf("%s definitions = %#v", tc.manager, defs)
		}
	}
	if defs := serviceDefinitions(setup.SupervisorDocker, setup.TopologyCombined, root); len(defs) != 2 || defs[0].Source != "" || defs[1].Source != "" {
		t.Fatalf("Docker definitions should use compose directly: %#v", defs)
	}
}

func TestCommandGrammar(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code string
	}{
		{"setup", []string{"setup"}, "configuration_required"},
		{"doctor", []string{"doctor"}, "configuration_required"},
		{"service start", []string{"service", "start"}, "configuration_required"},
		{"service stop", []string{"service", "stop"}, "configuration_required"},
		{"service restart", []string{"service", "restart"}, "configuration_required"},
		{"service status", []string{"service", "status"}, "configuration_required"},
		{"connect", []string{"connect"}, ""},
		{"empty", nil, "invalid_command"},
		{"unknown", []string{"wat"}, "invalid_command"},
		{"extra", []string{"setup", "now"}, "invalid_command"},
		{"bare service", []string{"service"}, "invalid_command"},
		{"unknown service action", []string{"service", "wat"}, "invalid_command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := runForTest(tt.args)
			if tt.name == "connect" {
				return
			}
			if len(report.Actions) != 1 || report.Actions[0].Code != tt.code {
				t.Fatalf("got %#v, want action %q", report, tt.code)
			}
		})
	}
}

func TestJSONOutputDoesNotLeakEnvironment(t *testing.T) {
	const secret = "setup-secret-value"
	_ = os.Setenv("LUMEN_TEST_SECRET", secret)
	t.Cleanup(func() { _ = os.Unsetenv("LUMEN_TEST_SECRET") })
	body, err := json.Marshal(runForTest([]string{"setup"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), secret) || strings.Contains(string(body), "LUMEN_TEST_SECRET") {
		t.Fatalf("output leaked environment value or name: %s", body)
	}
}

func TestConnectDoesNotMutateStateRoot(t *testing.T) {
	root := secureTestDir(t)
	t.Setenv("HOME", root)
	t.Setenv("LUMEN_DATA_DIR", root)
	if report := runForTest([]string{"connect"}); report.Outcome != actionRequired {
		t.Fatalf("unexpected report: %#v", report)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("connect mutated state root: %v", entries)
	}
}

func TestConnectIsReservedWithoutPairing(t *testing.T) {
	report := runForTest([]string{"connect"})
	if report.Outcome != actionRequired || report.AvailableInBlock != 3 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestSetupComposesPlannerAndJournal(t *testing.T) {
	d := secureTestDir(t)
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	t.Setenv("LUMEN_SETUP_TOPOLOGY", "combined")
	report := runForTest([]string{"setup"})
	if report.Actions == nil || report.Actions[0].Code == "setup_adapters_required" {
		t.Fatalf("setup still uses placeholder composition: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(d, "setup", "setup-journal.json")); err != nil {
		t.Fatalf("setup did not create durable journal: %v", err)
	}
}

func TestDoctorComposesWholeDeploymentStates(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyCombined)
	t.Setenv("LUMEN_SETUP_TOPOLOGY", "external")
	t.Setenv("LUMEN_SETUP_PROFILE", "hardened")
	report := runForTest([]string{"doctor"})
	if report.Actions == nil || report.Actions[0].Code == "setup_adapters_required" {
		t.Fatalf("doctor still uses placeholder composition: %#v", report)
	}
	if report.States == nil || report.States["host"] == "" {
		t.Fatalf("doctor omitted normalized states: %#v", report)
	}
	body, err := json.Marshal(report)
	if err != nil || strings.Contains(string(body), f.dataDir) {
		t.Fatalf("doctor leaked path or failed to encode: %s (%v)", body, err)
	}
}

func TestAggregateSupervisorStatesRequiresAllServicesRunning(t *testing.T) {
	tests := []struct {
		name   string
		states []setup.ServiceState
		ready  bool
		state  string
	}{
		{
			name: "all required services running",
			states: []setup.ServiceState{
				{Name: setup.ServiceHermes, State: setup.StateRunning},
				{Name: setup.ServiceHost, State: setup.StateRunning},
			},
			ready: true,
			state: setup.StateRunning,
		},
		{
			name: "host stopped",
			states: []setup.ServiceState{
				{Name: setup.ServiceHermes, State: setup.StateRunning},
				{Name: setup.ServiceHost, State: setup.StateStopped},
			},
			state: setup.StateStopped,
		},
		{
			name: "hermes unknown",
			states: []setup.ServiceState{
				{Name: setup.ServiceHermes, State: setup.StateUnknown},
				{Name: setup.ServiceHost, State: setup.StateRunning},
			},
			state: setup.StateUnknown,
		},
		{
			name: "duplicate hermes service",
			states: []setup.ServiceState{
				{Name: setup.ServiceHermes, State: setup.StateRunning},
				{Name: setup.ServiceHermes, State: setup.StateRunning},
			},
			state: setup.StateUnknown,
		},
		{
			name: "unrelated service",
			states: []setup.ServiceState{
				{Name: setup.ServiceName("other"), State: setup.StateRunning},
				{Name: setup.ServiceHost, State: setup.StateRunning},
			},
			state: setup.StateUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready, state := aggregateSupervisorStates(tt.states, nil)
			if ready != tt.ready || state != tt.state {
				t.Fatalf("aggregateSupervisorStates() = (%t, %q), want (%t, %q)", ready, state, tt.ready, tt.state)
			}
		})
	}
}

func TestPublicCommandsDoNotExposeUntrustedMetadata(t *testing.T) {
	d := secureTestDir(t)
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	t.Setenv("LUMEN_LUMEN_VERSION", "/Users/alice/secret-token")
	for _, command := range []string{"setup", "doctor"} {
		body, err := json.Marshal(runForTest([]string{command}))
		if err != nil || strings.Contains(string(body), "alice") || strings.Contains(string(body), "secret-token") {
			t.Fatalf("%s leaked metadata: %s (%v)", command, body, err)
		}
	}
}

func TestCombinedServiceControlsBothServices(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyCombined)
	calls := f.calls
	clearFile(t, calls)

	report := runForTest([]string{"service", "restart"})
	if report.Outcome != setup.Ready || report.Topology != setup.TopologyCombined {
		t.Fatalf("service report = %#v", report)
	}
	got := readLines(t, calls)
	if !containsLine(got, "restart lumen-hermes.service") || !containsLine(got, "restart lumen-host.service") {
		t.Fatalf("combined service calls = %#v", got)
	}
}

func TestExternalServiceControlsHostOnly(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyExternal)
	calls := f.calls
	clearFile(t, calls)

	report := runForTest([]string{"service", "restart"})
	if report.Outcome != setup.Ready || report.Topology != setup.TopologyExternal {
		t.Fatalf("service report = %#v", report)
	}
	got := readLines(t, calls)
	if containsLine(got, "restart lumen-hermes.service") || !containsLine(got, "restart lumen-host.service") {
		t.Fatalf("external service calls = %#v", got)
	}
}

func fakeSystemd(t *testing.T) (string, string) {
	t.Helper()
	d := secureTestDir(t)
	calls := filepath.Join(d, "calls")
	tool := filepath.Join(d, "systemctl")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + calls + "\"\nif [ \"$1\" = status ]; then printf '%s\\n' 'active (running)'; fi\n"
	if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", d+string(os.PathListSeparator)+os.Getenv("PATH"))
	return d, calls
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

func TestSetupRunsCombinedPlan(t *testing.T) {
	h := newSetupJourneyFixture(t, setup.TopologyCombined)
	report := runForTest([]string{"setup"})
	if report.Outcome != setup.Ready || report.Topology != setup.TopologyCombined {
		t.Fatalf("combined setup report = %#v", report)
	}
	for _, path := range []string{filepath.Join(h.dataDir, "lumen.json"), filepath.Join(h.dataDir, "hermes", "hermes.json"), filepath.Join(h.dataDir, "hermes", "hermes.token")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("combined setup missing %s: %v", path, err)
		}
	}
	journal, err := setup.NewJournal(filepath.Join(h.dataDir, "setup"))
	if err != nil || journal.Next() != setup.Validated {
		t.Fatalf("combined journal = %#v err=%v", journal, err)
	}
	for _, want := range []string{"enable lumen-hermes.service", "enable lumen-host.service", "start lumen-hermes.service", "start lumen-host.service"} {
		if !containsLine(readLines(t, h.calls), want) {
			t.Fatalf("combined supervisor calls missing %q: %v", want, readLines(t, h.calls))
		}
	}
}

func TestSetupRunsExternalPlan(t *testing.T) {
	h := newSetupJourneyFixture(t, setup.TopologyExternal)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("external Hermes received lifecycle method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
		case "/v1/capabilities":
			_, _ = w.Write([]byte(`{"object":"capabilities","platform":"test","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":true,"run_events_sse":true,"run_approval":true,"run_stop":true}}`))
		default:
			http.NotFound(w, r)
		}
	})
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	remote := &http.Server{Handler: handler}
	go func() { _ = remote.Serve(listener) }()
	defer remote.Close()
	t.Setenv("LUMEN_HERMES_BASE_URL", "http://"+listener.Addr().String())
	t.Setenv("LUMEN_HERMES_IDENTITY", "fixture-leaf")
	report := runForTest([]string{"setup"})
	if report.Outcome != setup.Ready || report.Topology != setup.TopologyExternal {
		t.Fatalf("external setup report = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(h.dataDir, "lumen.json")); err != nil {
		t.Fatalf("external setup missing Host config: %v", err)
	}
	for _, path := range []string{filepath.Join(h.dataDir, "hermes"), filepath.Join(h.dataDir, "hermes", "hermes.json"), filepath.Join(h.dataDir, "hermes", "hermes.token")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("external setup created Hermes-owned path %s: %v", path, err)
		}
	}
	adoption, err := setup.LoadExternalAdoption(filepath.Join(h.dataDir, "setup", "external-adoption.json"))
	if err != nil || adoption.CredentialFile != h.credential {
		t.Fatalf("external adoption = %#v err=%v", adoption, err)
	}
	if want := fixtureDigest([]byte("loopback:" + adoption.Endpoint)); adoption.EndpointIdentityDigest != want {
		t.Fatalf("external identity digest = %q, want verified loopback identity digest %q", adoption.EndpointIdentityDigest, want)
	}
	for _, line := range readLines(t, h.calls) {
		if strings.Contains(line, "hermes") {
			t.Fatalf("external supervisor controlled Hermes: %q", line)
		}
	}
}

func TestSetupRerunPreservesIdentityAndCredentials(t *testing.T) {
	h := newSetupJourneyFixture(t, setup.TopologyExternal)
	firstEndpoint := newExternalHermesServer(t)
	t.Setenv("LUMEN_HERMES_BASE_URL", firstEndpoint)
	t.Setenv("LUMEN_HERMES_IDENTITY", "untrusted-first-identity")
	if report := runForTest([]string{"setup"}); report.Outcome != setup.Ready {
		t.Fatalf("initial external setup report = %#v", report)
	}

	adoptionPath := filepath.Join(h.dataDir, "setup", "external-adoption.json")
	configPath := filepath.Join(h.dataDir, "lumen.json")
	journalPath := filepath.Join(h.dataDir, "setup", "setup-journal.json")
	bindingPath := filepath.Join(h.dataDir, "setup", "setup-binding.json")
	beforeAdoption, err := os.ReadFile(adoptionPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeJournal, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeBinding, err := os.ReadFile(bindingPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeCalls, err := os.ReadFile(h.calls)
	if err != nil {
		t.Fatal(err)
	}

	secondCredential := filepath.Join(secureTestDir(t), "remote.token")
	if err := os.WriteFile(secondCredential, []byte("different-remote-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	secondEndpoint := newExternalHermesServer(t)
	t.Setenv("LUMEN_HERMES_BASE_URL", secondEndpoint)
	t.Setenv("LUMEN_HERMES_IDENTITY", "untrusted-second-identity")
	t.Setenv("LUMEN_HERMES_CREDENTIAL_FILE", secondCredential)
	report := runForTest([]string{"setup"})
	if report.Outcome != setup.ActionRequired || len(report.Actions) != 1 || report.Actions[0].Code != "external_adoption_failed" {
		t.Fatalf("changed external setup report = %#v", report)
	}

	for path, before := range map[string][]byte{
		adoptionPath: beforeAdoption,
		configPath:   beforeConfig,
		journalPath:  beforeJournal,
		bindingPath:  beforeBinding,
		h.calls:      beforeCalls,
	} {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s after rejected rerun: %v", path, err)
		}
		if !bytes.Equal(after, before) {
			t.Fatalf("rejected rerun mutated %s", path)
		}
	}
}

func TestSetupExternalAdoptionFailureDoesNotMutateState(t *testing.T) {
	root := secureTestDir(t)
	dataDir := filepath.Join(root, "state")
	credential := filepath.Join(root, "credential")
	if err := os.WriteFile(credential, []byte("remote-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUMEN_DATA_DIR", dataDir)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	t.Setenv("LUMEN_SETUP_TOPOLOGY", "external")
	t.Setenv("LUMEN_SUPERVISOR", string(setup.SupervisorSystemd))
	t.Setenv("LUMEN_HERMES_CREDENTIAL_FILE", credential)
	t.Setenv("LUMEN_HERMES_BASE_URL", "http://127.0.0.1:1")

	report := runForTest([]string{"setup"})
	if report.Outcome != setup.ActionRequired || len(report.Actions) != 1 || report.Actions[0].Code != "external_adoption_failed" {
		t.Fatalf("external adoption report = %#v", report)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("failed external adoption mutated state directory: %v", err)
	}
}

func TestSetupRequiresExplicitTopologyWithoutMutation(t *testing.T) {
	dataDir := filepath.Join(secureTestDir(t), "state")
	t.Setenv("LUMEN_DATA_DIR", dataDir)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	t.Setenv("LUMEN_SETUP_TOPOLOGY", "")
	report := runForTest([]string{"setup"})
	if report.Outcome != setup.ActionRequired || len(report.Actions) != 1 || report.Actions[0].Code != "invalid_topology" {
		t.Fatalf("missing topology report = %#v", report)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("missing topology mutated state: %v", err)
	}
}

func TestExternalPersonalAlphaUsesHardenedHermesTransport(t *testing.T) {
	if got := externalHermesProfile(setup.PersonalAlpha); got != hermes.ProfileHardened {
		t.Fatalf("personal-alpha external Hermes profile = %q", got)
	}
}

type setupJourneyFixture struct {
	dataDir, calls, credential string
}

func newSetupJourneyFixture(t *testing.T, topology setup.Topology) setupJourneyFixture {
	t.Helper()
	root := secureTestDir(t)
	dataDir := filepath.Join(root, "state")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	manifestDir := filepath.Join(root, "deploy")
	if err := os.MkdirAll(filepath.Join(manifestDir, "systemd"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"lumen-host.service", "lumen-hermes.service"} {
		if err := os.WriteFile(filepath.Join(manifestDir, "systemd", name), []byte("[Service]\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	lumenBody, hermesBody := []byte("lumen-fixture"), []byte("hermes-fixture")
	lumenPath, hermesPath := filepath.Join(root, "lumen"), filepath.Join(root, "hermes")
	if err := os.WriteFile(lumenPath, lumenBody, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hermesPath, hermesBody, 0700); err != nil {
		t.Fatal(err)
	}
	lumenDigest := fixtureDigest(lumenBody)
	hermesDigest := fixtureDigest(hermesBody)
	osName, arch := runtime.GOOS, runtime.GOARCH
	if osName == "darwin" {
		osName = "darwin"
	}
	manifest := fmt.Sprintf(`{"schemaVersion":1,"topology":%q,"artifacts":[{"name":"lumen","version":"1.0.0","os":%q,"architecture":%q,"profile":"development","url":"https://downloads.lumen.dev/lumen/1.0.0/%s-%s","size":%d,"sha256":%q,"contractVersion":1,"executableMode":448,"ownership":"lumen"}`,
		topology, osName, arch, osName, arch, len(lumenBody), lumenDigest)
	if topology == setup.TopologyCombined {
		manifest += fmt.Sprintf(`,{"name":"hermes","version":"1.0.0","os":%q,"architecture":%q,"profile":"development","url":"https://downloads.lumen.dev/hermes/1.0.0/%s-%s","size":%d,"sha256":%q,"contractVersion":1,"executableMode":448,"ownership":"hermes"}`,
			osName, arch, osName, arch, len(hermesBody), hermesDigest)
	}
	manifest += "]}"
	manifestPath := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(root, "supervisor.calls")
	toolDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(toolDir, 0700); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(toolDir, "systemctl")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + calls + "\"\nif [ \"$1\" = status ]; then printf '%s\\n' 'active (running)'; fi\n"
	if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	credential := filepath.Join(root, "remote.token")
	if err := os.WriteFile(credential, []byte("remote-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUMEN_DATA_DIR", dataDir)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	t.Setenv("LUMEN_SETUP_TOPOLOGY", string(topology))
	t.Setenv("LUMEN_SUPERVISOR", string(setup.SupervisorSystemd))
	t.Setenv("LUMEN_MANIFEST", manifestPath)
	t.Setenv("LUMEN_LUMEN_ARTIFACT", lumenPath)
	t.Setenv("LUMEN_HERMES_ARTIFACT", hermesPath)
	t.Setenv("LUMEN_HERMES_CREDENTIAL_FILE", credential)
	t.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(root)
	return setupJourneyFixture{dataDir: dataDir, calls: calls, credential: credential}
}

func newExternalHermesServer(t *testing.T) string {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("external Hermes received lifecycle method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
		case "/v1/capabilities":
			_, _ = w.Write([]byte(`{"object":"capabilities","platform":"test","auth":{"type":"bearer","required":true},"features":{"run_submission":true,"run_status":true,"run_events_sse":true,"run_approval":true,"run_stop":true}}`))
		default:
			http.NotFound(w, r)
		}
	})
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	remote := &http.Server{Handler: handler}
	go func() { _ = remote.Serve(listener) }()
	t.Cleanup(func() { _ = remote.Close() })
	return "http://" + listener.Addr().String()
}

func fixtureDigest(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}
