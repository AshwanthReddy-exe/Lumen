package space

func audit(s State, event AuditEventType, id string, actor string, operation CommandType, outcome string) State {
	s.Audit = append(s.Audit, AuditEvent{Event: event, RequestID: id, ActorID: actor, HostID: s.HostID, Epoch: s.Epoch, Operation: operation, Outcome: outcome})
	return s
}
