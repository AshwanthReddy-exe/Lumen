package space

import "testing"

func executionState(status Outcome) State {
	return State{SchemaVersion: 1, SpaceID: "s", OwnerID: "owner", HostID: "host", Epoch: 7,
		Nodes: map[string]Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}},
		Tasks: map[string]Task{"task": {ID: "task", OriginNodeID: "owner", TargetNodeID: "host", HostEpoch: 7, Status: status}}, Audit: []AuditEvent{}}
}

func TestHostRunLifecycle(t *testing.T) {
	s := executionState(OutcomeQueued)
	d := Apply(s, Command{Type: CommandDispatchHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "d", TaskID: "task", RuntimeRunID: "run-1", RuntimeProfileDigest: "sha256:p1", DispatchedAt: 100, ReconcileBy: 200})
	if d.Rejection != "" || d.Receipt.Outcome != OutcomeDispatched {
		t.Fatalf("dispatch: %#v", d)
	}
	if d.State.Tasks["task"].Status != OutcomeDispatched || d.State.HostRuns["task"].RuntimeRunID != "run-1" {
		t.Fatalf("mapping not committed: %#v", d.State)
	}
	r := Apply(d.State, Command{Type: CommandReconcileHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "r", TaskID: "task", RuntimeRunID: "run-1", RuntimeProfileDigest: "sha256:p1", Evidence: EvidenceRunning, ObservedAt: 110})
	if r.Rejection != "" || r.State.Tasks["task"].Status != OutcomeRunning {
		t.Fatalf("running: %#v", r)
	}
	c := Apply(r.State, Command{Type: CommandRequestHostRunCancellation, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "owner", RequestID: "c", TaskID: "task", RuntimeRunID: "run-1", RuntimeProfileDigest: "sha256:p1", ObservedAt: 120})
	if c.Rejection != "" || c.State.Tasks["task"].Status != OutcomeCancelling {
		t.Fatalf("cancel: %#v", c)
	}
	term := Apply(c.State, Command{Type: CommandReconcileHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "x", TaskID: "task", RuntimeRunID: "run-1", RuntimeProfileDigest: "sha256:p1", Evidence: EvidenceCancelled, ObservedAt: 130})
	if term.Rejection != "" || term.State.Tasks["task"].Status != OutcomeCancelled {
		t.Fatalf("terminal: %#v", term)
	}
}

func TestHostRunRejectsInvalidAuthorityMappingAndCancellationOwner(t *testing.T) {
	cases := []struct {
		name, actor, host string
		epoch             int
		run, digest       string
		dispatched, by    int64
		want              string
	}{
		{"wrong host", "other", "host", 7, "r", "p", 1, 2, "unauthorized_actor"},
		{"stale epoch", "host", "host", 6, "r", "p", 1, 2, "stale_host_epoch"},
		{"bad deadline", "host", "host", 7, "r", "p", 2, 2, "invalid_deadline"},
		{"empty mapping", "host", "host", 7, "", "", 1, 2, "invalid_identifier"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := Apply(executionState(OutcomeQueued), Command{Type: CommandDispatchHostRun, SpaceID: "s", HostID: tc.host, Epoch: tc.epoch, ActorID: tc.actor, RequestID: "d", TaskID: "task", RuntimeRunID: tc.run, RuntimeProfileDigest: tc.digest, DispatchedAt: tc.dispatched, ReconcileBy: tc.by})
			if tr.Rejection != tc.want {
				t.Fatalf("got %q want %q", tr.Rejection, tc.want)
			}
		})
	}
	d := Apply(executionState(OutcomeQueued), Command{Type: CommandDispatchHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "d", TaskID: "task", RuntimeRunID: "r", RuntimeProfileDigest: "p", DispatchedAt: 1, ReconcileBy: 2})
	c := Apply(d.State, Command{Type: CommandRequestHostRunCancellation, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "c", TaskID: "task", RuntimeRunID: "r", RuntimeProfileDigest: "p", ObservedAt: 1})
	if c.Rejection != "unauthorized_actor" {
		t.Fatalf("non-owner cancellation: %#v", c)
	}
}

func TestHostRunTimeoutRestartUnknownAndEvidenceBoundResolution(t *testing.T) {
	d := Apply(executionState(OutcomeQueued), Command{Type: CommandDispatchHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "d", TaskID: "task", RuntimeRunID: "r", RuntimeProfileDigest: "p", DispatchedAt: 1, ReconcileBy: 10})
	u := Apply(d.State, Command{Type: CommandReconcileHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "u", TaskID: "task", RuntimeRunID: "r", RuntimeProfileDigest: "p", Evidence: EvidenceUnavailable, ObservedAt: 10})
	if u.Rejection != "" || u.State.Tasks["task"].Status != OutcomeUnknown {
		t.Fatalf("timeout: %#v", u)
	}
	bad := Apply(u.State, Command{Type: CommandReconcileHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "bad", TaskID: "task", RuntimeRunID: "wrong", RuntimeProfileDigest: "p", Evidence: EvidenceCompleted, ObservedAt: 11})
	if bad.Rejection != "run_mapping_mismatch" {
		t.Fatalf("unbound evidence: %#v", bad)
	}
	good := Apply(u.State, Command{Type: CommandReconcileHostRun, SpaceID: "s", HostID: "host", Epoch: 7, ActorID: "host", RequestID: "good", TaskID: "task", RuntimeRunID: "r", RuntimeProfileDigest: "p", Evidence: EvidenceCompleted, ObservedAt: 11})
	if good.Rejection != "" || good.State.Tasks["task"].Status != OutcomeCompleted {
		t.Fatalf("bound resolution: %#v", good)
	}
}
