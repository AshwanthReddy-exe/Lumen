package space

func audit(s State, event AuditEventType, id string) State {
	s.Audit = append(s.Audit, AuditEvent{Event: event, RequestID: id})
	return s
}
