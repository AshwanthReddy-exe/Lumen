package conversation

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"

	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

type ProjectedMessage struct {
	Role    space.MessageRole `json:"role"`
	Content string            `json:"content"`
}

type Projection struct {
	Instructions string             `json:"instructions"`
	Input        string             `json:"input"`
	Messages     []ProjectedMessage `json:"messages"`
	Context      string             `json:"context,omitempty"`
}

// Project deterministically selects the newest bounded canonical messages and
// accepted owner preferences. It never consults Hermes sessions or memory.
func Project(state space.State, conversationID string, persona space.Persona, profile space.RuntimeProfile) (Projection, error) {
	conversation, ok := state.Conversations[conversationID]
	if !ok || persona.Status != space.PersonaActive || persona.Digest != space.DigestText(persona.Instructions) || profile.MaxMessages > space.MaxProjectedMessages || profile.MaxContextBytes > space.MaxProjectedContextBytes {
		return Projection{}, errors.New("invalid conversation projection")
	}
	messages := state.Messages[conversationID]
	if len(messages) > profile.MaxMessages && profile.MaxMessages > 0 {
		messages = messages[len(messages)-profile.MaxMessages:]
	} else if len(messages) > space.MaxProjectedMessages {
		messages = messages[len(messages)-space.MaxProjectedMessages:]
	}
	projected := make([]ProjectedMessage, 0, len(messages))
	for _, message := range messages {
		projected = append(projected, ProjectedMessage{Role: message.Role, Content: message.Content})
	}
	contextText := projectedPreferences(state.ContextRecords, profile.MaxContextBytes)
	instructions := persona.Instructions
	if contextText != "" {
		instructions += "\n\nHost-accepted preferences (canonical):\n" + contextText
	}
	input := ""
	if len(messages) > 0 {
		input = messages[len(messages)-1].Content
	}
	_ = conversation
	return Projection{Instructions: instructions, Input: input, Messages: projected, Context: contextText}, nil
}

func projectedPreferences(records map[string]space.ContextRecord, maxBytes int) string {
	type selected struct {
		ID      string          `json:"id"`
		Payload json.RawMessage `json:"payload"`
	}
	items := make([]selected, 0)
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		record := records[id]
		if record.Namespace != "user.preferences/v1" || record.Provenance != "owner" || record.AcceptedAt <= 0 || len(record.Payload) == 0 || !json.Valid(record.Payload) {
			continue
		}
		var typed map[string]string
		if json.Unmarshal(record.Payload, &typed) != nil || len(typed) != 1 {
			continue
		}
		for key := range typed {
			if key != "preferred_name" && key != "communication_style" && key != "locale" {
				delete(typed, key)
			}
		}
		if len(typed) == 0 {
			continue
		}
		payload, _ := json.Marshal(typed)
		items = append(items, selected{ID: id, Payload: payload})
	}
	if len(items) == 0 {
		return ""
	}
	var b bytes.Buffer
	b.WriteByte('[')
	for _, item := range items {
		entry, _ := json.Marshal(item)
		required := len(entry) + 1 // entry plus the closing bracket
		if b.Len() > 1 {
			required++ // separator before every entry after the first
		}
		if maxBytes > 0 && b.Len()+required > maxBytes {
			break
		}
		if b.Len() > 1 {
			b.WriteByte(',')
		}
		b.Write(entry)
	}
	b.WriteByte(']')
	if b.Len() == 2 {
		return ""
	}
	return b.String()
}
