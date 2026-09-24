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
		state = NormalizeState(state)
		if err := ValidateState(state); err != nil {
			return State{}, err
		}
		return cloneCanonicalState(state)
	}
	if state.SchemaVersion != StateSchemaVersionV1 {
		return State{}, fmt.Errorf("unsupported state schema version %d", state.SchemaVersion)
	}
	migrated, err := cloneCanonicalState(state)
	if err != nil {
		return State{}, err
	}
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

func cloneCanonicalState(state State) (State, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return State{}, fmt.Errorf("serialize state for migration: %w", err)
	}
	var copy State
	if err := json.Unmarshal(b, &copy); err != nil {
		return State{}, fmt.Errorf("deserialize state for migration: %w", err)
	}
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
	return copy, nil
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
	if state.Personas == nil || state.RuntimeProfiles == nil || state.Conversations == nil || state.Messages == nil || state.Surfaces == nil || state.ContextRecords == nil || state.RuntimeSessions == nil || state.RuntimeCertifications == nil {
		return errors.New("schema v2 canonical collections are not initialized")
	}
	for id, persona := range state.Personas {
		if id != persona.ID || persona.SchemaVersion != 1 || persona.Status != PersonaActive && persona.Status != PersonaRetired || persona.Digest != DigestText(persona.Instructions) {
			return fmt.Errorf("invalid immutable persona %q", id)
		}
	}
	for id, profile := range state.RuntimeProfiles {
		if id != profile.ID || profile.SchemaVersion != 1 || profile.PersonaID == "" || profile.PersonaDigest == "" || profile.MaxTurns < 1 || profile.MaxMessages > MaxProjectedMessages || profile.MaxContextBytes > MaxProjectedContextBytes || profile.MemoryRead || profile.MemoryWrite || profile.Digest != RuntimeProfileDigest(profile) {
			return fmt.Errorf("invalid runtime profile %q", id)
		}
		persona, ok := state.Personas[profile.PersonaID]
		if !ok || persona.Digest != profile.PersonaDigest {
			return fmt.Errorf("runtime profile %q references unknown persona", id)
		}
	}
	for id, conversation := range state.Conversations {
		if id != conversation.ID || conversation.OwnerID == "" || conversation.SurfaceID == "" || conversation.PersonaID == "" || conversation.Status != ConversationActive && conversation.Status != ConversationArchived || conversation.NextSequence == 0 && len(state.Messages[id]) > 0 || conversation.SizeBytes < 0 || conversation.SizeBytes > MaxConversationBytes {
			return fmt.Errorf("invalid conversation %q", id)
		}
		if _, ok := state.Surfaces[conversation.SurfaceID]; !ok {
			return fmt.Errorf("conversation %q references unknown surface", id)
		}
		if persona, ok := state.Personas[conversation.PersonaID]; !ok || persona.Status != PersonaActive {
			return fmt.Errorf("conversation %q references unknown persona", id)
		}
		messages := state.Messages[id]
		if actual := ConversationSizeBytes(messages); actual != conversation.SizeBytes || actual > MaxConversationBytes {
			return fmt.Errorf("conversation %q has invalid byte accounting", id)
		}
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
	for id, surface := range state.Surfaces {
		if id != surface.ID || surface.SchemaVersion != 1 || surface.Type == "" {
			return fmt.Errorf("invalid surface %q", id)
		}
	}
	for id, record := range state.ContextRecords {
		if id != record.ID || record.Namespace == "" || record.SchemaVersion != 1 || record.Provenance == "" || record.Classification == "" || record.Digest == "" || record.AcceptedAt <= 0 {
			return fmt.Errorf("invalid context record %q", id)
		}
		if len(record.Payload) > 0 && !json.Valid(record.Payload) {
			return fmt.Errorf("invalid context payload %q", id)
		}
		if record.Digest != DigestText(string(record.Payload)) {
			return fmt.Errorf("context record %q digest mismatch", id)
		}
	}
	for id, session := range state.RuntimeSessions {
		if id != session.ConversationID || session.ConversationID == "" || session.RuntimeIdentity == "" || session.HermesSessionID == "" || session.PersonaDigest == "" || session.ProfileDigest == "" || session.HostEpoch <= 0 {
			return fmt.Errorf("invalid runtime session %q", id)
		}
		conversation, ok := state.Conversations[session.ConversationID]
		if !ok {
			return fmt.Errorf("runtime session %q references unknown conversation", id)
		}
		persona, ok := state.Personas[conversation.PersonaID]
		if !ok || persona.Digest != session.PersonaDigest {
			return fmt.Errorf("runtime session %q persona mismatch", id)
		}
		profile, ok := profileByDigest(state.RuntimeProfiles, session.ProfileDigest)
		if !ok || profile.PersonaDigest != session.PersonaDigest {
			return fmt.Errorf("runtime session %q profile mismatch", id)
		}
	}
	for id, certification := range state.RuntimeCertifications {
		if id != certification.ID || certification.RuntimeIdentity == "" || certification.EndpointIdentity == "" || certification.ProfileDigest == "" || certification.ExpiresAt <= 0 || certification.Limits.Version != 1 {
			return fmt.Errorf("invalid runtime certification %q", id)
		}
		profile, ok := profileByDigest(state.RuntimeProfiles, certification.ProfileDigest)
		if !ok || !limitsCompatible(certification.Limits, profile) || !stringSetEqual(certification.EffectiveToolsets, profile.AllowedFeatureSet) || certification.MemoryRead != profile.MemoryRead || certification.MemoryWrite != profile.MemoryWrite {
			return fmt.Errorf("runtime certification %q does not match profile", id)
		}
	}
	return nil
}

func ConversationSizeBytes(messages []Message) int {
	total := 0
	for _, message := range messages {
		total += len(message.Content)
	}
	return total
}

func profileByDigest(profiles map[string]RuntimeProfile, digest string) (RuntimeProfile, bool) {
	for _, profile := range profiles {
		if profile.Digest == digest {
			return profile, true
		}
	}
	return RuntimeProfile{}, false
}

func limitsCompatible(limits RuntimeProfileLimits, profile RuntimeProfile) bool {
	return limits.MaxTurns == profile.MaxTurns && limits.MaxMessages == profile.MaxMessages && limits.MaxContextBytes == profile.MaxContextBytes && limits.MaxInputTokens == profile.MaxInputTokens && limits.MaxOutputTokens == profile.MaxOutputTokens && limits.MaxTotalTokens == profile.MaxTotalTokens && limits.DeadlineSeconds == profile.DeadlineSeconds
}

func stringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]struct{}, len(a))
	for _, value := range a {
		seen[value] = struct{}{}
	}
	for _, value := range b {
		if _, ok := seen[value]; !ok {
			return false
		}
	}
	return true
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
