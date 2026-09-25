package space

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	StateSchemaVersionV1      = 1
	StateSchemaVersionV2      = 2
	CurrentStateSchemaVersion = StateSchemaVersionV2

	MaxChatMessageBytes      = 64 << 10
	MaxConversationBytes     = 512 << 10
	MaxProjectedMessages     = 20
	MaxProjectedContextBytes = 32 << 10
)

type ConversationStatus string

const (
	ConversationActive   ConversationStatus = "active"
	ConversationArchived ConversationStatus = "archived"
)

type MessageRole string

const (
	MessageUser      MessageRole = "user"
	MessageAssistant MessageRole = "assistant"
)

type PersonaStatus string

const (
	PersonaActive  PersonaStatus = "active"
	PersonaRetired PersonaStatus = "retired"
)

type Conversation struct {
	ID           string             `json:"id"`
	OwnerID      string             `json:"ownerId"`
	SurfaceID    string             `json:"surfaceId"`
	PersonaID    string             `json:"personaId"`
	Status       ConversationStatus `json:"status"`
	CreatedAt    int64              `json:"createdAt"`
	UpdatedAt    int64              `json:"updatedAt"`
	NextSequence uint64             `json:"nextSequence"`
	SizeBytes    int                `json:"sizeBytes"`
}

type Message struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversationId"`
	Sequence       uint64      `json:"sequence"`
	Role           MessageRole `json:"role"`
	AuthorID       string      `json:"authorId"`
	SurfaceID      string      `json:"surfaceId"`
	Content        string      `json:"content"`
	ContentDigest  string      `json:"contentDigest"`
	CreatedAt      int64       `json:"createdAt"`
	TaskID         string      `json:"taskId,omitempty"`
}

type Persona struct {
	ID            string        `json:"id"`
	SchemaVersion int           `json:"schemaVersion"`
	Version       int           `json:"version"`
	Instructions  string        `json:"instructions"`
	Digest        string        `json:"digest"`
	Status        PersonaStatus `json:"status"`
}

type Surface struct {
	ID            string `json:"id"`
	SchemaVersion int    `json:"schemaVersion"`
	Type          string `json:"type"`
}

type ContextRecord struct {
	ID             string          `json:"id"`
	Namespace      string          `json:"namespace"`
	SchemaVersion  int             `json:"schemaVersion"`
	OriginNodeID   string          `json:"originNodeId"`
	Version        int             `json:"version"`
	Provenance     string          `json:"provenance"`
	Classification string          `json:"classification"`
	RetentionUntil int64           `json:"retentionUntil"`
	AcceptedAt     int64           `json:"acceptedAt"`
	Digest         string          `json:"digest"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

type RuntimeSessionMapping struct {
	ConversationID string `json:"conversationId"`
	// Pending reserves the session before runtime certification can perform I/O.
	Pending               bool   `json:"pending,omitempty"`
	RuntimeIdentity       string `json:"runtimeIdentity"`
	HermesSessionID       string `json:"hermesSessionId"`
	PersonaDigest         string `json:"personaDigest"`
	ProfileDigest         string `json:"profileDigest"`
	HostEpoch             int    `json:"hostEpoch"`
	LastProjectedSequence uint64 `json:"lastProjectedSequence"`
}

type RuntimeProfile struct {
	ID                string   `json:"id"`
	SchemaVersion     int      `json:"schemaVersion"`
	PersonaID         string   `json:"personaId"`
	PersonaDigest     string   `json:"personaDigest"`
	Provider          string   `json:"provider"`
	Model             string   `json:"model"`
	AllowedFeatureSet []string `json:"allowedFeatureSet"`
	MemoryRead        bool     `json:"memoryRead"`
	MemoryWrite       bool     `json:"memoryWrite"`
	MaxTurns          int      `json:"maxTurns"`
	MaxMessages       int      `json:"maxMessages"`
	MaxContextBytes   int      `json:"maxContextBytes"`
	MaxInputTokens    int      `json:"maxInputTokens"`
	MaxOutputTokens   int      `json:"maxOutputTokens"`
	MaxTotalTokens    int      `json:"maxTotalTokens"`
	DeadlineSeconds   int      `json:"deadlineSeconds"`
	Digest            string   `json:"digest"`
}

type RuntimeProfileLimits struct {
	Version         int `json:"version"`
	MaxTurns        int `json:"maxTurns"`
	MaxMessages     int `json:"maxMessages"`
	MaxContextBytes int `json:"maxContextBytes"`
	MaxInputTokens  int `json:"maxInputTokens"`
	MaxOutputTokens int `json:"maxOutputTokens"`
	MaxTotalTokens  int `json:"maxTotalTokens"`
	DeadlineSeconds int `json:"deadlineSeconds"`
}

type RuntimeCertification struct {
	ID                string               `json:"id"`
	RuntimeIdentity   string               `json:"runtimeIdentity"`
	EndpointIdentity  string               `json:"endpointIdentity"`
	ArtifactDigest    string               `json:"artifactDigest,omitempty"`
	ProcessIdentity   string               `json:"processIdentity,omitempty"`
	HermesVersion     string               `json:"hermesVersion"`
	PluginIdentity    string               `json:"pluginIdentity"`
	PluginCommit      string               `json:"pluginCommit"`
	ProfileDigest     string               `json:"profileDigest"`
	ConfigDigest      string               `json:"configDigest"`
	EffectiveToolsets []string             `json:"effectiveToolsets"`
	MemoryRead        bool                 `json:"memoryRead"`
	MemoryWrite       bool                 `json:"memoryWrite"`
	Limits            RuntimeProfileLimits `json:"limits"`
	Evidence          string               `json:"evidence"`
	ExpiresAt         int64                `json:"expiresAt"`
}

const DefaultPersonaInstructions = "You are Lumen, the user's private Space intelligence. Use only canonical context supplied by the Host. Distinguish known context from inference. Never claim memory, permissions, tools, or completed actions not represented by Host records. Ask when ambiguity changes privacy, authority, or target selection."

func DigestText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func DefaultPersona() Persona {
	return Persona{ID: "lumen.persona.default/v1", SchemaVersion: 1, Version: 1, Instructions: DefaultPersonaInstructions, Digest: DigestText(DefaultPersonaInstructions), Status: PersonaActive}
}

func DefaultRuntimeProfile() RuntimeProfile {
	p := RuntimeProfile{ID: "lumen.chat.default/v1", SchemaVersion: 1, PersonaID: DefaultPersona().ID, PersonaDigest: DefaultPersona().Digest, AllowedFeatureSet: []string{}, MemoryRead: false, MemoryWrite: false, MaxTurns: 1, MaxMessages: MaxProjectedMessages, MaxContextBytes: MaxProjectedContextBytes}
	p.Digest = RuntimeProfileDigest(p)
	return p
}

func RuntimeProfileDigest(profile RuntimeProfile) string {
	profile.Digest = ""
	b, _ := json.Marshal(profile)
	return DigestText(string(b))
}
