package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

func secureTestDir(t *testing.T) string {
	t.Helper()
	root, err := os.UserHomeDir()
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

func TestWaitForHostReadyRequiresAuthenticatedReadyResponse(t *testing.T) {
	root := secureTestDir(t)
	credentialPath := filepath.Join(root, "operator.credential")
	if err := control.WriteCredential(credentialPath, bytes.Repeat([]byte{7}, control.CredentialSize)); err != nil {
		t.Fatal(err)
	}
	cfg := host.Config{DataDir: root, SocketPath: filepath.Join(root, "host.sock"), CredentialPath: credentialPath}
	server, err := control.NewServer(cfg.SocketPath, cfg.CredentialPath, func(_ context.Context, req control.Request) control.Response {
		if req.Command != "status" {
			return control.Response{Error: "unsupported"}
		}
		return control.Response{OK: true, Data: map[string]any{"status": "ready"}}
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if err := waitForHostReady(context.Background(), cfg); err != nil {
		t.Fatalf("ready Host rejected: %v", err)
	}
}

func TestWaitForHostReadyFailsWhenHostIsUnavailable(t *testing.T) {
	root := secureTestDir(t)
	credentialPath := filepath.Join(root, "operator.credential")
	if err := control.WriteCredential(credentialPath, bytes.Repeat([]byte{7}, control.CredentialSize)); err != nil {
		t.Fatal(err)
	}
	cfg := host.Config{DataDir: root, SocketPath: filepath.Join(root, "missing.sock"), CredentialPath: credentialPath}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := waitForHostReady(ctx, cfg); err == nil {
		t.Fatal("unavailable Host reported ready")
	}
}

func TestWaitForServicesReadyRetriesUntilAllOwnedServicesRun(t *testing.T) {
	services := []setup.ServiceName{setup.ServiceHermes, setup.ServiceHost}
	checks := 0
	err := waitForServicesReady(context.Background(), services, func(context.Context) ([]setup.ServiceState, error) {
		checks++
		if checks == 1 {
			return []setup.ServiceState{{Name: setup.ServiceHermes, State: setup.StateRunning}, {Name: setup.ServiceHost, State: setup.StateStopped}}, nil
		}
		return []setup.ServiceState{{Name: setup.ServiceHermes, State: setup.StateRunning}, {Name: setup.ServiceHost, State: setup.StateRunning}}, nil
	})
	if err != nil {
		t.Fatalf("services that became ready within the deadline were rejected: %v", err)
	}
	if checks < 2 {
		t.Fatalf("status checked %d times; expected a retry", checks)
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
		{setup.SupervisorLaunchd, "", ".plist"},
		{setup.SupervisorRunit, "deploy/termux/run", "/run"},
	} {
		defs := serviceDefinitions(tc.manager, setup.TopologyExternal, root)
		if len(defs) != 1 {
			t.Fatalf("%s definitions = %#v", tc.manager, defs)
		}
		if tc.manager == setup.SupervisorLaunchd {
			if defs[0].Source != "" || defs[0].Destination != "" || !strings.HasSuffix(defs[0].Path, tc.ext) {
				t.Fatalf("LaunchAgent must be generated, got %#v", defs[0])
			}
		} else if defs[0].Source != tc.source || !strings.HasSuffix(defs[0].Destination, tc.ext) {
			t.Fatalf("%s definitions = %#v", tc.manager, defs)
		}
	}
	if defs := serviceDefinitions(setup.SupervisorDocker, setup.TopologyCombined, root); len(defs) != 2 || defs[0].Source != "" || defs[1].Source != "" {
		t.Fatalf("Docker definitions should use compose directly: %#v", defs)
	}
	if defs := serviceDefinitions(setup.SupervisorLaunchd, setup.TopologyCombined, root); len(defs) != 0 {
		t.Fatalf("combined Mac definitions must not launch the wrong Hermes service: %#v", defs)
	}
}

func TestWriteLaunchAgentIsPrivateEscapedAndIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := host.Config{
		DataDir: filepath.Join(dir, "space & state"), SocketPath: filepath.Join(dir, "space & state", "host.sock"),
		CredentialPath: filepath.Join(dir, "space & state", "operator.credential"), HermesBaseURL: "https://hermes.example.test",
		HermesProfile: hermes.ProfileHardened, HermesBearerPath: filepath.Join(dir, "hermes.token"),
	}
	path := filepath.Join(dir, "services", "dev.lumen.host.plist")
	if err := writeLaunchAgent(path, "/opt/lumen-host", cfg); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeLaunchAgent(path, "/opt/lumen-host", cfg); err != nil {
		t.Fatalf("identical rerun: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v err=%v", info.Mode(), err)
	}
	if strings.Contains(string(first), "space & state") || !strings.Contains(string(first), "space &amp; state") || strings.Contains(string(first), "token-content") {
		t.Fatalf("LaunchAgent XML escaping or secret handling failed: %s", first)
	}
	if bytes.Contains(first, []byte("LUMEN_REQUIRE_VALIDATED_SETUP")) {
		t.Fatal("staged LaunchAgent unexpectedly requires completed setup")
	}
	validated, err := launchAgentContent("/opt/lumen-host", cfg, "validated")
	if err != nil || !bytes.Contains(validated, []byte("LUMEN_REQUIRE_VALIDATED_SETUP")) {
		t.Fatalf("published LaunchAgent is not activation-gated: err=%v", err)
	}
	if !strings.Contains(string(first), "<key>ProgramArguments</key><array><string>/opt/lumen-host</string><string>serve</string><string>--setup-staging</string></array>") {
		t.Fatal("LaunchAgent ProgramArguments must be plist string elements")
	}
	if bytes.Contains(validated, []byte("--setup-staging")) {
		t.Fatal("published LaunchAgent retained the setup-only staging invocation")
	}
	if err := writeLaunchAgent(path, "/different/lumen-host", cfg); err == nil {
		t.Fatal("different LaunchAgent definition overwrote existing file")
	}
}

func TestLaunchAgentPublicationErrorRequiresPublishedRegularFile(t *testing.T) {
	dir := t.TempDir()
	writeErr := fmt.Errorf("%w: sync", setup.ErrDurabilityUncertain)
	path := filepath.Join(dir, "published.plist")

	if err := os.WriteFile(path, []byte("gated"), 0600); err != nil {
		t.Fatal(err)
	}
	info, statErr := os.Lstat(path)
	got := launchAgentPublicationError(writeErr, info, statErr)
	if !errors.Is(got, setup.ErrLaunchAgentActivationPending) || !errors.Is(got, setup.ErrDurabilityUncertain) {
		t.Fatalf("published regular file error = %v, want both pending and durability errors", got)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "target"), path); err != nil {
		t.Fatal(err)
	}
	info, statErr = os.Lstat(path)
	got = launchAgentPublicationError(writeErr, info, statErr)
	if !errors.Is(got, setup.ErrDurabilityUncertain) || errors.Is(got, setup.ErrLaunchAgentActivationPending) {
		t.Fatalf("symlink publication error = %v, want only durability uncertainty", got)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	info, statErr = os.Lstat(path)
	got = launchAgentPublicationError(writeErr, info, statErr)
	if !errors.Is(got, setup.ErrDurabilityUncertain) || errors.Is(got, setup.ErrLaunchAgentActivationPending) {
		t.Fatalf("missing publication error = %v, want only durability uncertainty", got)
	}
	got = launchAgentPublicationError(writeErr, nil, errors.New("injected stat I/O error"))
	if !errors.Is(got, setup.ErrDurabilityUncertain) || !errors.Is(got, setup.ErrLaunchAgentActivationPending) {
		t.Fatalf("unknown publication state error = %v, want pending and durability uncertainty", got)
	}
	ordinary := errors.New("write failed")
	if got := launchAgentPublicationError(ordinary, nil, nil); !errors.Is(got, ordinary) || errors.Is(got, setup.ErrLaunchAgentActivationPending) {
		t.Fatalf("ordinary publication error = %v, want unchanged error", got)
	}
}

func TestVerifyExistingLaunchAgentDoesNotPublishMissingDefinition(t *testing.T) {
	dir := secureTestDir(t)
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "dev.lumen.host.plist")
	content := []byte("expected plist")
	if err := verifyExistingLaunchAgent(path, content); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preflight created an auto-load definition: %v", err)
	}
	if err := writePrivateLaunchAgent(path, content); err != nil {
		t.Fatal(err)
	}
	if err := verifyExistingLaunchAgent(path, content); err != nil {
		t.Fatal(err)
	}
	if err := verifyExistingLaunchAgent(path, []byte("different plist")); err == nil {
		t.Fatal("preflight accepted a conflicting auto-load definition")
	}
}

func TestVerifyExistingLaunchAgentRejectsMacOSACLOnDefinition(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS ACLs are platform-specific")
	}
	dir := secureTestDir(t)
	path := filepath.Join(dir, "dev.lumen.host.plist")
	content := []byte("expected plist")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("/bin/chmod", "+a", "everyone allow write", path).CombinedOutput(); err != nil {
		t.Fatalf("could not create ACL fixture: %v (%s)", err, output)
	}
	if err := verifyExistingLaunchAgent(path, content); err == nil {
		t.Fatal("ACL-writable LaunchAgent definition was accepted")
	}
}

func TestLaunchAgentUsesStagedDefinitionBeforeAutoLoadPath(t *testing.T) {
	dataDir := filepath.Join(secureTestDir(t), "space")
	launchAgents := filepath.Join(secureTestDir(t), "Library", "LaunchAgents")
	stage, published := launchAgentDefinitionPaths(dataDir, launchAgents)
	if stage == published || !strings.HasPrefix(stage, filepath.Join(dataDir, "setup", "services")+string(os.PathSeparator)) {
		t.Fatalf("staged and auto-load paths are not isolated: stage=%q published=%q", stage, published)
	}
	if published != filepath.Join(launchAgents, "dev.lumen.host.plist") {
		t.Fatalf("published path=%q", published)
	}
}

func TestLaunchAgentDirectoryIsAutoLoadedAndOwnerControlled(t *testing.T) {
	home := t.TempDir()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	dir, err := launchAgentDirectoryForHome(home)
	if err != nil {
		t.Fatal(err)
	}
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(resolvedHome, "Library", "LaunchAgents")
	if dir != want {
		t.Fatalf("LaunchAgent directory = %q, want %q", dir, want)
	}
	if err := os.Chmod(dir, 0770); err != nil {
		t.Fatal(err)
	}
	if _, err := launchAgentDirectoryForHome(home); err == nil {
		t.Fatal("group-writable LaunchAgents directory was accepted")
	}
}

func TestCanonicalAccountHomeIgnoresHomeEnvironment(t *testing.T) {
	if os.Getenv("LUMEN_CANONICAL_HOME_CHILD") == "1" {
		got, err := canonicalAccountHome()
		if !accountHomeLookupUsesSystemRecord {
			if err == nil {
				t.Fatal("unsafe osusergo lookup was accepted")
			}
			return
		}
		if err != nil {
			t.Fatal("account-home lookup failed")
		}
		if got != os.Getenv("LUMEN_EXPECTED_ACCOUNT_HOME") {
			t.Fatal("canonical account home ignored the system account record")
		}
		return
	}
	account, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	redirectedHome := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCanonicalAccountHomeIgnoresHomeEnvironment$")
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "HOME" && key != "LUMEN_CANONICAL_HOME_CHILD" && key != "LUMEN_EXPECTED_ACCOUNT_HOME" {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env,
		"HOME="+redirectedHome,
		"LUMEN_CANONICAL_HOME_CHILD=1",
		"LUMEN_EXPECTED_ACCOUNT_HOME="+filepath.Clean(account.HomeDir),
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fresh-process home lookup failed with redirected HOME: %v\n%s", err, output)
	}
}

func TestLaunchAgentArtifactMustBeUnreplaceableByOtherUsers(t *testing.T) {
	body := []byte("verified lumen host artifact")
	manifest := setup.Artifact{
		Name: "lumen", Version: "1.0.0", OS: "darwin", Architecture: runtime.GOARCH,
		Profile: setup.Development, URL: "https://downloads.example.test/lumen", Size: int64(len(body)),
		SHA256: fixtureDigest(body), ContractVersion: 1, ExecutableMode: 0700, Ownership: "lumen",
	}
	secureDir := secureTestDir(t)
	securePath := filepath.Join(secureDir, "lumen-host")
	if err := os.WriteFile(securePath, body, 0700); err != nil {
		t.Fatal(err)
	}
	if got, err := verifiedLaunchAgentArtifactPath(securePath, manifest); err != nil || got == "" {
		t.Fatalf("secure artifact rejected: path=%q err=%v", got, err)
	}
	sharedDir := filepath.Join(secureDir, "shared")
	if err := os.Mkdir(sharedDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sharedDir, 0770); err != nil {
		t.Fatal(err)
	}
	sharedPath := filepath.Join(sharedDir, "lumen-host")
	if err := os.WriteFile(sharedPath, body, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := verifiedLaunchAgentArtifactPath(sharedPath, manifest); err == nil {
		t.Fatal("replaceable artifact path was accepted")
	}
	linkPath := filepath.Join(secureDir, "lumen-link")
	if err := os.Symlink(securePath, linkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := verifiedLaunchAgentArtifactPath(linkPath, manifest); err == nil {
		t.Fatal("symlinked artifact path was accepted")
	}
}

func TestLaunchAgentArtifactRejectsMacOSACLOnParent(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS ACLs are platform-specific")
	}
	root := secureTestDir(t)
	dir := filepath.Join(root, "acl")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("/bin/chmod", "+a", "everyone allow add_file,delete_child", dir).CombinedOutput(); err != nil {
		t.Fatalf("could not create ACL fixture: %v (%s)", err, output)
	}
	acl, err := exec.Command("/bin/ls", "-lde", dir).Output()
	if err != nil || !strings.Contains(string(acl), "everyone allow add_file,delete_child") {
		t.Fatalf("ACL fixture was not installed: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0022 != 0 {
		t.Fatalf("ACL fixture unexpectedly has group/other write bits: mode=%v", info.Mode().Perm())
	}
	body := []byte("verified lumen host artifact")
	path := filepath.Join(dir, "lumen-host")
	if err := os.WriteFile(path, body, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := setup.Artifact{
		Name: "lumen", Version: "1.0.0", OS: "darwin", Architecture: runtime.GOARCH,
		Profile: setup.Development, URL: "https://downloads.example.test/lumen", Size: int64(len(body)),
		SHA256: fixtureDigest(body), ContractVersion: 1, ExecutableMode: 0700, Ownership: "lumen",
	}
	if _, err := verifiedLaunchAgentArtifactPath(path, manifest); err == nil {
		t.Fatal("artifact beneath ACL-controlled parent was accepted")
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
	d := filepath.Join(secureTestDir(t), "state")
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	t.Setenv("LUMEN_SETUP_TOPOLOGY", "combined")
	t.Setenv("LUMEN_SUPERVISOR", string(setup.SupervisorSystemd))
	report := runForTest([]string{"setup"})
	if runtime.GOOS == "darwin" {
		if report.Outcome != setup.ActionRequired || len(report.Actions) != 1 || report.Actions[0].Code != "macos_combined_gateway_unsupported" {
			t.Fatalf("combined Mac setup must fail before mutation: %#v", report)
		}
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Fatalf("unsupported Mac setup mutated the data directory: %v", err)
		}
		return
	}
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
	if report.Outcome != setup.ActionRequired || len(report.Actions) != 1 || report.Actions[0].Code != "setup_identity_mismatch" {
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

func TestSetupExternalBindingRejectsImmutableEvidenceBeforeRecreatingMissingAdoption(t *testing.T) {
	h := newSetupJourneyFixture(t, setup.TopologyExternal)
	firstEndpoint, hermesCalls := newExternalHermesServerWithCalls(t)
	t.Setenv("LUMEN_HERMES_BASE_URL", firstEndpoint)
	t.Setenv("LUMEN_HERMES_IDENTITY", "missing-adoption-first")
	if report := runForTest([]string{"setup"}); report.Outcome != setup.Ready {
		t.Fatalf("initial external setup report = %#v", report)
	}
	adoptionPath := filepath.Join(h.dataDir, "setup", "external-adoption.json")
	if err := os.Remove(adoptionPath); err != nil {
		t.Fatal(err)
	}
	before := snapshotRegularFiles(t, h.dataDir)
	beforeCalls := hermesCalls.Load()

	secondCredential := filepath.Join(secureTestDir(t), "remote.token")
	if err := os.WriteFile(secondCredential, []byte("different-remote-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUMEN_HERMES_CREDENTIAL_FILE", secondCredential)
	t.Setenv("LUMEN_HERMES_IDENTITY", "missing-adoption-second")
	report := runForTest([]string{"setup"})
	if report.Outcome != setup.ActionRequired {
		t.Fatalf("changed external setup report = %#v", report)
	}
	if after := snapshotRegularFiles(t, h.dataDir); !reflect.DeepEqual(after, before) {
		t.Fatalf("rejected setup mutated data directory: before=%v after=%v", before, after)
	}
	if afterCalls := hermesCalls.Load(); afterCalls != beforeCalls {
		t.Fatalf("rejected credential substitution contacted Hermes: before=%d after=%d", beforeCalls, afterCalls)
	}
}

func snapshotRegularFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			contents, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			files[rel] = contents
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
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

func TestExternalSetupSelectsHostFromCombinedReleaseManifest(t *testing.T) {
	newSetupJourneyFixture(t, setup.TopologyCombined)
	state := &setupState{
		topology: setup.TopologyExternal,
		profile:  setup.Development,
		plan: setup.PlanResult{
			Topology: setup.TopologyExternal, Platform: setup.PlatformLinux, Architecture: runtime.GOARCH,
		},
	}
	selected, err := state.selectedArtifacts()
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected["lumen"].Name != "lumen" {
		t.Fatalf("external topology selected artifacts = %#v, want Host only", selected)
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
	previousPlatform, previousHostStatus := setupPlatformOverride, setupHostStatusOverride
	setupPlatformOverride = "linux"
	setupHostStatusOverride = func(context.Context, host.Config) (string, error) { return setup.StateRunning, nil }
	t.Cleanup(func() {
		setupPlatformOverride, setupHostStatusOverride = previousPlatform, previousHostStatus
	})
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
	osName, arch := "linux", runtime.GOARCH
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

func TestSetupCompletesAndRerunsWithActiveHostStore(t *testing.T) {
	newSetupJourneyFixture(t, setup.TopologyExternal)
	t.Setenv("LUMEN_HERMES_BASE_URL", newExternalHermesServer(t))
	t.Setenv("LUMEN_HERMES_IDENTITY", "fixture-leaf")
	var active *store.Store
	setupHostStatusOverride = func(_ context.Context, cfg host.Config) (string, error) {
		if active == nil {
			var err error
			active, err = store.Open(filepath.Join(cfg.DataDir, "state.json"), filepath.Join(cfg.DataDir, "state.key"))
			if err != nil {
				return "", err
			}
		}
		return setup.StateRunning, nil
	}
	t.Cleanup(func() {
		if active != nil {
			_ = active.Close()
		}
	})

	for attempt := 1; attempt <= 2; attempt++ {
		report := runForTest([]string{"setup"})
		if report.Outcome != setup.Ready {
			t.Fatalf("setup attempt %d with active Host store = %#v", attempt, report)
		}
	}
	if active == nil {
		t.Fatal("test did not hold the active Host state-store lock")
	}
}

func newExternalHermesServer(t *testing.T) string {
	t.Helper()
	endpoint, _ := newExternalHermesServerWithCalls(t)
	return endpoint
}

func newExternalHermesServerWithCalls(t *testing.T) (string, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
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
	return "http://" + listener.Addr().String(), &calls
}

func fixtureDigest(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}
