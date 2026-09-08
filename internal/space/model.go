package space

import "fmt"

type Grant string

const (
	GrantDeny  Grant = "deny"
	GrantAsk   Grant = "ask"
	GrantAllow Grant = "allow"
)

type IdentityKind string

const (
	IdentityOwner IdentityKind = "owner"
	IdentityHost  IdentityKind = "host"
)

type State struct {
	SchemaVersion int          `json:"schemaVersion"`
	SpaceID       string       `json:"spaceId,omitempty"`
	OwnerID       string       `json:"ownerId,omitempty"`
	HostID        string       `json:"hostId,omitempty"`
	Epoch         int          `json:"epoch,omitempty"`
	Identities    []Identity   `json:"identities,omitempty"`
	Capabilities  []Capability `json:"capabilities,omitempty"`
	Audit         []AuditEvent `json:"audit"`
}
type Identity struct {
	ID   string       `json:"id"`
	Kind IdentityKind `json:"kind"`
}
type Capability struct {
	ID    string `json:"id"`
	Grant Grant  `json:"grant"`
}
type AuditEvent struct {
	Event     string `json:"event"`
	RequestID string `json:"requestId"`
}
type Command struct {
	Type      string `json:"type"`
	SpaceID   string `json:"spaceId"`
	OwnerID   string `json:"ownerId"`
	HostID    string `json:"hostId"`
	RequestID string `json:"requestId"`
}
type Transition struct {
	State     State
	Receipt   Receipt
	Rejection string
}
type Receipt struct {
	RequestID string `json:"requestId,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
}
type FixtureSuite struct {
	SchemaVersion int           `json:"schemaVersion"`
	Cases         []FixtureCase `json:"cases"`
}
type FixtureCase struct {
	Name     string          `json:"name"`
	Initial  State           `json:"initial"`
	Commands []Command       `json:"commands"`
	Expected FixtureExpected `json:"expected"`
}
type FixtureExpected struct {
	State State        `json:"state"`
	Audit []AuditEvent `json:"audit"`
}

func Apply(state State, command Command) Transition {
	if command.Type != "create_space" {
		return Transition{State: state, Rejection: fmt.Sprintf("unsupported command: %s", command.Type)}
	}
	if state.SpaceID != "" {
		return Transition{State: state, Rejection: "space already exists"}
	}
	if command.SpaceID == "" || command.OwnerID == "" || command.HostID == "" || command.OwnerID == command.HostID {
		return Transition{State: state, Rejection: "invalid space identities"}
	}
	state.SpaceID, state.OwnerID, state.HostID, state.Epoch = command.SpaceID, command.OwnerID, command.HostID, 1
	state.Identities = []Identity{{command.OwnerID, IdentityOwner}, {command.HostID, IdentityHost}}
	state.Capabilities = []Capability{{"agent.run/execute", GrantAsk}}
	state.Audit = append(state.Audit, AuditEvent{"space.created", command.RequestID})
	return Transition{State: state, Receipt: Receipt{command.RequestID, "applied"}}
}
