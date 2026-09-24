package space

import (
	"strings"
	"testing"
)

func TestConversationCreateAndSendPersistAtomicIntent(t *testing.T) {
	s := conversationState()
	created := Apply(s, Command{
		Type:           CommandCreateConversation,
		SpaceID:        "space",
		HostID:         "host",
		Epoch:          1,
		ActorID:        "owner",
		ConversationID: "conversation-1",
		SurfaceID:      "web",
		RequestID:      "create-1",
	})
	if created.Rejection != "" {
		t.Fatalf("create rejected: %#v", created)
	}

	sent := Apply(created.State, Command{
		Type:                  CommandSendConversation,
		SpaceID:               "space",
		HostID:                "host",
		Epoch:                 1,
		ActorID:               "owner",
		ConversationID:        "conversation-1",
		SurfaceID:             "web",
		TaskID:                "task-1",
		Content:               "hello",
		RuntimeIdempotencyKey: "runtime-task-1",
		RuntimeProfileDigest:  DefaultRuntimeProfile().Digest,
		CreatedAt:             100,
		ReconcileBy:           200,
		RequestID:             "send-1",
	})
	if sent.Rejection != "" || sent.Receipt.Outcome != OutcomeQueued {
		t.Fatalf("send result: %#v", sent)
	}
	if got := len(sent.State.Messages["conversation-1"]); got != 1 {
		t.Fatalf("messages = %d, want 1", got)
	}
	if got := sent.State.Messages["conversation-1"][0]; got.Role != MessageUser || got.Sequence != 1 || got.TaskID != "task-1" || got.Content != "hello" {
		t.Fatalf("user message = %#v", got)
	}
	if got := sent.State.Tasks["task-1"]; got.Status != OutcomeQueued || got.CommandID != "send-1" {
		t.Fatalf("task = %#v", got)
	}
	if got := sent.State.HostCreates["task-1"]; got.IdempotencyKey != "runtime-task-1" || got.RuntimeProfileDigest != DefaultRuntimeProfile().Digest {
		t.Fatalf("runtime intent = %#v", got)
	}
	if err := ValidateState(sent.State); err != nil {
		t.Fatalf("state invalid after send: %v", err)
	}
}

func TestRuntimeSessionBindingRequiresDurableReservation(t *testing.T) {
	s := conversationWithSend(t)
	bind := Command{Type: CommandBindRuntimeSession, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "bind-1", ConversationID: "conversation-1", HermesSessionID: "lumen-session:conversation-1", RuntimeIdentity: "hermes:test", RuntimeProfileDigest: DefaultRuntimeProfile().Digest}
	if tr := Apply(s, bind); tr.Rejection == "" {
		t.Fatal("runtime session bound without durable reservation")
	}
	reserve := bind
	reserve.Type, reserve.RequestID, reserve.RuntimeIdentity = CommandReserveRuntimeSession, "reserve-1", ""
	reserved := Apply(s, reserve)
	if reserved.Rejection != "" || !reserved.State.RuntimeSessions["conversation-1"].Pending {
		t.Fatalf("reservation rejected: %#v", reserved)
	}
	bound := Apply(reserved.State, bind)
	if bound.Rejection != "" || bound.State.RuntimeSessions["conversation-1"].Pending || bound.State.RuntimeSessions["conversation-1"].RuntimeIdentity != "hermes:test" {
		t.Fatalf("reserved session did not bind: %#v", bound)
	}
}

func TestConversationCompletionAppendsDeterministicAssistantOnce(t *testing.T) {
	s := conversationWithSend(t)
	completed := Apply(s, Command{
		Type:      CommandCompleteConversation,
		SpaceID:   "space",
		HostID:    "host",
		Epoch:     1,
		ActorID:   "host",
		TaskID:    "task-1",
		Outcome:   OutcomeCompleted,
		Output:    "world",
		CreatedAt: 300, RequestID: "complete-1",
	})
	if completed.Rejection != "" || completed.Receipt.Outcome != OutcomeCompleted {
		t.Fatalf("completion result: %#v", completed)
	}
	messages := completed.State.Messages["conversation-1"]
	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}
	assistant := messages[1]
	if assistant.ID != AssistantMessageID("task-1") || assistant.Role != MessageAssistant || assistant.Sequence != 2 || assistant.TaskID != "task-1" || assistant.Content != "world" {
		t.Fatalf("assistant message = %#v", assistant)
	}
	duplicate := Apply(completed.State, Command{
		Type:      CommandCompleteConversation,
		SpaceID:   "space",
		HostID:    "host",
		Epoch:     1,
		ActorID:   "host",
		TaskID:    "task-1",
		Outcome:   OutcomeCompleted,
		Output:    "world",
		CreatedAt: 300, RequestID: "complete-2",
	})
	if duplicate.Rejection != "" || len(duplicate.State.Messages["conversation-1"]) != 2 {
		t.Fatalf("duplicate completion: %#v", duplicate)
	}
}

func TestConversationDuplicateAndConflictingRequestIDs(t *testing.T) {
	s := conversationWithSend(t)
	command := Command{
		Type:      CommandCompleteConversation,
		SpaceID:   "space",
		HostID:    "host",
		Epoch:     1,
		ActorID:   "host",
		TaskID:    "task-1",
		Outcome:   OutcomeCompleted,
		Output:    "world",
		RequestID: "complete-idempotent",
	}
	first := Apply(s, command)
	replayed := Apply(first.State, command)
	if !replayed.Replayed || replayed.Rejection != "" || replayed.Receipt != first.Receipt {
		t.Fatalf("replay = %#v, first = %#v", replayed, first)
	}
	command.Output = "different"
	conflict := Apply(first.State, command)
	if conflict.Rejection != "idempotency_key_reused" {
		t.Fatalf("conflict = %#v", conflict)
	}
}

func TestConversationSendRejectsOversizeInputAndPreservesSequence(t *testing.T) {
	s := conversationState()
	created := Apply(s, Command{Type: CommandCreateConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", ConversationID: "conversation-1", SurfaceID: "web", RequestID: "create-1"})
	tooLarge := Apply(created.State, Command{Type: CommandSendConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", ConversationID: "conversation-1", SurfaceID: "web", TaskID: "task-1", Content: strings.Repeat("x", MaxChatMessageBytes+1), CreatedAt: 100, ReconcileBy: 200, RequestID: "send-large"})
	if tooLarge.Rejection != "message_too_large" {
		t.Fatalf("oversize result = %#v", tooLarge)
	}
	if len(tooLarge.State.Messages["conversation-1"]) != 0 || tooLarge.State.Conversations["conversation-1"].NextSequence != 1 {
		t.Fatalf("oversize mutated conversation: %#v", tooLarge.State.Conversations["conversation-1"])
	}
}

func TestConversationCompletionIsMonotonicAndUnknownHasNoAssistant(t *testing.T) {
	s := conversationWithSend(t)
	unknown := Apply(s, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", Outcome: OutcomeUnknown, RequestID: "unknown-1"})
	if unknown.Rejection != "" || unknown.State.Tasks["task-1"].Status != OutcomeUnknown || len(unknown.State.Messages["conversation-1"]) != 1 {
		t.Fatalf("unknown completion: %#v", unknown)
	}
	lateCompleted := Apply(unknown.State, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", Outcome: OutcomeCompleted, Output: "late", RequestID: "complete-late"})
	if lateCompleted.Rejection != "" || lateCompleted.State.Tasks["task-1"].Status != OutcomeUnknown || len(lateCompleted.State.Messages["conversation-1"]) != 1 {
		t.Fatalf("late completion downgraded unknown: %#v", lateCompleted)
	}

	s = conversationWithSend(t)
	completed := Apply(s, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", Outcome: OutcomeCompleted, Output: "world", RequestID: "complete-first"})
	lateUnknown := Apply(completed.State, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", Outcome: OutcomeUnknown, RequestID: "unknown-late"})
	if lateUnknown.Rejection != "" || lateUnknown.State.Tasks["task-1"].Status != OutcomeCompleted || len(lateUnknown.State.Messages["conversation-1"]) != 2 {
		t.Fatalf("late unknown changed completed: %#v", lateUnknown)
	}
}

func TestConversationSendRequiresDispatchableDeadlineAndMatchingDispatch(t *testing.T) {
	created := Apply(conversationState(), Command{Type: CommandCreateConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", ConversationID: "conversation-1", SurfaceID: "web", RequestID: "create-1"})
	for name, edit := range map[string]func(*Command){
		"missing created at":      func(c *Command) { c.CreatedAt = 0; c.ReconcileBy = 200 },
		"nonpositive created at":  func(c *Command) { c.CreatedAt = -1; c.ReconcileBy = 200 },
		"deadline at created":     func(c *Command) { c.CreatedAt = 100; c.ReconcileBy = 100 },
		"deadline before created": func(c *Command) { c.CreatedAt = 100; c.ReconcileBy = 99 },
	} {
		command := conversationSendCommand()
		command.RequestID = "send-deadline-" + name
		edit(&command)
		tr := Apply(created.State, command)
		if tr.Rejection != "invalid_deadline" {
			t.Fatalf("%s: %#v", name, tr)
		}
		if len(tr.State.Messages["conversation-1"]) != 0 || len(tr.State.HostCreates) != 0 || len(tr.State.Tasks) != 0 {
			t.Fatalf("%s mutated canonical state: %#v", name, tr.State)
		}
	}

	sent := Apply(created.State, conversationSendCommand())
	if sent.Rejection != "" {
		t.Fatalf("send rejected: %#v", sent)
	}
	dispatched := Apply(sent.State, Command{Type: CommandDispatchHostRun, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", RuntimeRunID: "runtime-run-1", RuntimeIdempotencyKey: "runtime-task-1", RuntimeProfileDigest: DefaultRuntimeProfile().Digest, DispatchedAt: 100, ReconcileBy: 200, RequestID: "dispatch-1"})
	if dispatched.Rejection != "" || dispatched.State.Tasks["task-1"].Status != OutcomeDispatched {
		t.Fatalf("matching dispatch: %#v", dispatched)
	}
}

func TestConversationSendRequiresExactSurfaceIdentity(t *testing.T) {
	created := Apply(conversationState(), Command{Type: CommandCreateConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", ConversationID: "conversation-1", SurfaceID: "web", RequestID: "create-1"})
	for name, surface := range map[string]string{"missing": "", "wrong": "desktop"} {
		command := conversationSendCommand()
		command.RequestID = "send-surface-" + name
		command.SurfaceID = surface
		tr := Apply(created.State, command)
		want := "surface_mismatch"
		if surface == "" {
			want = "invalid_identifier"
		}
		if tr.Rejection != want {
			t.Fatalf("%s: got %#v, want %q", name, tr, want)
		}
	}
}

func TestConversationCompletionRejectsTerminalGenericTaskBeforeMonotonicShortCircuit(t *testing.T) {
	s := conversationWithSend(t)
	task := s.Tasks["task-1"]
	task.Status = OutcomeCompleted
	task.CapabilityID = "agent.run/execute"
	s.Tasks[task.ID] = task
	tr := Apply(s, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", Outcome: OutcomeCompleted, Output: "late", RequestID: "generic-capability"})
	if tr.Rejection != "conversation_capability_mismatch" {
		t.Fatalf("generic capability: %#v", tr)
	}

	s = conversationState()
	s.Tasks = map[string]Task{"generic": {ID: "generic", CapabilityID: "agent.run/execute", Status: OutcomeCompleted}}
	tr = Apply(s, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "generic", Outcome: OutcomeCompleted, Output: "late", RequestID: "generic-linkage"})
	if tr.Rejection != "conversation_unknown" {
		t.Fatalf("generic linkage: %#v", tr)
	}
}

func TestConversationCompletionValidatesLateTerminalOutputBounds(t *testing.T) {
	s := conversationWithSend(t)
	completed := Apply(s, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", Outcome: OutcomeCompleted, Output: "world", RequestID: "complete-first"})
	for name, output := range map[string]string{
		"oversize":     strings.Repeat("x", MaxChatMessageBytes+1),
		"invalid utf8": string([]byte{0xff}),
	} {
		tr := Apply(completed.State, Command{Type: CommandCompleteConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", Outcome: OutcomeCompleted, Output: output, RequestID: "late-output-" + name})
		want := "message_too_large"
		if name == "invalid utf8" {
			want = "invalid_utf8"
		}
		if tr.Rejection != want {
			t.Fatalf("%s: %#v, want %q", name, tr, want)
		}
	}
}

func conversationWithSend(t *testing.T) State {
	t.Helper()
	created := Apply(conversationState(), Command{Type: CommandCreateConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", ConversationID: "conversation-1", SurfaceID: "web", RequestID: "create-1"})
	if created.Rejection != "" {
		t.Fatalf("create rejected: %#v", created)
	}
	sent := Apply(created.State, conversationSendCommand())
	if sent.Rejection != "" {
		t.Fatalf("send rejected: %#v", sent)
	}
	return sent.State
}

func conversationSendCommand() Command {
	return Command{Type: CommandSendConversation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", ConversationID: "conversation-1", SurfaceID: "web", TaskID: "task-1", Content: "hello", RuntimeIdempotencyKey: "runtime-task-1", RuntimeProfileDigest: DefaultRuntimeProfile().Digest, CreatedAt: 100, ReconcileBy: 200, RequestID: "send-1"}
}

func conversationState() State {
	persona := DefaultPersona()
	profile := DefaultRuntimeProfile()
	return State{
		SchemaVersion:         StateSchemaVersionV2,
		SpaceID:               "space",
		OwnerID:               "owner",
		HostID:                "host",
		Epoch:                 1,
		Audit:                 []AuditEvent{},
		Conversations:         map[string]Conversation{},
		Messages:              map[string][]Message{},
		Personas:              map[string]Persona{persona.ID: persona},
		Surfaces:              map[string]Surface{"web": {ID: "web", SchemaVersion: 1, Type: "web"}},
		ContextRecords:        map[string]ContextRecord{},
		RuntimeSessions:       map[string]RuntimeSessionMapping{},
		RuntimeProfiles:       map[string]RuntimeProfile{profile.ID: profile},
		RuntimeCertifications: map[string]RuntimeCertification{},
	}
}
