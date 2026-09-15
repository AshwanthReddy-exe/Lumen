package space

import (
	"encoding/json"
	"errors"
	"fmt"
)

// MigrateV1ToV2 is a pure, idempotent migration of canonical Space state.
// Existing authority records are copied byte-for-byte at the field level; the
// new collections are initialized without importing runtime-owned state.
func MigrateV1ToV2(state State) (State, error) {
	if state.SchemaVersion == StateSchemaVersionV2 {
		return cloneCanonicalState(state), nil
	}
	if state.SchemaVersion != StateSchemaVersionV1 {
		return State{}, fmt.Errorf("unsupported state schema version %d", state.SchemaVersion)
	}
	migrated := cloneCanonicalState(state)
	migrated.SchemaVersion = StateSchemaVersionV2
	migrated.Conversations = map[string]Conversation{}
	migrated.Messages = map[string][]Message{}
	migrated.Personas = map[string]Persona{}
	migrated.Surfaces = map[string]Surface{}
	migrated.ContextRecords = map[string]ContextRecord{}
	migrated.RuntimeSessions = map[string]RuntimeSessionMapping{}
	migrated.RuntimeProfiles = map[string]RuntimeProfile{}
	migrated.RuntimeCertifications = map[string]RuntimeCertification{}
	persona := DefaultPersona()
	profile := DefaultRuntimeProfile()
	migrated.Personas[persona.ID] = persona
	migrated.RuntimeProfiles[profile.ID] = profile
	return migrated, nil
}

func cloneCanonicalState(state State) State {
	b, _ := json.Marshal(state)
	var copy State
	_ = json.Unmarshal(b, &copy)
	if state.SchemaVersion == StateSchemaVersionV2 {
		// New collections are deliberately empty after migration, but State's
		// compatibility tags omit empty maps from JSON. Preserve their
		// initialized-v2 meaning across the pure clone used for idempotence.
		if state.Conversations != nil && copy.Conversations == nil {
			copy.Conversations = map[string]Conversation{}
		}
		if state.Messages != nil && copy.Messages == nil {
			copy.Messages = map[string][]Message{}
		}
		if state.Personas != nil && copy.Personas == nil {
			copy.Personas = map[string]Persona{}
		}
		if state.Surfaces != nil && copy.Surfaces == nil {
			copy.Surfaces = map[string]Surface{}
		}
		if state.ContextRecords != nil && copy.ContextRecords == nil {
			copy.ContextRecords = map[string]ContextRecord{}
		}
		if state.RuntimeSessions != nil && copy.RuntimeSessions == nil {
			copy.RuntimeSessions = map[string]RuntimeSessionMapping{}
		}
		if state.RuntimeProfiles != nil && copy.RuntimeProfiles == nil {
			copy.RuntimeProfiles = map[string]RuntimeProfile{}
		}
		if state.RuntimeCertifications != nil && copy.RuntimeCertifications == nil {
			copy.RuntimeCertifications = map[string]RuntimeCertification{}
		}
	}
	return copy
}

// ValidateState validates schema and the immutable digests/bounds required by
// canonical conversation records. Legacy v1 state remains valid for reads and
// is upgraded only by an explicit migration.
func ValidateState(state State) error {
	if state.SchemaVersion != StateSchemaVersionV1 && state.SchemaVersion != StateSchemaVersionV2 {
		return fmt.Errorf("unsupported state schema version %d", state.SchemaVersion)
	}
	if state.SchemaVersion == StateSchemaVersionV1 {
		return nil
	}
	if state.Personas == nil || state.RuntimeProfiles == nil || state.Conversations == nil || state.Messages == nil || state.ContextRecords == nil || state.RuntimeSessions == nil || state.RuntimeCertifications == nil {
		return errors.New("schema v2 canonical collections are not initialized")
	}
	for id, persona := range state.Personas {
		if id != persona.ID || persona.SchemaVersion != 1 || persona.Status != PersonaActive && persona.Status != PersonaRetired || persona.Digest != DigestText(persona.Instructions) {
			return fmt.Errorf("invalid immutable persona %q", id)
		}
	}
	for id, profile := range state.RuntimeProfiles {
		if id != profile.ID || profile.SchemaVersion != 1 || profile.MaxTurns < 1 || profile.MaxMessages > MaxProjectedMessages || profile.MaxContextBytes > MaxProjectedContextBytes || profile.MemoryRead || profile.MemoryWrite || profile.Digest != RuntimeProfileDigest(profile) {
			return fmt.Errorf("invalid runtime profile %q", id)
		}
	}
	for id, conversation := range state.Conversations {
		if id != conversation.ID || conversation.NextSequence == 0 && len(state.Messages[id]) > 0 || conversation.SizeBytes < 0 || conversation.SizeBytes > MaxConversationBytes {
			return fmt.Errorf("invalid conversation %q", id)
		}
		messages := state.Messages[id]
		if len(messages) > 0 && conversation.NextSequence != uint64(len(messages)+1) {
			return fmt.Errorf("conversation %q sequence is not contiguous", id)
		}
		for i, message := range messages {
			if message.ConversationID != id || message.Sequence != uint64(i+1) || (message.Role != MessageUser && message.Role != MessageAssistant) || len(message.Content) > MaxChatMessageBytes || message.ContentDigest != DigestText(message.Content) {
				return fmt.Errorf("invalid message %q", message.ID)
			}
		}
	}
	for id, messages := range state.Messages {
		if _, ok := state.Conversations[id]; !ok && len(messages) != 0 {
			return fmt.Errorf("messages without conversation %q", id)
		}
	}
	return nil
}

// NormalizeState restores initialized empty v2 collections omitted by the
// compatibility JSON tags. It is used only after strict decoding persisted
// state; callers constructing a new v2 state must initialize its collections.
func NormalizeState(state State) State {
	if state.SchemaVersion != StateSchemaVersionV2 {
		return state
	}
	if state.Conversations == nil {
		state.Conversations = map[string]Conversation{}
	}
	if state.Messages == nil {
		state.Messages = map[string][]Message{}
	}
	if state.Personas == nil {
		state.Personas = map[string]Persona{}
	}
	if state.Surfaces == nil {
		state.Surfaces = map[string]Surface{}
	}
	if state.ContextRecords == nil {
		state.ContextRecords = map[string]ContextRecord{}
	}
	if state.RuntimeSessions == nil {
		state.RuntimeSessions = map[string]RuntimeSessionMapping{}
	}
	if state.RuntimeProfiles == nil {
		state.RuntimeProfiles = map[string]RuntimeProfile{}
	}
	if state.RuntimeCertifications == nil {
		state.RuntimeCertifications = map[string]RuntimeCertification{}
	}
	return state
}
