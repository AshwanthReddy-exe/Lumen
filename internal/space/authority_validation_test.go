package space

import "testing"

func validState() State {
	return State{SchemaVersion: 1, SpaceID: "s", OwnerID: "owner", HostID: "host", Epoch: 1, Nodes: map[string]Node{"owner": {"owner", "paired"}, "host": {"host", "paired"}, "mac": {"mac", "paired"}}, Advertisements: []CapabilityKey{{"mac", "cap", "run"}}, Grants: map[string]Grant{"mac|cap|run": GrantAllow}, Tasks: map[string]Task{}}
}

func TestAuthorityRequiresHostEpochAndCompletionActor(t *testing.T) {
	s := validState()
	tr := Apply(s, Command{Type: CommandSubmit, SpaceID: "s", OriginNodeID: "owner", TargetNodeID: "mac", CapabilityID: "cap", Action: "run", ActionFingerprint: "d", TaskID: "t", RequestID: "x"})
	if tr.Rejection != "unauthorized_actor" {
		t.Fatalf("missing host: %#v", tr)
	}
	s = validState()
	s.Tasks["t"] = Task{ID: "t", TargetNodeID: "mac", Status: OutcomeQueued}
	tr = Apply(s, Command{Type: CommandComplete, SpaceID: "s", HostID: "host", Epoch: 1, TaskID: "t", Outcome: OutcomeCompleted, RequestID: "x"})
	if tr.Rejection != "unauthorized_actor" {
		t.Fatalf("missing actor: %#v", tr)
	}
}

func TestPolicyCommandsRequireCurrentHostContext(t *testing.T) {
	for _, c := range []Command{
		{Type: CommandPairNode, SpaceID: "s", HostID: "", Epoch: 1, ActorID: "owner", NodeID: "x", RequestID: "p"},
		{Type: CommandAdvertiseCapability, SpaceID: "s", HostID: "other", Epoch: 1, ActorID: "mac", NodeID: "mac", CapabilityID: "cap", Action: "run", RequestID: "a"},
		{Type: CommandSetGrant, SpaceID: "wrong", HostID: "host", Epoch: 0, ActorID: "owner", NodeID: "mac", CapabilityID: "cap", Action: "run", Grant: GrantAllow, RequestID: "g"},
	} {
		if tr := Apply(validState(), c); tr.Rejection != "stale_host_epoch" {
			t.Fatalf("context rejection: %#v", tr)
		}
	}
}

func TestUnknownGrantDefaultsDeny(t *testing.T) {
	s := validState()
	s.Grants["mac|cap|run"] = "allow_all"
	tr := Apply(s, Command{Type: CommandSubmit, SpaceID: "s", HostID: "host", Epoch: 1, OriginNodeID: "owner", TargetNodeID: "mac", CapabilityID: "cap", Action: "run", ActionFingerprint: "d", TaskID: "t", RequestID: "x"})
	if tr.Rejection != "grant_denied" {
		t.Fatalf("unknown grant: %#v", tr)
	}
}

func TestApplyDoesNotMutateInput(t *testing.T) {
	s := validState()
	before := len(s.Audit)
	_ = Apply(s, Command{Type: CommandSubmit, SpaceID: "s", HostID: "host", Epoch: 1, OriginNodeID: "owner", TargetNodeID: "mac", CapabilityID: "cap", Action: "run", ActionFingerprint: "d", TaskID: "t", RequestID: "x"})
	if len(s.Audit) != before || len(s.Tasks) != 0 || len(s.Commands) != 0 {
		t.Fatalf("input mutated: %#v", s)
	}
}

func TestRejectedReplayIsStable(t *testing.T) {
	s := validState()
	c := Command{Type: CommandSubmit, SpaceID: "s", HostID: "host", Epoch: 1, OriginNodeID: "owner", TargetNodeID: "mac", CapabilityID: "missing", Action: "run", ActionFingerprint: "d", TaskID: "t", RequestID: "x"}
	first := Apply(s, c)
	second := Apply(first.State, c)
	if first.Rejection != "capability_not_advertised" || second.Rejection != first.Rejection {
		t.Fatalf("replay: %#v %#v", first, second)
	}
}

func TestApprovalRequiresIdentifierAndPositiveTimes(t *testing.T) {
	s := validState()
	s.Grants["mac|cap|run"] = GrantAsk
	s.Tasks["t"] = Task{ID: "t", TargetNodeID: "mac", CapabilityID: "cap", Action: "run", ActionFingerprint: "d", Status: OutcomeAwaitingPermission}
	tr := Apply(s, Command{Type: CommandApprove, SpaceID: "s", HostID: "host", Epoch: 1, ActorID: "owner", TargetNodeID: "mac", ActionFingerprint: "d", TaskID: "t", ApprovedAt: 0, ExpiresAt: 1, RequestID: "x"})
	if tr.Rejection != "approval_expired" {
		t.Fatalf("invalid approval: %#v", tr)
	}
}

func TestApprovalObservedAtMustBeWithinWindow(t *testing.T) {
	s := validState()
	s.Tasks["t"] = Task{ID: "t", TargetNodeID: "mac", ActionFingerprint: "d", Status: OutcomeAwaitingPermission}
	base := Command{Type: CommandApprove, SpaceID: "s", HostID: "host", Epoch: 1, ActorID: "owner", TargetNodeID: "mac", ActionFingerprint: "d", TaskID: "t", ApprovalID: "a", ApprovedAt: 100, ExpiresAt: 120}
	for name, command := range map[string]Command{
		"missing observation": base,
		"before approval":     func() Command { c := base; c.ObservedAt = 99; return c }(),
		"at expiry":           func() Command { c := base; c.ObservedAt = 120; return c }(),
		"future approval":     func() Command { c := base; c.ApprovedAt = 121; c.ExpiresAt = 140; c.ObservedAt = 120; return c }(),
	} {
		command.RequestID = "reject-" + name
		if tr := Apply(s, command); tr.Rejection != "approval_expired" {
			t.Fatalf("%s: %#v", name, tr)
		}
	}
	valid := base
	valid.RequestID = "valid"
	valid.ObservedAt = 100
	if tr := Apply(s, valid); tr.Rejection != "" {
		t.Fatalf("valid approval rejected: %#v", tr)
	}
}
