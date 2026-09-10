package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

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
	root := t.TempDir()
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
	d := t.TempDir()
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	report := runForTest([]string{"setup"})
	if report.Actions == nil || report.Actions[0].Code == "setup_adapters_required" {
		t.Fatalf("setup still uses placeholder composition: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(d, "setup", "setup-journal.json")); err != nil {
		t.Fatalf("setup did not create durable journal: %v", err)
	}
}

func TestDoctorComposesWholeDeploymentStates(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LUMEN_DATA_DIR", d)
	t.Setenv("LUMEN_SETUP_PROFILE", "development")
	report := runForTest([]string{"doctor"})
	if report.Actions == nil || report.Actions[0].Code == "setup_adapters_required" {
		t.Fatalf("doctor still uses placeholder composition: %#v", report)
	}
	if report.States == nil || report.States["host"] == "" {
		t.Fatalf("doctor omitted normalized states: %#v", report)
	}
	body, err := json.Marshal(report)
	if err != nil || strings.Contains(string(body), d) {
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
	d := t.TempDir()
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
