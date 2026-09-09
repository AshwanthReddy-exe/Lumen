package space

func paired(s State, id string) bool { n, ok := s.Nodes[id]; return ok && n.Status != "revoked" }
func owner(s State, id string) bool  { return id == s.OwnerID && paired(s, id) }
func key(c Command) string           { return c.TargetNodeID + "|" + c.CapabilityID + "|" + c.Action }
func advertised(s State, k string) bool {
	for _, a := range s.Advertisements {
		if a.NodeID+"|"+a.CapabilityID+"|"+a.Action == k {
			return true
		}
	}
	return false
}
func valid(v string) bool { return v != "" }
func grant(s State, k string) Grant {
	if s.Grants == nil {
		return GrantDeny
	}
	g := s.Grants[k]
	if g != GrantDeny && g != GrantAsk && g != GrantAllow {
		return GrantDeny
	}
	return g
}
