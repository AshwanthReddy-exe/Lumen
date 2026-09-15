package space

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCanonicalRecordsAreVersionedAndBounded(t *testing.T) {
	if MaxChatMessageBytes != 64<<10 || MaxConversationBytes != 512<<10 {
		t.Fatalf("message/conversation bounds = %d/%d", MaxChatMessageBytes, MaxConversationBytes)
	}
	if MaxProjectedMessages != 20 || MaxProjectedContextBytes != 32<<10 {
		t.Fatalf("projection bounds = %d/%d", MaxProjectedMessages, MaxProjectedContextBytes)
	}

	persona := DefaultPersona()
	profile := DefaultRuntimeProfile()
	state := State{
		SchemaVersion: 2,
		SpaceID:       "space",
		OwnerID:       "owner",
		HostID:        "host",
		Audit:         []AuditEvent{},
		Conversations: map[string]Conversation{"conversation": {ID: "conversation", OwnerID: "owner", SurfaceID: "web", PersonaID: persona.ID, Status: ConversationActive, NextSequence: 2, SizeBytes: len("hello")}},
		Messages: map[string][]Message{"conversation": {
			{ID: "message-1", ConversationID: "conversation", Sequence: 1, Role: MessageUser, AuthorID: "owner", SurfaceID: "web", Content: "hello", ContentDigest: DigestText("hello")},
		}},
		Personas:              map[string]Persona{persona.ID: persona},
		Surfaces:              map[string]Surface{"web": {ID: "web", Type: "web", SchemaVersion: 1}},
		ContextRecords:        map[string]ContextRecord{"preference": {ID: "preference", Namespace: "user.preferences/v1", SchemaVersion: 1, Provenance: "owner", Classification: "private", AcceptedAt: 1, Payload: json.RawMessage(`"Ada"`), Digest: DigestText(`"Ada"`)}},
		RuntimeProfiles:       map[string]RuntimeProfile{profile.ID: profile},
		RuntimeCertifications: map[string]RuntimeCertification{"cert": {ID: "cert", RuntimeIdentity: "hermes", EndpointIdentity: "endpoint", ProfileDigest: profile.Digest, Limits: profileLimits(profile), ExpiresAt: 100}},
		RuntimeSessions:       map[string]RuntimeSessionMapping{"conversation": {ConversationID: "conversation", RuntimeIdentity: "hermes", HermesSessionID: "opaque-session", PersonaDigest: persona.Digest, ProfileDigest: profile.Digest, HostEpoch: 1, LastProjectedSequence: 1}},
	}
	if err := ValidateState(state); err != nil {
		t.Fatal(err)
	}
	state.Personas[persona.ID] = Persona{ID: persona.ID, SchemaVersion: persona.SchemaVersion, Version: persona.Version, Instructions: persona.Instructions, Digest: "sha256:tampered", Status: persona.Status}
	if err := ValidateState(state); err == nil {
		t.Fatal("tampered immutable persona accepted")
	}
}

func TestMigrateV1ToV2PreservesAuthorityAndIsIdempotent(t *testing.T) {
	v1 := State{
		SchemaVersion: 1,
		SpaceID:       "space",
		OwnerID:       "owner",
		HostID:        "host",
		Epoch:         7,
		Identities:    []Identity{{ID: "owner", Kind: IdentityOwner}, {ID: "host", Kind: IdentityHost}},
		Capabilities:  []Capability{{ID: "agent.run/execute", Grant: GrantAsk}},
		Audit:         []AuditEvent{{Event: AuditSpaceCreated, RequestID: "create", ActorID: "owner", HostID: "host", Epoch: 1, Operation: CommandCreateSpace, Outcome: "applied"}},
		Nodes:         map[string]Node{"owner": {ID: "owner", Status: "paired"}, "host": {ID: "host", Status: "paired"}},
		Grants:        map[string]Grant{"host|agent.run/execute|run": GrantAsk},
		Tasks:         map[string]Task{"task": {ID: "task", CommandID: "submit", HostEpoch: 7, Status: OutcomeQueued}},
		Commands:      map[string]RecordedCommand{"create": {Type: CommandCreateSpace, Content: "create"}},
	}
	v2, err := MigrateV1ToV2(v1)
	if err != nil {
		t.Fatal(err)
	}
	if v2.SchemaVersion != 2 || v2.SpaceID != v1.SpaceID || v2.OwnerID != v1.OwnerID || v2.HostID != v1.HostID || v2.Epoch != v1.Epoch {
		t.Fatalf("authority changed: %#v", v2)
	}
	if len(v2.Identities) != len(v1.Identities) || len(v2.Audit) != len(v1.Audit) || len(v2.Nodes) != len(v1.Nodes) || len(v2.Tasks) != len(v1.Tasks) || len(v2.Commands) != len(v1.Commands) {
		t.Fatal("existing authority records were not preserved")
	}
	if len(v2.Conversations) != 0 || len(v2.Messages) != 0 || len(v2.ContextRecords) != 0 || len(v2.RuntimeSessions) != 0 {
		t.Fatal("new collections were not initialized empty")
	}
	if len(v2.Personas) != 1 || len(v2.RuntimeProfiles) != 1 {
		t.Fatal("default persona/profile missing")
	}
	again, err := MigrateV1ToV2(v2)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateState(again); err != nil {
		t.Fatal(err)
	}
	if again.SchemaVersion != v2.SchemaVersion || !reflect.DeepEqual(again.Personas, v2.Personas) || !reflect.DeepEqual(again.RuntimeProfiles, v2.RuntimeProfiles) {
		t.Fatal("migration was not idempotent")
	}
	if _, err := MigrateV1ToV2(State{SchemaVersion: 3}); err == nil {
		t.Fatal("future schema accepted")
	}
}

func TestRuntimeCertificationAttestsCompatibleProfileLimits(t *testing.T) {
	state := canonicalValidationState()
	profile := state.RuntimeProfiles[DefaultRuntimeProfile().ID]
	state.RuntimeCertifications["cert"] = RuntimeCertification{
		ID: "cert", RuntimeIdentity: "hermes", EndpointIdentity: "endpoint", ProfileDigest: profile.Digest,
		Limits:    RuntimeProfileLimits{Version: 1, MaxTurns: profile.MaxTurns, MaxMessages: profile.MaxMessages, MaxContextBytes: profile.MaxContextBytes, MaxInputTokens: profile.MaxInputTokens, MaxOutputTokens: profile.MaxOutputTokens, MaxTotalTokens: profile.MaxTotalTokens, DeadlineSeconds: profile.DeadlineSeconds},
		ExpiresAt: 100,
	}
	if err := ValidateState(state); err != nil {
		t.Fatal(err)
	}
	cert := state.RuntimeCertifications["cert"]
	cert.Limits.MaxTurns++
	state.RuntimeCertifications["cert"] = cert
	if err := ValidateState(state); err == nil {
		t.Fatal("certification with incompatible limits accepted")
	}
}

func TestValidateStateComputesConversationByteAccounting(t *testing.T) {
	state := canonicalValidationState()
	conversation := state.Conversations["conversation"]
	conversation.NextSequence = 3
	conversation.SizeBytes = 0
	state.Conversations[conversation.ID] = conversation
	state.Messages[conversation.ID] = []Message{
		{ID: "message-1", ConversationID: conversation.ID, Sequence: 1, Role: MessageUser, Content: strings.Repeat("a", MaxChatMessageBytes)},
		{ID: "message-2", ConversationID: conversation.ID, Sequence: 2, Role: MessageAssistant, Content: strings.Repeat("b", MaxChatMessageBytes)},
	}
	for i := range state.Messages[conversation.ID] {
		state.Messages[conversation.ID][i].ContentDigest = DigestText(state.Messages[conversation.ID][i].Content)
	}
	if err := ValidateState(state); err == nil {
		t.Fatal("conversation over byte bound accepted")
	}
	state.Messages[conversation.ID] = state.Messages[conversation.ID][:1]
	conversation.NextSequence = 2
	conversation.SizeBytes = len(state.Messages[conversation.ID][0].Content)
	state.Conversations[conversation.ID] = conversation
	if err := ValidateState(state); err != nil {
		t.Fatal(err)
	}
	conversation.SizeBytes++
	state.Conversations[conversation.ID] = conversation
	if err := ValidateState(state); err == nil {
		t.Fatal("conversation with inaccurate byte accounting accepted")
	}
}

func TestValidateStateRejectsMalformedCanonicalRecordFamilies(t *testing.T) {
	cases := []struct {
		name string
		edit func(*State)
	}{
		{"surface", func(s *State) { s.Surfaces["web"] = Surface{ID: "other", Type: "web", SchemaVersion: 1} }},
		{"context", func(s *State) {
			s.ContextRecords["preference"] = ContextRecord{ID: "preference", Namespace: "user.preferences/v1", SchemaVersion: 1, Digest: "sha256:wrong", Payload: json.RawMessage(`{"name":"Ada"}`)}
		}},
		{"session", func(s *State) {
			s.RuntimeSessions["conversation"] = RuntimeSessionMapping{ConversationID: "missing", RuntimeIdentity: "hermes", HermesSessionID: "opaque", ProfileDigest: DefaultRuntimeProfile().Digest, PersonaDigest: DefaultPersona().Digest, HostEpoch: 1}
		}},
		{"certification", func(s *State) {
			s.RuntimeCertifications["cert"] = RuntimeCertification{ID: "cert", RuntimeIdentity: "", EndpointIdentity: "endpoint", ProfileDigest: "missing"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := canonicalValidationState()
			tc.edit(&state)
			if err := ValidateState(state); err == nil {
				t.Fatal("malformed canonical record accepted")
			}
		})
	}
}

func TestMigrateRejectsUnserializableStateWithoutMutatingInput(t *testing.T) {
	v1 := State{SchemaVersion: 1, SpaceID: "space", OwnerID: "owner", HostID: "host", Audit: []AuditEvent{}, ContextRecords: map[string]ContextRecord{"bad": {ID: "bad", Payload: json.RawMessage("{")}}}
	before := v1
	if _, err := MigrateV1ToV2(v1); err == nil {
		t.Fatal("unserializable state migrated")
	}
	if !reflect.DeepEqual(v1, before) {
		t.Fatal("failed migration mutated input")
	}
}

func canonicalValidationState() State {
	persona := DefaultPersona()
	profile := DefaultRuntimeProfile()
	return State{
		SchemaVersion: 2, SpaceID: "space", OwnerID: "owner", HostID: "host", Audit: []AuditEvent{},
		Conversations: map[string]Conversation{"conversation": {ID: "conversation", OwnerID: "owner", SurfaceID: "web", PersonaID: persona.ID, Status: ConversationActive, NextSequence: 1}},
		Messages:      map[string][]Message{"conversation": {}}, Personas: map[string]Persona{persona.ID: persona}, Surfaces: map[string]Surface{"web": {ID: "web", Type: "web", SchemaVersion: 1}},
		ContextRecords:  map[string]ContextRecord{"preference": {ID: "preference", Namespace: "user.preferences/v1", SchemaVersion: 1, Provenance: "owner", Classification: "private", AcceptedAt: 1, Digest: DigestText("")}},
		RuntimeProfiles: map[string]RuntimeProfile{profile.ID: profile}, RuntimeSessions: map[string]RuntimeSessionMapping{}, RuntimeCertifications: map[string]RuntimeCertification{},
	}
}

func profileLimits(profile RuntimeProfile) RuntimeProfileLimits {
	return RuntimeProfileLimits{Version: 1, MaxTurns: profile.MaxTurns, MaxMessages: profile.MaxMessages, MaxContextBytes: profile.MaxContextBytes, MaxInputTokens: profile.MaxInputTokens, MaxOutputTokens: profile.MaxOutputTokens, MaxTotalTokens: profile.MaxTotalTokens, DeadlineSeconds: profile.DeadlineSeconds}
}
