package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCommandGrammar(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code string
	}{
		{"setup", []string{"setup"}, "setup_not_implemented"},
		{"doctor", []string{"doctor"}, "doctor_not_implemented"},
		{"service start", []string{"service", "start"}, "service_not_implemented"},
		{"service stop", []string{"service", "stop"}, "service_not_implemented"},
		{"service restart", []string{"service", "restart"}, "service_not_implemented"},
		{"service status", []string{"service", "status"}, "service_not_implemented"},
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
