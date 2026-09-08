package space

import "encoding/json"

func Apply(s State, c Command) Transition {
	id := commandID(c)
	if id == "" {
		return reject(s, c, "invalid_identifier")
	}
	contentBytes, _ := json.Marshal(c)
	content := string(contentBytes)
	if s.Commands != nil {
		if old, ok := s.Commands[id]; ok {
			if old.Content != content {
				return reject(s, c, "idempotency_key_reused")
			}
			return Transition{State: audit(s, AuditCommandReplayed, id), Receipt: receipt(old, id)}
		}
	}
	if s.Commands == nil {
		s.Commands = map[string]RecordedCommand{}
	}
	var tr Transition
	switch c.Type {
	case CommandCreateSpace:
		tr = create(s, c)
	case CommandPairNode:
		tr = pair(s, c)
	case CommandAdvertiseCapability:
		tr = advertise(s, c)
	case CommandSetGrant:
		tr = setGrant(s, c)
	case CommandSubmit:
		tr = submit(s, c)
	case CommandApprove:
		tr = approve(s, c)
	case CommandComplete:
		tr = complete(s, c)
	case CommandRevokeNode:
		tr = revoke(s, c)
	case CommandRecoverAfterRestart:
		tr = recover(s, c)
	default:
		return reject(s, c, "unsupported_command")
	}
	if tr.Rejection != "" {
		tr.State.Commands[id] = RecordedCommand{Type: c.Type, Content: content, Rejection: tr.Rejection}
		return tr
	}
	tr.State.Commands[id] = RecordedCommand{Type: c.Type, Content: content, Receipt: &tr.Receipt}
	return tr
}
func receipt(r RecordedCommand, id string) Receipt {
	if r.Receipt != nil {
		return *r.Receipt
	}
	return Receipt{RequestID: id, Outcome: OutcomeFailed}
}
func reject(s State, c Command, reason string) Transition {
	id := commandID(c)
	return Transition{State: audit(s, AuditCommandRejected, id), Rejection: reason}
}
func accepted(s State, c Command, out Outcome, subject string) Transition {
	id := commandID(c)
	s = audit(s, AuditCommandAccepted, id)
	return Transition{State: s, Receipt: Receipt{RequestID: id, Outcome: out, SubjectID: subject}}
}
func create(s State, c Command) Transition {
	if s.SpaceID != "" {
		return reject(s, c, "space_already_exists")
	}
	if !valid(c.SpaceID) || !valid(c.OwnerID) || !valid(c.HostID) || c.OwnerID == c.HostID {
		return reject(s, c, "invalid_space_identities")
	}
	s.SpaceID = c.SpaceID
	s.OwnerID = c.OwnerID
	s.HostID = c.HostID
	s.Epoch = 1
	s.Identities = []Identity{{c.OwnerID, IdentityOwner}, {c.HostID, IdentityHost}}
	s.Capabilities = []Capability{{"agent.run/execute", GrantAsk}}
	s.Nodes = map[string]Node{c.OwnerID: {c.OwnerID, "paired"}, c.HostID: {c.HostID, "paired"}}
	s = audit(s, AuditSpaceCreated, commandID(c))
	return Transition{State: s, Receipt: Receipt{RequestID: commandID(c), Outcome: OutcomeApplied}}
}
func pair(s State, c Command) Transition {
	if !owner(s, c.ActorID) || !valid(c.NodeID) {
		return reject(s, c, "unauthorized_actor")
	}
	if _, ok := s.Nodes[c.NodeID]; ok {
		return reject(s, c, "node_already_paired")
	}
	if s.Nodes == nil {
		s.Nodes = map[string]Node{}
	}
	s.Nodes[c.NodeID] = Node{c.NodeID, "paired"}
	return accepted(s, c, OutcomeApplied, c.NodeID)
}
func advertise(s State, c Command) Transition {
	if c.ActorID != c.NodeID || !paired(s, c.NodeID) || !valid(c.CapabilityID) || !valid(c.Action) {
		return reject(s, c, "unauthorized_actor")
	}
	s.Advertisements = append(s.Advertisements, CapabilityKey{c.NodeID, c.CapabilityID, c.Action})
	return accepted(s, c, OutcomeApplied, c.NodeID)
}
func setGrant(s State, c Command) Transition {
	k := c.NodeID + "|" + c.CapabilityID + "|" + c.Action
	if !owner(s, c.ActorID) {
		return reject(s, c, "unauthorized_actor")
	}
	if !paired(s, c.NodeID) {
		return reject(s, c, "node_revoked")
	}
	if !advertised(s, k) {
		return reject(s, c, "capability_not_advertised")
	}
	if s.Grants == nil {
		s.Grants = map[string]Grant{}
	}
	s.Grants[k] = c.Grant
	return accepted(s, c, OutcomeApplied, c.NodeID)
}
func submit(s State, c Command) Transition {
	if c.SpaceID != s.SpaceID {
		return reject(s, c, "invalid_space")
	}
	if c.HostID != "" && c.HostID != s.HostID {
		return reject(s, c, "unauthorized_actor")
	}
	if c.Epoch != 0 && c.Epoch != s.Epoch {
		return reject(s, c, "stale_host_epoch")
	}
	if !paired(s, c.OriginNodeID) || !paired(s, c.TargetNodeID) {
		return reject(s, c, "node_unknown")
	}
	k := key(c)
	if !advertised(s, k) {
		return reject(s, c, "capability_not_advertised")
	}
	g := grant(s, k)
	if g == GrantDeny || g == "" {
		return reject(s, c, "grant_denied")
	}
	if s.Tasks == nil {
		s.Tasks = map[string]Task{}
	}
	if _, ok := s.Tasks[c.TaskID]; ok {
		return reject(s, c, "task_already_exists")
	}
	out := OutcomeQueued
	if g == GrantAsk {
		out = OutcomeAwaitingPermission
	}
	s.Tasks[c.TaskID] = Task{ID: c.TaskID, CommandID: commandID(c), OriginNodeID: c.OriginNodeID, TargetNodeID: c.TargetNodeID, CapabilityID: c.CapabilityID, Action: c.Action, ActionFingerprint: c.ActionFingerprint, HostEpoch: s.Epoch, Status: out}
	return accepted(s, c, out, c.TaskID)
}
func approve(s State, c Command) Transition {
	t, ok := s.Tasks[c.TaskID]
	if c.SpaceID != s.SpaceID {
		return reject(s, c, "invalid_space")
	}
	if c.Epoch != 0 && c.Epoch != s.Epoch {
		return reject(s, c, "stale_host_epoch")
	}
	if !owner(s, c.ActorID) {
		return reject(s, c, "unauthorized_actor")
	}
	if !ok {
		return reject(s, c, "task_unknown")
	}
	if t.Status != OutcomeAwaitingPermission {
		return reject(s, c, "approval_not_required")
	}
	if c.TargetNodeID != t.TargetNodeID || c.ActionFingerprint != t.ActionFingerprint {
		return reject(s, c, "approval_mismatch")
	}
	if c.ApprovedAt >= c.ExpiresAt {
		return reject(s, c, "approval_expired")
	}
	if s.Approvals == nil {
		s.Approvals = map[string]Approval{}
	}
	if _, exists := s.Approvals[c.ApprovalID]; exists {
		return reject(s, c, "approval_already_consumed")
	}
	s.Approvals[c.ApprovalID] = Approval{ID: c.ApprovalID, TaskID: c.TaskID, ActorNodeID: c.ActorID, TargetNodeID: c.TargetNodeID, ActionFingerprint: c.ActionFingerprint, ExpiresAt: c.ExpiresAt, ConsumedAt: c.ApprovedAt}
	t.Status = OutcomeQueued
	s.Tasks[c.TaskID] = t
	return accepted(s, c, OutcomeQueued, c.TaskID)
}
func complete(s State, c Command) Transition {
	t, ok := s.Tasks[c.TaskID]
	if c.SpaceID != s.SpaceID {
		return reject(s, c, "invalid_space")
	}
	if c.Epoch != 0 && c.Epoch != s.Epoch {
		return reject(s, c, "stale_host_epoch")
	}
	if !ok {
		return reject(s, c, "task_unknown")
	}
	if c.ActorID != "" && c.ActorID != t.TargetNodeID {
		return reject(s, c, "unauthorized_actor")
	}
	if t.Status != OutcomeQueued {
		return reject(s, c, "invalid_task_state")
	}
	if c.Outcome != OutcomeCompleted && c.Outcome != OutcomeFailed && c.Outcome != OutcomeUnknown {
		return reject(s, c, "invalid_completion_outcome")
	}
	t.Status = c.Outcome
	s.Tasks[c.TaskID] = t
	return accepted(s, c, c.Outcome, c.TaskID)
}
func revoke(s State, c Command) Transition {
	if !owner(s, c.ActorID) {
		return reject(s, c, "unauthorized_actor")
	}
	if c.NodeID == s.OwnerID {
		return reject(s, c, "owner_revocation_forbidden")
	}
	if c.NodeID == s.HostID {
		return reject(s, c, "active_host_revocation_forbidden")
	}
	n, ok := s.Nodes[c.NodeID]
	if !ok {
		return reject(s, c, "node_unknown")
	}
	if n.Status == "revoked" {
		return reject(s, c, "node_revoked")
	}
	n.Status = "revoked"
	s.Nodes[c.NodeID] = n
	for id, t := range s.Tasks {
		if (t.OriginNodeID == c.NodeID || t.TargetNodeID == c.NodeID) && (t.Status == OutcomeQueued || t.Status == OutcomeAwaitingPermission) {
			t.Status = OutcomeFailed
			t.TerminalReason = "node_revoked"
			s.Tasks[id] = t
		}
	}
	return accepted(s, c, OutcomeApplied, c.NodeID)
}
func recover(s State, c Command) Transition {
	if c.SpaceID != s.SpaceID {
		return reject(s, c, "invalid_space")
	}
	if c.Epoch != 0 && c.Epoch != s.Epoch {
		return reject(s, c, "stale_host_epoch")
	}
	if c.ActorID != s.HostID {
		return reject(s, c, "unauthorized_actor")
	}
	for id, t := range s.Tasks {
		if t.Status == OutcomeQueued {
			t.Status = OutcomeUnknown
			t.TerminalReason = "host_restarted"
			s.Tasks[id] = t
		}
	}
	return accepted(s, c, OutcomeApplied, s.SpaceID)
}
