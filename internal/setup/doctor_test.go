package setup

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDoctorDistinguishesDegradedAndRedactsDetails(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{HostReady: true, HermesReady: false, HermesState: "unavailable", SupervisorReady: true, BootReady: true, CredentialsReady: true, IsolationReady: true, ArtifactsReady: true, ArtifactState: "installed"}
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
		return DoctorEvidence{Stage: Validated, HostReady: true, HermesReady: true, SupervisorReady: true, BootReady: true, CredentialsReady: true, IsolationReady: true, ArtifactsReady: true, ArtifactState: "installed", HermesVersion: "1.2.3", LumenVersion: "2.3.4", HostState: StateRunning, HermesState: StateRunning, SupervisorState: StateRunning, BootState: StateRunning}
	}}.Check(context.Background())
	if r.Outcome != Ready || r.HermesVersion != "1.2.3" || r.LumenVersion != "2.3.4" || r.States["host"] != StateRunning {
		t.Fatalf("got %#v", r)
	}
}

func TestDoctorRequiresRunningSupervisorAndBoot(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, HostReady: true, HostState: StateRunning, HermesReady: true, HermesState: StateRunning, SupervisorReady: true, SupervisorState: StateStopped, BootReady: true, BootState: StateStopped, CredentialsReady: true, IsolationReady: true}
	}}.Check(context.Background())
	if r.Outcome != ActionRequired {
		t.Fatalf("stopped supervisor/boot reported %#v", r)
	}
}

func TestDoctorRedactsErrorDetails(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{HostError: errors.New("open /Users/alice/private/token: secret"), HermesError: errors.New("https://user:pass@example.invalid")}
	}}.Check(context.Background())
	b, err := json.Marshal(r)
	if err != nil || strings.Contains(string(b), "alice") || strings.Contains(string(b), "secret") || strings.Contains(string(b), "user:pass") {
		t.Fatalf("leaked diagnostic: %s (%v)", b, err)
	}
}

func TestDoctorHermesProbeErrorIsDegradedNotReady(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, HostReady: true, HostState: StateRunning, HermesReady: true, HermesState: StateRunning, HermesError: errors.New("provider secret"), SupervisorReady: true, SupervisorState: StateRunning, BootReady: true, BootState: StateRunning, CredentialsReady: true, IsolationReady: true, ArtifactsReady: true, ArtifactState: "installed"}
	}}.Check(context.Background())
	if r.Outcome != Degraded {
		t.Fatalf("got %#v", r)
	}
}

func TestDoctorMissingArtifactEvidenceIsActionRequired(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, HostReady: true, HostState: StateRunning, HermesReady: false, HermesState: "unavailable", SupervisorReady: true, SupervisorState: StateRunning, BootReady: true, BootState: StateRunning, CredentialsReady: true, IsolationReady: true}
	}}.Check(context.Background())
	if r.Outcome != ActionRequired {
		t.Fatalf("missing artifacts reported %#v", r)
	}
}

func TestDoctorRejectsMalformedPublicMetadata(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, LumenVersion: "/Users/alice/token", HermesVersion: "not-version", HostReady: true, HostState: StateRunning, HermesReady: true, HermesState: StateRunning, SupervisorReady: true, SupervisorState: StateRunning, BootReady: true, BootState: StateRunning, CredentialsReady: true, IsolationReady: true, ArtifactsReady: true, ArtifactState: "installed"}
	}}.Check(context.Background())
	b, _ := json.Marshal(r)
	if r.Outcome != ActionRequired || strings.Contains(string(b), "alice") || strings.Contains(string(b), "not-version") {
		t.Fatalf("metadata leaked: %#v %s", r, b)
	}
}

func TestDoctorAcceptsInstalledArtifactRunningState(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, HostReady: true, HostState: StateRunning, HermesReady: false, HermesState: "unavailable", SupervisorReady: true, SupervisorState: StateRunning, BootReady: true, BootState: StateRunning, CredentialsReady: true, IsolationReady: true, ArtifactsReady: true, ArtifactState: StateRunning}
	}}.Check(context.Background())
	if r.Outcome != Degraded {
		t.Fatalf("installed artifacts changed outcome: %#v", r)
	}
}

func TestDoctorPublishesNormalizedDeploymentStateOnly(t *testing.T) {
	r := Doctor{Observe: func(context.Context) DoctorEvidence {
		return DoctorEvidence{Stage: Validated, HostReady: true, HostState: StateRunning, HermesReady: true, HermesState: StateRunning, SupervisorReady: true, SupervisorState: StateRunning, BootReady: true, BootState: StateRunning, CredentialsReady: true, IsolationReady: true, ArtifactsReady: true, ArtifactState: StateRunning}
	}}.Check(context.Background())
	if r.Outcome != Ready {
		t.Fatalf("got %#v", r)
	}
	if _, ok := r.States["browser_extension_control"]; ok {
		t.Fatal("Hermes-discovered feature became public Lumen state")
	}
}
