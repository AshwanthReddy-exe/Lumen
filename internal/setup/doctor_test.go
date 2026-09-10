package setup

import (
	"context"
	"strings"
	"testing"
)

func TestDoctorDistinguishesDegradedAndRedactsDetails(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{HostReady: true, HermesReady: false, SupervisorReady: true, BootReady: true, CredentialsReady: true, IsolationReady: true}
	}}.Check(context.Background())
	if r.Outcome != Degraded {
		t.Fatalf("got %#v", r)
	}
	r = Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{HostReady: false, HermesReady: true, SupervisorReady: false, BootReady: false, CredentialsReady: false, IsolationReady: false}
	}}.Check(context.Background())
	if r.Outcome != ActionRequired || len(r.Actions) == 0 {
		t.Fatalf("got %#v", r)
	}
	for _, a := range r.Actions {
		if strings.Contains(a.Detail, "/Users/") {
			t.Fatal("path leaked")
		}
	}
}

func TestDoctorTreatsUnknownAndFailedStatesAsActionRequired(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, HostReady: true, HermesReady: true, SupervisorReady: true, BootReady: true, CredentialsReady: true, IsolationReady: true, HostState: StateUnknown, HermesState: StateFailed, SupervisorState: StateRunning, BootState: StateRunning}
	}}.Check(context.Background())
	if r.Outcome != ActionRequired {
		t.Fatalf("got %#v", r)
	}
}

func TestDoctorIncludesSafeVersionsAndStates(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, HostReady: true, HermesReady: true, SupervisorReady: true, BootReady: true, CredentialsReady: true, IsolationReady: true, HermesVersion: "1.2.3", LumenVersion: "2.3.4", HostState: StateRunning, HermesState: StateRunning, SupervisorState: StateRunning, BootState: StateRunning}
	}}.Check(context.Background())
	if r.Outcome != Ready || r.HermesVersion != "1.2.3" || r.LumenVersion != "2.3.4" || r.States["host"] != StateRunning {
		t.Fatalf("got %#v", r)
	}
}
