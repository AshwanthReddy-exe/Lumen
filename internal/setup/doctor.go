package setup

import (
	"context"
	"strings"
)

type DoctorEvidence struct {
	Stage            Stage
	Profile          Profile
	Platform         Platform
	LumenVersion     string
	HostReady        bool
	HostState        string
	HermesReady      bool
	HermesState      string
	HermesVersion    string
	ArtifactsReady   bool
	ArtifactState    string
	SupervisorState  string
	BootState        string
	SupervisorReady  bool
	BootReady        bool
	CredentialsReady bool
	IsolationReady   bool
	HostError        error
	HermesError      error
}

type Doctor struct {
	Observe func(context.Context) DoctorEvidence
}

func (d Doctor) Check(ctx context.Context) Report {
	if d.Observe == nil {
		return Report{Outcome: ActionRequired, Actions: []Action{{Code: "doctor_unavailable"}}}
	}
	e := d.Observe(ctx)
	r := Report{Stage: e.Stage, Profile: e.Profile, Platform: e.Platform, LumenVersion: e.LumenVersion, HermesVersion: e.HermesVersion, States: map[string]string{}}
	r.LumenVersion, r.HermesVersion = e.LumenVersion, e.HermesVersion
	if e.HostState == StateUnknown || e.HostState == StateFailed || e.HermesState == StateUnknown || e.HermesState == StateFailed || e.SupervisorState == StateUnknown || e.BootState == StateUnknown {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "invalid_runtime_state"})
	}
	if r.Stage == "" {
		r.Stage = Validated
	}
	if e.Stage != "" && e.Stage != Validated {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "setup_incomplete"})
	}
	for name, state := range map[string]string{"host": e.HostState, "hermes": e.HermesState, "supervisor": e.SupervisorState, "boot": e.BootState, "artifacts": e.ArtifactState} {
		if state != "" {
			r.States[name] = state
		}
	}
	if !e.ArtifactsReady && e.ArtifactState != "" {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "artifacts_unavailable"})
	}
	if (e.HostState != "" && e.HostState != StateRunning) || e.HermesState == StateUnknown || e.HermesState == StateFailed || (e.SupervisorState != "" && e.SupervisorState != StateRunning) || e.BootState == StateUnknown || e.BootState == StateFailed {
		if e.HostState != "" && e.HostState != StateRunning {
			r.Outcome = ActionRequired
			r.Actions = append(r.Actions, Action{Code: "host_unready"})
		}
		if e.SupervisorState != "" && e.SupervisorState != StateRunning {
			r.Outcome = ActionRequired
			r.Actions = append(r.Actions, Action{Code: "supervisor_unavailable"})
		}
		if e.BootState != "" && e.BootState != StateRunning {
			r.Outcome = ActionRequired
			r.Actions = append(r.Actions, Action{Code: "boot_not_ready"})
		}
	}
	if !e.CredentialsReady {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "credentials_required"})
	}
	if !e.IsolationReady {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "isolation_required"})
	}
	if !e.SupervisorReady {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "supervisor_unavailable"})
	}
	if !e.BootReady {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "boot_not_ready"})
	}
	if !e.HostReady {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "host_unready"})
	}
	if e.HostError != nil {
		r.Outcome = ActionRequired
		r.Actions = append(r.Actions, Action{Code: "host_unready"})
	}
	if !e.HermesReady {
		if r.Outcome == "" {
			r.Outcome = Degraded
		}
		r.Actions = append(r.Actions, Action{Code: "hermes_unavailable"})
	}
	if r.Outcome == "" {
		r.Outcome = Ready
	}
	if !e.HermesReady && e.HermesState != StateUnknown && e.HermesState != StateFailed {
		r.States["hermes"] = "unavailable"
	}
	return r
}

func redact(s string) string {
	for _, part := range strings.Fields(s) {
		if strings.Contains(part, "/") || strings.Contains(part, "\\") {
			s = strings.ReplaceAll(s, part, "[redacted]")
		}
	}
	return s
}
