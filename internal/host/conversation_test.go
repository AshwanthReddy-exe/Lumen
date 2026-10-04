package host

import "testing"

func TestMemoryControlArgumentsAreScoped(t *testing.T) {
	if !validConversationArguments("memory save", map[string]string{"request_id": "save", "memory_id": "fact-1", "text": "hello"}) {
		t.Fatal("memory save rejected")
	}
	if validConversationArguments("memory save", map[string]string{"request_id": "save", "memory_id": "fact-1", "text": "hello", "actor_id": "host"}) {
		t.Fatal("caller authority accepted")
	}
	if !validConversationArguments("memory list", map[string]string{}) || !validConversationArguments("memory delete", map[string]string{"request_id": "delete", "memory_id": "fact-1"}) {
		t.Fatal("memory list or delete rejected")
	}
}
