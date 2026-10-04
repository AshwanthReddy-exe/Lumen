package conversation

import (
	"errors"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

var ErrPersonaUnavailable = errors.New("conversation persona unavailable")

// CompilePersona returns a value copy of the active immutable persona bound to
// a conversation. Callers cannot alter the canonical record through it.
func CompilePersona(state space.State, conversationID string) (space.Persona, error) {
	conversation, ok := state.Conversations[conversationID]
	if !ok {
		return space.Persona{}, ErrPersonaUnavailable
	}
	persona, ok := state.Personas[conversation.PersonaID]
	if !ok || persona.Status != space.PersonaActive || persona.Digest != space.DigestText(persona.Instructions) {
		return space.Persona{}, ErrPersonaUnavailable
	}
	return persona, nil
}
