package contract

import (
	space "github.com/AshwanthReddy-exe/Lumen/internal/space"
	"testing"
)

func TestThreeNodeAuthorityScenario(t *testing.T) {
	s := space.State{SchemaVersion: 1}
	apply := func(c space.Command) space.Transition { tr := space.Apply(s, c); s = tr.State; return tr }
	must := func(c space.Command, out space.Outcome) {
		tr := apply(c)
		if tr.Rejection != "" || tr.Receipt.Outcome != out {
			t.Fatalf("%s: %#v", c.Type, tr)
		}
	}
	must(space.Command{Type: space.CommandCreateSpace, SpaceID: "s", OwnerID: "owner", HostID: "host", RequestID: "create"}, space.OutcomeApplied)
	must(space.Command{Type: space.CommandPairNode, ActorID: "owner", NodeID: "mac", RequestID: "pair"}, space.OutcomeApplied)
	must(space.Command{Type: space.CommandAdvertiseCapability, ActorID: "mac", NodeID: "mac", CapabilityID: "agent.run/execute", Action: "run", RequestID: "advertise"}, space.OutcomeApplied)
	must(space.Command{Type: space.CommandSetGrant, ActorID: "owner", NodeID: "mac", CapabilityID: "agent.run/execute", Action: "run", Grant: space.GrantAsk, RequestID: "grant"}, space.OutcomeApplied)
	must(space.Command{Type: space.CommandSubmit, SpaceID: "s", HostID: "host", Epoch: 1, OriginNodeID: "owner", TargetNodeID: "mac", CapabilityID: "agent.run/execute", Action: "run", ActionFingerprint: "digest", TaskID: "task", RequestID: "submit"}, space.OutcomeAwaitingPermission)
	must(space.Command{Type: space.CommandApprove, SpaceID: "s", HostID: "host", Epoch: 1, ActorID: "owner", TargetNodeID: "mac", ActionFingerprint: "digest", TaskID: "task", ApprovalID: "approval", ApprovedAt: 1, ExpiresAt: 2, RequestID: "approve"}, space.OutcomeQueued)
	must(space.Command{Type: space.CommandComplete, SpaceID: "s", HostID: "host", Epoch: 1, ActorID: "mac", TaskID: "task", Outcome: space.OutcomeCompleted, RequestID: "complete"}, space.OutcomeCompleted)
	if tr := apply(space.Command{Type: space.CommandComplete, SpaceID: "s", HostID: "host", Epoch: 1, ActorID: "mac", TaskID: "task", Outcome: space.OutcomeCompleted, RequestID: "complete"}); tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeCompleted {
		t.Fatalf("duplicate completion replay: %#v", tr)
	}
	if tr := apply(space.Command{Type: space.CommandComplete, SpaceID: "s", HostID: "host", Epoch: 1, ActorID: "mac", TaskID: "task", Outcome: space.OutcomeFailed, RequestID: "complete"}); tr.Rejection != "idempotency_key_reused" {
		t.Fatalf("changed-content collision: %#v", tr)
	}
}
