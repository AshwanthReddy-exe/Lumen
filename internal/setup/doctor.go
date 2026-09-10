package setup

import (
	"context"
	"regexp"
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

var publicVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func (d Doctor) Check(ctx context.Context) Report {
	if d.Observe == nil {
		return Report{Outcome: ActionRequired, Actions: []Action{{Code: "doctor_unavailable"}}}
	}
	e := d.Observe(ctx)
	r := Report{States: map[string]string{}}
	if e.Stage == "" {
		r.Stage = Validated
	} else if validStage(e.Stage) {
		r.Stage = e.Stage
	} else {
		r.Stage = Validated
		r.Actions = append(r.Actions, Action{Code: "metadata_invalid"})
		r.Outcome = ActionRequired
	}
	if e.Profile != "" {
		if validProfile(e.Profile) {
			r.Profile = e.Profile
		} else {
			r.Actions = append(r.Actions, Action{Code: "metadata_invalid"})
			r.Outcome = ActionRequired
		}
	}
	if e.Platform != "" {
		if validPlatform(e.Platform) {
			r.Platform = e.Platform
		} else {
			r.Actions = append(r.Actions, Action{Code: "metadata_invalid"})
			r.Outcome = ActionRequired
		}
	}
	if e.LumenVersion != "" {
		if publicVersion.MatchString(e.LumenVersion) {
			r.LumenVersion = e.LumenVersion
		} else {
			r.Actions = append(r.Actions, Action{Code: "metadata_invalid"})
			r.Outcome = ActionRequired
		}
	}
	if e.HermesVersion != "" {
		if publicVersion.MatchString(e.HermesVersion) {
			r.HermesVersion = e.HermesVersion
		} else {
			r.Actions = append(r.Actions, Action{Code: "metadata_invalid"})
			r.Outcome = ActionRequired
		}
	}
	for name, state := range map[string]string{"host": e.HostState, "hermes": e.HermesState, "supervisor": e.SupervisorState, "boot": e.BootState, "artifacts": e.ArtifactState} {
		if state == "" {
			continue
		}
		if !validPublicState(name, state) {
			r.Actions = append(r.Actions, Action{Code: "metadata_invalid"})
			r.Outcome = ActionRequired
			continue
		}
		r.States[name] = state
	}
	if validStage(e.Stage) && e.Stage != "" && e.Stage != Validated {
		r.Actions = append(r.Actions, Action{Code: "setup_incomplete"})
		r.Outcome = ActionRequired
	}
	if !e.ArtifactsReady {
		r.Actions = append(r.Actions, Action{Code: "artifacts_unavailable"})
		r.Outcome = ActionRequired
	}
	if e.HostState != "" && e.HostState != StateRunning {
		r.Actions = append(r.Actions, Action{Code: "host_unready"})
		r.Outcome = ActionRequired
	}
	if e.SupervisorState != "" && e.SupervisorState != StateRunning {
		r.Actions = append(r.Actions, Action{Code: "supervisor_unavailable"})
		r.Outcome = ActionRequired
	}
	if e.BootState != "" && e.BootState != StateRunning {
		r.Actions = append(r.Actions, Action{Code: "boot_not_ready"})
		r.Outcome = ActionRequired
	}
	if !e.CredentialsReady {
		r.Actions = append(r.Actions, Action{Code: "credentials_required"})
		r.Outcome = ActionRequired
	}
	if !e.IsolationReady {
		r.Actions = append(r.Actions, Action{Code: "isolation_required"})
		r.Outcome = ActionRequired
	}
	if !e.SupervisorReady {
		r.Actions = append(r.Actions, Action{Code: "supervisor_unavailable"})
		r.Outcome = ActionRequired
	}
	if !e.BootReady {
		r.Actions = append(r.Actions, Action{Code: "boot_not_ready"})
		r.Outcome = ActionRequired
	}
	if !e.HostReady || e.HostError != nil {
		r.Actions = append(r.Actions, Action{Code: "host_unready"})
		r.Outcome = ActionRequired
	}
	if e.HermesError != nil {
		e.HermesReady = false
		if r.Outcome == "" {
			r.Outcome = Degraded
		}
	}
	if !e.HermesReady {
		if r.Outcome == "" {
			r.Outcome = Degraded
		}
		r.Actions = append(r.Actions, Action{Code: "hermes_unavailable"})
		r.States["hermes"] = "unavailable"
	}
	if r.Outcome == "" {
		r.Outcome = Ready
	}
	if len(r.Actions) > 1 {
		seen := make(map[string]bool, len(r.Actions))
		actions := r.Actions[:0]
		for _, action := range r.Actions {
			if !seen[action.Code] {
				seen[action.Code] = true
				actions = append(actions, action)
			}
		}
		r.Actions = actions
	}
	return r
}

func validPublicState(name, state string) bool {
	switch name {
	case "host", "supervisor", "boot":
		return state == StateRunning || state == StateStopped || state == StateFailed || state == StateUnknown
	case "hermes":
		return state == StateRunning || state == StateStopped || state == StateFailed || state == StateUnknown || state == "unavailable"
	case "artifacts":
		return state == "installed" || state == "missing" || state == StateUnknown || state == StateRunning
	default:
		return false
	}
}

func validPlatform(p Platform) bool {
	return p == PlatformMacOS || p == PlatformLinux || p == PlatformTermux
}
