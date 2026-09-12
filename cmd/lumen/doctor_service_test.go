package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

func TestDoctorUsesLiveTopologyEvidence(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyExternal)
	var supervisorCalls []struct {
		manager  setup.Supervisor
		services []setup.ServiceName
	}
	var probeCalls []string
	old := doctorDepsOverride
	doctorDepsOverride = &doctorDeps{
		hostStatus: func(context.Context, host.Config) (string, error) {
			return setup.StateRunning, nil
		},
		supervisorStatus: func(_ context.Context, manager setup.Supervisor, services []setup.ServiceName) ([]setup.ServiceState, error) {
			supervisorCalls = append(supervisorCalls, struct {
				manager  setup.Supervisor
				services []setup.ServiceName
			}{manager, append([]setup.ServiceName(nil), services...)})
			return []setup.ServiceState{{Name: setup.ServiceHost, State: setup.StateRunning}}, nil
		},
		hermesProbe: func(context.Context, host.Config, *setup.ExternalAdoption) hermesObservation {
			probeCalls = append(probeCalls, "identity", "health", "capabilities")
			return hermesObservation{Ready: true, Version: "1.0.0", IdentityMatch: true, AuthenticationValid: true, CompatibilityValid: true}
		},
		bootStatus: func(context.Context, setup.Supervisor, []setup.ServiceName) (bool, error) {
			return true, nil
		},
	}
	t.Cleanup(func() { doctorDepsOverride = old })
	t.Setenv("LUMEN_SETUP_TOPOLOGY", string(setup.TopologyCombined))
	t.Setenv("LUMEN_SETUP_PROFILE", string(setup.Hardened))
	if err := os.Remove(filepath.Join(f.dataDir, "host.sock")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	report := runForTest([]string{"doctor"})
	if report.Outcome != setup.Ready || report.Topology != setup.TopologyExternal || report.Profile != setup.Development {
		t.Fatalf("doctor report = %#v", report)
	}
	if report.States["host"] != setup.StateRunning || report.States["supervisor"] != setup.StateRunning || report.States["boot"] != setup.StateRunning {
		t.Fatalf("doctor states = %#v", report.States)
	}
	if len(supervisorCalls) != 1 || supervisorCalls[0].manager != setup.SupervisorSystemd || !reflect.DeepEqual(supervisorCalls[0].services, []setup.ServiceName{setup.ServiceHost}) {
		t.Fatalf("supervisor calls = %#v", supervisorCalls)
	}
	if !reflect.DeepEqual(probeCalls, []string{"identity", "health", "capabilities"}) {
		t.Fatalf("Hermes probe order = %#v", probeCalls)
	}
}

func TestDoctorExternalOutageIsDegraded(t *testing.T) {
	newDurableDoctorFixture(t, setup.TopologyExternal)
	old := doctorDepsOverride
	doctorDepsOverride = &doctorDeps{
		hostStatus: func(context.Context, host.Config) (string, error) { return setup.StateRunning, nil },
		supervisorStatus: func(context.Context, setup.Supervisor, []setup.ServiceName) ([]setup.ServiceState, error) {
			return []setup.ServiceState{{Name: setup.ServiceHost, State: setup.StateRunning}}, nil
		},
		hermesProbe: func(context.Context, host.Config, *setup.ExternalAdoption) hermesObservation {
			return hermesObservation{Unavailable: true, Err: errors.New("transport unavailable")}
		},
		bootStatus: func(context.Context, setup.Supervisor, []setup.ServiceName) (bool, error) { return true, nil },
	}
	t.Cleanup(func() { doctorDepsOverride = old })

	report := runForTest([]string{"doctor"})
	if report.Outcome != setup.Degraded || !hasAction(report, "hermes_unavailable") {
		t.Fatalf("outage report = %#v", report)
	}
	if !report.PreviouslyValidated || !report.EndpointIdentityMatch || !report.AuthenticationValid || !report.CompatibilityValid || !report.BindingMatch {
		t.Fatalf("validated evidence was not preserved: %#v", report)
	}
}

func TestDoctorExternalMismatchRequiresAction(t *testing.T) {
	tests := []struct {
		name        string
		observation hermesObservation
	}{
		{name: "verified identity change", observation: hermesObservation{IdentityMatch: false, AuthenticationValid: true, CompatibilityValid: true}},
		{name: "credential rejection", observation: hermesObservation{IdentityMatch: true, AuthenticationValid: false, CompatibilityValid: true}},
		{name: "incompatible capabilities", observation: hermesObservation{IdentityMatch: true, AuthenticationValid: true, CompatibilityValid: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newDurableDoctorFixture(t, setup.TopologyExternal)
			calls := 0
			old := doctorDepsOverride
			doctorDepsOverride = &doctorDeps{
				hostStatus: func(context.Context, host.Config) (string, error) { return setup.StateRunning, nil },
				supervisorStatus: func(context.Context, setup.Supervisor, []setup.ServiceName) ([]setup.ServiceState, error) {
					return []setup.ServiceState{{Name: setup.ServiceHost, State: setup.StateRunning}}, nil
				},
				hermesProbe: func(context.Context, host.Config, *setup.ExternalAdoption) hermesObservation {
					calls++
					return tt.observation
				},
				bootStatus: func(context.Context, setup.Supervisor, []setup.ServiceName) (bool, error) { return true, nil },
			}
			t.Cleanup(func() { doctorDepsOverride = old })

			report := runForTest([]string{"doctor"})
			if report.Outcome != setup.ActionRequired || !hasAction(report, "hermes_binding_mismatch") || calls != 1 {
				t.Fatalf("mismatch report = %#v calls=%d", report, calls)
			}
		})
	}

	t.Run("adoption binding mismatch", func(t *testing.T) {
		f := newDurableDoctorFixture(t, setup.TopologyExternal)
		adoption, err := setup.LoadExternalAdoption(filepath.Join(f.dataDir, "setup", "external-adoption.json"))
		if err != nil {
			t.Fatal(err)
		}
		adoption.EndpointIdentityDigest = fixtureDigest([]byte("changed identity"))
		if err := setup.SaveExternalAdoption(filepath.Join(f.dataDir, "setup", "external-adoption.json"), adoption); err != nil {
			t.Fatal(err)
		}
		report := runForTest([]string{"doctor"})
		if report.Outcome != setup.ActionRequired || !hasAction(report, "setup_identity_mismatch") {
			t.Fatalf("binding mismatch report = %#v", report)
		}
	})
}

func TestDoctorRejectsMissingArtifactEvidence(t *testing.T) {
	newDurableDoctorFixture(t, setup.TopologyExternal)
	rewriteBinding(t, func(binding map[string]any) {
		delete(binding, "artifactPaths")
		delete(binding, "artifactDigests")
	})
	setReadyDoctorDeps(t)

	// A running service must not make an unbound artifact set ready.
	report := runForTest([]string{"doctor"})
	if report.Outcome != setup.ActionRequired || !hasAction(report, "setup_identity_mismatch") {
		 t.Fatalf("missing artifact evidence report = %#v", report)
	}
}

func TestDoctorRejectsMissingOrReplacedDurableArtifact(t *testing.T) {
	for _, tt := range []struct {
		name string
		mutate func(t *testing.T, path string)
	}{
		{name: "missing", mutate: func(t *testing.T, path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "replaced", mutate: func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("replacement"), 0700); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newDurableDoctorFixture(t, setup.TopologyExternal)
			tt.mutate(t, f.artifact)
			setReadyDoctorDeps(t)

			report := runForTest([]string{"doctor"})
			if report.Outcome != setup.ActionRequired || !hasAction(report, "setup_identity_mismatch") {
				t.Fatalf("artifact %s report = %#v", tt.name, report)
			}
		})
	}
}

func TestDoctorRejectsEmptyCombinedEndpointBinding(t *testing.T) {
	newDurableDoctorFixture(t, setup.TopologyCombined)
	setReadyDoctorDeps(t)
	rewriteBinding(t, func(binding map[string]any) {
		delete(binding, "endpointOriginDigest")
	})

	report := runForTest([]string{"doctor"})
	if report.Outcome != setup.ActionRequired || !hasAction(report, "setup_identity_mismatch") {
		t.Fatalf("empty endpoint binding report = %#v", report)
	}
}

func TestServiceRejectsEmptyReferenceEvidence(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyExternal)
	rewriteBinding(t, func(binding map[string]any) {
		delete(binding, "referenceDigests")
	})
	clearFile(t, f.calls)

	report := runForTest([]string{"service", "status"})
	if report.Outcome != setup.ActionRequired || !hasAction(report, "setup_identity_mismatch") {
		t.Fatalf("empty reference evidence report = %#v", report)
	}
	if got, err := os.ReadFile(f.calls); err != nil {
		t.Fatal(err)
	} else if len(got) != 0 {
		t.Fatalf("empty reference evidence made supervisor calls: %q", got)
	}
}

func TestDoctorRejectsCoordinatedExternalReferenceChange(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyExternal)
	setReadyDoctorDeps(t)
	newCredential := filepath.Join(filepath.Dir(f.credential), "replacement.token")
	if err := os.WriteFile(newCredential, []byte("replacement-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	adoptionPath := filepath.Join(f.dataDir, "setup", "external-adoption.json")
	adoption, err := setup.LoadExternalAdoption(adoptionPath)
	if err != nil {
		t.Fatal(err)
	}
	adoption.CredentialFile = newCredential
	if err := setup.SaveExternalAdoption(adoptionPath, adoption); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(f.dataDir, "lumen.json")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(config, &document); err != nil {
		t.Fatal(err)
	}
	hermesConfig, ok := document["hermes"].(map[string]any)
	if !ok {
		t.Fatal("missing Hermes config")
	}
	hermesConfig["bearer_file"] = newCredential
	updated, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, append(updated, '\n'), 0600); err != nil {
		t.Fatal(err)
	}

	report := runForTest([]string{"doctor"})
	if report.Outcome != setup.ActionRequired || !hasAction(report, "setup_identity_mismatch") {
		t.Fatalf("coordinated reference change report = %#v", report)
	}
}

func TestServiceStopRejectsStillRunningService(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyExternal)
	clearFile(t, f.calls)

	report := runForTest([]string{"service", "stop"})
	if report.Outcome != setup.ActionRequired {
		t.Fatalf("still-running stop report = %#v", report)
	}
	if !hasAction(report, "service_unavailable") && !hasAction(report, "service_not_stopped") {
		t.Fatalf("still-running stop actions = %#v", report.Actions)
	}
}

func TestExternalServiceUsesDurableHostOnly(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyExternal)
	t.Setenv("LUMEN_SETUP_TOPOLOGY", string(setup.TopologyCombined))
	t.Setenv("LUMEN_SETUP_PROFILE", string(setup.Hardened))
	clearFile(t, f.calls)

	report := runForTest([]string{"service", "restart"})
	if report.Outcome != setup.Ready || report.Topology != setup.TopologyExternal || report.Profile != setup.Development {
		t.Fatalf("service report = %#v", report)
	}
	calls := readLines(t, f.calls)
	if containsLine(calls, "restart lumen-hermes.service") || !containsLine(calls, "restart lumen-host.service") {
		t.Fatalf("external service calls = %#v", calls)
	}

	t.Run("missing binding makes no calls", func(t *testing.T) {
		d := t.TempDir()
		t.Setenv("LUMEN_DATA_DIR", d)
		clearFile(t, f.calls)
		report := runForTest([]string{"service", "status"})
		calls, err := os.ReadFile(f.calls)
		if err != nil {
			t.Fatal(err)
		}
		if report.Outcome != setup.ActionRequired || len(calls) != 0 {
			t.Fatalf("missing binding report=%#v calls=%q", report, calls)
		}
	})
}

func TestDefaultHermesProbeTreatsLocalConfigurationFailureAsActionRequired(t *testing.T) {
	obs := defaultHermesProbe(context.Background(), host.Config{
		HermesBaseURL:    "http://127.0.0.1:8642",
		HermesProfile:    "development",
		HermesBearerPath: filepath.Join(t.TempDir(), "missing-token"),
	}, nil)
	if obs.Unavailable {
		t.Fatalf("local client configuration was classified as outage: %#v", obs)
	}
}

func TestServiceRejectsIncompleteJournalWithoutSupervisorCalls(t *testing.T) {
	f := newDurableDoctorFixture(t, setup.TopologyExternal)
	if err := os.Remove(filepath.Join(f.dataDir, "setup", "setup-journal.json")); err != nil {
		t.Fatal(err)
	}
	clearFile(t, f.calls)
	report := runForTest([]string{"service", "status"})
	if report.Outcome != setup.ActionRequired || !hasAction(report, "setup_incomplete") {
		t.Fatalf("incomplete service report = %#v", report)
	}
	if got, err := os.ReadFile(f.calls); err != nil {
		t.Fatal(err)
	} else if len(got) != 0 {
		t.Fatalf("incomplete service made supervisor calls: %q", got)
	}
}

type durableDoctorFixture struct {
	dataDir    string
	calls      string
	credential string
	artifact   string
}

func newDurableDoctorFixture(t *testing.T, topology setup.Topology) durableDoctorFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "state")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	setupDir := filepath.Join(dataDir, "setup")
	if err := os.MkdirAll(setupDir, 0700); err != nil {
		t.Fatal(err)
	}
	credential := filepath.Join(root, "hermes.token")
	if err := os.WriteFile(credential, []byte("remote-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, "lumen.artifact")
	artifactBody := []byte("lumen-artifact")
	if err := os.WriteFile(artifact, artifactBody, 0700); err != nil {
		t.Fatal(err)
	}
	artifactPaths := map[string]string{"lumen": artifact}
	artifactDigests := map[string]string{"lumen": fixtureDigest(artifactBody)}
	if topology == setup.TopologyCombined {
		hermesArtifact := filepath.Join(root, "hermes.artifact")
		hermesBody := []byte("hermes-artifact")
		if err := os.WriteFile(hermesArtifact, hermesBody, 0700); err != nil {
			t.Fatal(err)
		}
		artifactPaths["hermes"] = hermesArtifact
		artifactDigests["hermes"] = fixtureDigest(hermesBody)
	}
	endpoint := "http://127.0.0.1:8642"
	request := setup.ConfigRequest{DataDir: dataDir, HermesDir: filepath.Join(dataDir, "hermes"), Profile: setup.Development, Topology: topology, HermesBaseURL: endpoint}
	if topology == setup.TopologyExternal {
		request.HermesCredentialFile = credential
	} else {
		request.HermesBearer = "local-secret"
	}
	if _, err := setup.WriteConfig(request); err != nil {
		t.Fatal(err)
	}
	cfg, err := host.ConfigFromFile(filepath.Join(dataDir, "lumen.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Initialize(cfg); err != nil {
		t.Fatal(err)
	}

	binding := setup.JournalBinding{Profile: setup.Development, Topology: topology, Supervisor: setup.SupervisorSystemd, ArtifactPaths: artifactPaths, ArtifactDigests: artifactDigests, PlanDigest: fixtureDigest([]byte("plan"))}
	if topology == setup.TopologyExternal {
		identity := "loopback:" + endpoint
		binding.EndpointOriginDigest = fixtureDigest([]byte(endpoint))
		binding.EndpointIdentityDigest = fixtureDigest([]byte(identity))
		binding.ReferenceDigests = setup.DigestReferences(map[string]string{"credential": credential})
		adoption := setup.ExternalAdoption{Endpoint: endpoint, EndpointOriginDigest: binding.EndpointOriginDigest, EndpointIdentityDigest: binding.EndpointIdentityDigest, Version: "1.0.0", CredentialFile: credential}
		if err := setup.SaveExternalAdoption(filepath.Join(setupDir, "external-adoption.json"), adoption); err != nil {
			t.Fatal(err)
		}
	} else {
		binding.EndpointOriginDigest, err = setup.EndpointOriginDigest(endpoint)
		if err != nil {
			t.Fatal(err)
		}
		binding.ReferenceDigests = setup.DigestReferences(map[string]string{"credential": filepath.Join(dataDir, "hermes", "hermes.token")})
	}
	j, err := setup.NewJournal(setupDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Bind(binding); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []setup.Stage{setup.Detected, setup.ArtifactsReady, setup.DirectoriesReady, setup.CredentialsReady, setup.ConfigurationReady, setup.HostInitialized, setup.ServicesInstalled, setup.ServicesStarted, setup.Validated} {
		if err := j.Record(setup.StageEvidence{Stage: stage, InputDigest: fixtureDigest([]byte(string(stage))), Profile: setup.Development, PlanDigest: binding.PlanDigest}); err != nil {
			t.Fatal(err)
		}
	}

	calls := filepath.Join(root, "supervisor.calls")
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + calls + "\"\nif [ \"$1\" = status ]; then printf '%s\\n' 'active (running)'; fi\n"
	if err := os.WriteFile(filepath.Join(bin, "systemctl"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(calls, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("LUMEN_DATA_DIR", dataDir)
	return durableDoctorFixture{dataDir: dataDir, calls: calls, credential: credential, artifact: artifact}
}

func hasAction(report setup.Report, code string) bool {
	for _, action := range report.Actions {
		if action.Code == code {
			return true
		}
	}
	return false
}

func clearFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
}

func setReadyDoctorDeps(t *testing.T) {
	t.Helper()
	old := doctorDepsOverride
	doctorDepsOverride = &doctorDeps{
		hostStatus: func(context.Context, host.Config) (string, error) { return setup.StateRunning, nil },
		supervisorStatus: func(_ context.Context, _ setup.Supervisor, services []setup.ServiceName) ([]setup.ServiceState, error) {
			states := make([]setup.ServiceState, 0, len(services))
			for _, name := range services {
				states = append(states, setup.ServiceState{Name: name, State: setup.StateRunning})
			}
			return states, nil
		},
		hermesProbe: func(context.Context, host.Config, *setup.ExternalAdoption) hermesObservation {
			return hermesObservation{Ready: true, IdentityMatch: true, AuthenticationValid: true, CompatibilityValid: true}
		},
		bootStatus: func(context.Context, setup.Supervisor, []setup.ServiceName) (bool, error) { return true, nil },
	}
	t.Cleanup(func() { doctorDepsOverride = old })
}

func rewriteBinding(t *testing.T, mutate func(map[string]any)) {
	t.Helper()
	dataDir := os.Getenv("LUMEN_DATA_DIR")
	path := filepath.Join(dataDir, "setup", "setup-binding.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var binding map[string]any
	if err := json.Unmarshal(raw, &binding); err != nil {
		t.Fatal(err)
	}
	mutate(binding)
	updated, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(updated, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
