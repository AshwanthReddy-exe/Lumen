package space

func executionContext(s State, c Command) string {
	if c.SpaceID != s.SpaceID {
		return "invalid_space"
	}
	if c.HostID != s.HostID || c.Epoch == 0 || c.Epoch != s.Epoch {
		return "stale_host_epoch"
	}
	if c.ActorID != s.HostID {
		return "unauthorized_actor"
	}
	return ""
}

func runMapping(s State, c Command) (HostRun, Task, string) {
	t, ok := s.Tasks[c.TaskID]
	r, rok := s.HostRuns[c.TaskID]
	if !ok {
		return HostRun{}, Task{}, "task_unknown"
	}
	if !rok || r.TaskID != c.TaskID || r.RuntimeRunID != c.RuntimeRunID || r.RuntimeProfileDigest != c.RuntimeProfileDigest || r.HostEpoch != s.Epoch {
		return HostRun{}, t, "run_mapping_mismatch"
	}
	return r, t, ""
}

func dispatchHostRun(s State, c Command) Transition {
	if reason := executionContext(s, c); reason != "" {
		return reject(s, c, reason)
	}
	if c.TaskID == "" || c.RuntimeRunID == "" || c.RuntimeProfileDigest == "" {
		return reject(s, c, "invalid_identifier")
	}
	if c.DispatchedAt <= 0 || c.ReconcileBy <= c.DispatchedAt {
		return reject(s, c, "invalid_deadline")
	}
	t, ok := s.Tasks[c.TaskID]
	if !ok {
		return reject(s, c, "task_unknown")
	}
	if t.HostEpoch != s.Epoch || (t.Status != OutcomeQueued && t.Status != OutcomeCreating) {
		return reject(s, c, "invalid_task_state")
	}
	if intent, exists := s.HostCreates[c.TaskID]; exists {
		if intent.IdempotencyKey != c.RuntimeIdempotencyKey || intent.RuntimeProfileDigest != c.RuntimeProfileDigest || intent.HostEpoch != c.Epoch || intent.ReconcileBy != c.ReconcileBy {
			return reject(s, c, "create_intent_mismatch")
		}
	} else if c.RuntimeIdempotencyKey != "" {
		return reject(s, c, "create_intent_missing")
	}
	if s.HostRuns == nil {
		s.HostRuns = map[string]HostRun{}
	}
	if _, exists := s.HostRuns[c.TaskID]; exists {
		return reject(s, c, "run_mapping_exists")
	}
	for _, r := range s.HostRuns {
		if r.RuntimeRunID == c.RuntimeRunID {
			return reject(s, c, "runtime_run_id_collision")
		}
	}
	s.HostRuns[c.TaskID] = HostRun{TaskID: c.TaskID, RuntimeRunID: c.RuntimeRunID, RuntimeProfileDigest: c.RuntimeProfileDigest, HostEpoch: s.Epoch, DispatchedAt: c.DispatchedAt, ReconcileBy: c.ReconcileBy}
	delete(s.HostCreates, c.TaskID)
	t.Status = OutcomeDispatched
	s.Tasks[c.TaskID] = t
	return accepted(s, c, OutcomeDispatched, c.TaskID)
}

func createHostRun(s State, c Command) Transition {
	if reason := executionContext(s, c); reason != "" {
		return reject(s, c, reason)
	}
	if c.TaskID == "" || c.RuntimeIdempotencyKey == "" || c.RuntimeProfileDigest == "" {
		return reject(s, c, "invalid_identifier")
	}
	if c.DispatchedAt <= 0 || c.ReconcileBy <= c.DispatchedAt {
		return reject(s, c, "invalid_deadline")
	}
	t, ok := s.Tasks[c.TaskID]
	if !ok {
		return reject(s, c, "task_unknown")
	}
	if t.HostEpoch != s.Epoch || t.Status != OutcomeQueued {
		return reject(s, c, "invalid_task_state")
	}
	if s.HostCreates == nil {
		s.HostCreates = map[string]HostCreate{}
	}
	if _, exists := s.HostCreates[c.TaskID]; exists {
		return reject(s, c, "create_intent_exists")
	}
	for _, intent := range s.HostCreates {
		if intent.IdempotencyKey == c.RuntimeIdempotencyKey {
			return reject(s, c, "runtime_idempotency_collision")
		}
	}
	s.HostCreates[c.TaskID] = HostCreate{TaskID: c.TaskID, IdempotencyKey: c.RuntimeIdempotencyKey, RuntimeProfileDigest: c.RuntimeProfileDigest, HostEpoch: s.Epoch, CreatedAt: c.DispatchedAt, ReconcileBy: c.ReconcileBy}
	t.Status = OutcomeCreating
	s.Tasks[c.TaskID] = t
	return accepted(s, c, OutcomeCreating, c.TaskID)
}

func requestRuntimeApproval(s State, c Command) Transition {
	if reason := executionContext(s, c); reason != "" {
		return reject(s, c, reason)
	}
	run, task, reason := runMapping(s, c)
	if reason != "" {
		return reject(s, c, reason)
	}
	if task.Status != OutcomeRunning && task.Status != OutcomeDispatched {
		return reject(s, c, "invalid_task_state")
	}
	if c.RuntimeApprovalID == "" || c.TargetNodeID == "" || c.ActionFingerprint == "" || c.ExpiresAt <= c.ObservedAt {
		return reject(s, c, "invalid_runtime_approval")
	}
	if s.RuntimeApprovals == nil {
		s.RuntimeApprovals = map[string]RuntimeApproval{}
	}
	if _, exists := s.RuntimeApprovals[c.RuntimeApprovalID]; exists {
		return reject(s, c, "runtime_approval_exists")
	}
	s.RuntimeApprovals[c.RuntimeApprovalID] = RuntimeApproval{ID: c.RuntimeApprovalID, TaskID: c.TaskID, RuntimeRunID: run.RuntimeRunID, ActorNodeID: c.ActorID, TargetNodeID: c.TargetNodeID, ActionFingerprint: c.ActionFingerprint, ExpiresAt: c.ExpiresAt, DeliveryState: "pending"}
	task.Status = OutcomeAwaitingPermission
	s.Tasks[c.TaskID] = task
	return accepted(s, c, OutcomeAwaitingPermission, c.TaskID)
}

func resolveRuntimeApproval(s State, c Command) Transition {
	if c.SpaceID != s.SpaceID || c.HostID != s.HostID || c.Epoch == 0 || c.Epoch != s.Epoch {
		return reject(s, c, "stale_host_epoch")
	}
	if !owner(s, c.ActorID) {
		return reject(s, c, "unauthorized_actor")
	}
	approval, ok := s.RuntimeApprovals[c.RuntimeApprovalID]
	run, runOK := s.HostRuns[c.TaskID]
	if !ok || !runOK || run.RuntimeRunID != c.RuntimeRunID || run.RuntimeProfileDigest != c.RuntimeProfileDigest || approval.TaskID != c.TaskID || approval.RuntimeRunID != c.RuntimeRunID || approval.TargetNodeID != c.TargetNodeID || approval.ActionFingerprint != c.ActionFingerprint {
		return reject(s, c, "runtime_approval_mismatch")
	}
	if c.ObservedAt <= 0 || c.ObservedAt >= approval.ExpiresAt {
		return reject(s, c, "approval_expired")
	}
	if c.Decision != "once" && c.Decision != "deny" {
		return reject(s, c, "invalid_approval")
	}
	task, ok := s.Tasks[c.TaskID]
	if !ok || task.Status != OutcomeAwaitingPermission {
		return reject(s, c, "approval_not_required")
	}
	if approval.Decision != "" && approval.Decision != c.Decision {
		return reject(s, c, "runtime_approval_decision_mismatch")
	}
	if approval.DeliveryState == "delivered" {
		return reject(s, c, "runtime_approval_already_delivered")
	}
	approval.Decision = c.Decision
	approval.DeliveryState = "pending"
	s.RuntimeApprovals[c.RuntimeApprovalID] = approval
	return accepted(s, c, OutcomeAwaitingPermission, c.TaskID)
}

func recordRuntimeApproval(s State, c Command) Transition {
	if reason := executionContext(s, c); reason != "" {
		return reject(s, c, reason)
	}
	approval, ok := s.RuntimeApprovals[c.RuntimeApprovalID]
	run, runOK := s.HostRuns[c.TaskID]
	if !ok || !runOK || approval.TaskID != c.TaskID || approval.RuntimeRunID != c.RuntimeRunID || run.RuntimeProfileDigest != c.RuntimeProfileDigest || approval.Decision != c.Decision || approval.TargetNodeID != c.TargetNodeID || approval.ActionFingerprint != c.ActionFingerprint {
		return reject(s, c, "runtime_approval_mismatch")
	}
	if c.DeliveryState != "delivered" && c.DeliveryState != "uncertain" {
		return reject(s, c, "invalid_runtime_approval_delivery")
	}
	if approval.DeliveryState == "delivered" {
		return reject(s, c, "runtime_approval_already_delivered")
	}
	if c.DeliveryState == "delivered" {
		task, exists := s.Tasks[c.TaskID]
		if !exists || task.Status != OutcomeAwaitingPermission {
			return reject(s, c, "approval_not_required")
		}
		if c.Decision == "deny" {
			task.Status = OutcomeFailed
			task.TerminalReason = "runtime_approval_denied"
		} else {
			task.Status = OutcomeRunning
		}
		s.Tasks[c.TaskID] = task
	}
	approval.DeliveryState = c.DeliveryState
	s.RuntimeApprovals[c.RuntimeApprovalID] = approval
	out := OutcomeAwaitingPermission
	if c.DeliveryState == "delivered" {
		out = s.Tasks[c.TaskID].Status
	}
	return accepted(s, c, out, c.TaskID)
}

func reconcileHostRun(s State, c Command) Transition {
	if reason := executionContext(s, c); reason != "" {
		return reject(s, c, reason)
	}
	r, t, reason := runMapping(s, c)
	if reason != "" {
		return reject(s, c, reason)
	}
	if c.ObservedAt < r.DispatchedAt {
		return reject(s, c, "invalid_timestamp")
	}
	if c.Evidence != EvidenceRunning && c.Evidence != EvidenceCompleted && c.Evidence != EvidenceFailed && c.Evidence != EvidenceCancelled && c.Evidence != EvidenceUnavailable {
		return reject(s, c, "invalid_evidence")
	}
	if t.Status == OutcomeCompleted || t.Status == OutcomeFailed || t.Status == OutcomeCancelled {
		return reject(s, c, "terminal_task_state")
	}
	if t.Status == OutcomeCancelling && c.Evidence == EvidenceRunning {
		return reject(s, c, "cancellation_in_progress")
	}
	if t.Status == OutcomeAwaitingPermission && c.Evidence == EvidenceRunning {
		return reject(s, c, "approval_pending")
	}
	if t.Status == OutcomeAwaitingPermission && c.Evidence != EvidenceRunning {
		pending := false
		for _, approval := range s.RuntimeApprovals {
			if approval.TaskID == c.TaskID && approval.DeliveryState != "delivered" {
				pending = c.ObservedAt < approval.ExpiresAt && !(c.Evidence == EvidenceUnavailable && c.ObservedAt >= r.ReconcileBy)
				break
			}
		}
		if pending {
			return reject(s, c, "approval_pending")
		}
	}
	var out Outcome
	switch c.Evidence {
	case EvidenceRunning:
		out = OutcomeRunning
	case EvidenceCompleted:
		out = OutcomeCompleted
	case EvidenceFailed:
		out = OutcomeFailed
	case EvidenceCancelled:
		out = OutcomeCancelled
	case EvidenceUnavailable:
		if c.ObservedAt < r.ReconcileBy {
			return reject(s, c, "reconciliation_pending")
		}
		out = OutcomeUnknown
	}
	t.Status = out
	s.Tasks[c.TaskID] = t
	return accepted(s, c, out, c.TaskID)
}

func cancelHostRun(s State, c Command) Transition {
	if c.SpaceID != s.SpaceID {
		return reject(s, c, "invalid_space")
	}
	if c.HostID != s.HostID || c.Epoch == 0 || c.Epoch != s.Epoch {
		return reject(s, c, "stale_host_epoch")
	}
	if c.ActorID != s.OwnerID || !owner(s, c.ActorID) {
		return reject(s, c, "unauthorized_actor")
	}
	_, t, reason := runMapping(s, c)
	if reason != "" {
		return reject(s, c, reason)
	}
	if c.ObservedAt <= 0 {
		return reject(s, c, "invalid_timestamp")
	}
	if t.Status != OutcomeDispatched && t.Status != OutcomeRunning {
		return reject(s, c, "invalid_task_state")
	}
	t.Status = OutcomeCancelling
	s.Tasks[c.TaskID] = t
	return accepted(s, c, OutcomeCancelling, c.TaskID)
}
