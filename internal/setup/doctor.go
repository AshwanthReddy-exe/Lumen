package setup

import (
	"context"
	"strings"
)

type DoctorEvidence struct {
	Stage            Stage
	HostReady        bool
	HermesReady      bool
	HermesVersion    string
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
	r := Report{Stage: e.Stage}
	if r.Stage == "" {
		r.Stage = Validated
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
	for i := range r.Actions {
		r.Actions[i].Detail = redact(r.Actions[i].Detail)
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
