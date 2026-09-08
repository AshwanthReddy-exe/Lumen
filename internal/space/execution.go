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
	if t.HostEpoch != s.Epoch || t.Status != OutcomeQueued {
		return reject(s, c, "invalid_task_state")
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
	t.Status = OutcomeDispatched
	s.Tasks[c.TaskID] = t
	return accepted(s, c, OutcomeDispatched, c.TaskID)
}

func reconcileHostRun(s State, c Command) Transition {
	if reason := executionContext(s, c); reason != "" {
		return reject(s, c, reason)
	}
	r, t, reason := runMapping(s, c)
	if reason != "" {
		return reject(s, c, reason)
	}
	if c.ObservedAt <= 0 {
		return reject(s, c, "invalid_timestamp")
	}
	if c.Evidence != EvidenceRunning && c.Evidence != EvidenceCompleted && c.Evidence != EvidenceFailed && c.Evidence != EvidenceCancelled && c.Evidence != EvidenceUnavailable {
		return reject(s, c, "invalid_evidence")
	}
	if t.Status == OutcomeCompleted || t.Status == OutcomeFailed || t.Status == OutcomeCancelled {
		return reject(s, c, "terminal_task_state")
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
