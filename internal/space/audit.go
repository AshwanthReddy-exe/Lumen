package space

import "strings"

func audit(s State, event AuditEventType, id string, actor string, operation CommandType, outcome string) State {
	s.Audit = append(s.Audit, AuditEvent{Event: event, RequestID: id, ActorID: actor, HostID: s.HostID, Epoch: s.Epoch, Operation: operation, Outcome: outcome})
	return s
}

func RuntimeCertificationInvalidationID(cert RuntimeCertification) string {
	return "cert-invalidate:" + DigestText(strings.Join([]string{cert.ID, cert.RuntimeIdentity, cert.ProfileDigest}, "\x00"))
}

func RuntimeCertificationInvalidated(state State, cert RuntimeCertification) bool {
	id := RuntimeCertificationInvalidationID(cert)
	for _, event := range state.Audit {
		if event.Event == AuditRuntimeCertificationInvalidated && event.RequestID == id {
			return true
		}
	}
	return false
}
