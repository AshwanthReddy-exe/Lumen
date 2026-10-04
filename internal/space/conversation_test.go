package space

import (
	"strings"
	"testing"
)

func TestOwnerMemoryLifecycleControlsFutureContext(t *testing.T) {
	s := conversationState()
	saved := Apply(s, Command{Type: CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "save-1", MemoryID: "fact-1", Content: "I prefer concise answers", CreatedAt: 100})
	if saved.Rejection != "" || saved.State.ContextRecords["fact-1"].Namespace != "user.memory/v1" {
		t.Fatalf("memory save: %#v", saved)
	}
	if tr := Apply(saved.State, Command{Type: CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "save-1", MemoryID: "fact-1", Content: "I prefer concise answers", CreatedAt: 100}); !tr.Replayed || tr.Rejection != "" {
		t.Fatalf("identical save did not replay: %#v", tr)
	}
	if tr := Apply(saved.State, Command{Type: CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "save-1", MemoryID: "fact-1", Content: "changed", CreatedAt: 100}); tr.Rejection != "idempotency_key_reused" {
		t.Fatalf("changed save reused key: %#v", tr)
	}
	denied := Apply(saved.State, Command{Type: CommandDeleteMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "delete-denied", MemoryID: "fact-1"})
	if denied.Rejection == "" || denied.State.ContextRecords["fact-1"].ID == "" {
		t.Fatalf("non-owner deleted memory: %#v", denied)
	}
	deleted := Apply(saved.State, Command{Type: CommandDeleteMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "delete-1", MemoryID: "fact-1"})
	if deleted.Rejection != "" {
		t.Fatalf("memory delete: %#v", deleted)
	}
	if _, exists := deleted.State.ContextRecords["fact-1"]; exists {
		t.Fatal("deleted memory remained in canonical Space")
	}
	for _, command := range deleted.State.Commands {
		if strings.Contains(command.Content, "I prefer concise answers") {
			t.Fatal("deleted memory retained in idempotency ledger")
		}
	}
}

func TestMemoryIDCannotCollideWithPreferenceRecord(t *testing.T) {
	s := conversationState()
	tr := Apply(s, Command{Type: CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "save", MemoryID: "user.preference.locale", Content: "collision", CreatedAt: 1})
	if tr.Rejection == "" {
		t.Fatal("memory claimed preference key")
	}
}

func TestMemoryRejectsOversizeAndUnboundedGrowth(t *testing.T) {
	s := conversationState()
	for _, input := range []Command{
		{MemoryID: strings.Repeat("x", 129), Content: "small", CreatedAt: 1},
		{MemoryID: "small", Content: strings.Repeat("x", 4097), CreatedAt: 1},
	} {
		input.Type, input.SpaceID, input.HostID, input.Epoch, input.ActorID, input.RequestID = CommandSaveMemory, "space", "host", 1, "owner", "bad"
		if tr := Apply(s, input); tr.Rejection == "" {
			t.Fatalf("oversize memory accepted: %#v", tr)
		}
	}
	for i := 0; i < 256; i++ {
		id := string(rune(0x1000 + i))
		tr := Apply(s, Command{Type: CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "save-" + id, MemoryID: id, Content: "small", CreatedAt: 1})
		if tr.Rejection != "" {
			t.Fatalf("memory %d rejected: %s", i, tr.Rejection)
		}
		s = tr.State
	}
	if tr := Apply(s, Command{Type: CommandSaveMemory, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "overflow", MemoryID: "last", Content: "small", CreatedAt: 1}); tr.Rejection == "" {
		t.Fatal("memory limit bypassed")
	}
}

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

func TestRuntimeCertificationInvalidationRequiresExactHostRun(t *testing.T) {
	s := conversationWithSend(t)
	reserve := Apply(s, Command{Type: CommandReserveRuntimeSession, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "reserve", ConversationID: "conversation-1", HermesSessionID: "session-1", RuntimeProfileDigest: DefaultRuntimeProfile().Digest})
	bind := Apply(reserve.State, Command{Type: CommandBindRuntimeSession, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "bind", ConversationID: "conversation-1", HermesSessionID: "session-1", RuntimeIdentity: "hermes:test", RuntimeProfileDigest: DefaultRuntimeProfile().Digest})
	dispatch := Apply(bind.State, Command{Type: CommandDispatchHostRun, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "dispatch", TaskID: "task-1", RuntimeRunID: "run-1", RuntimeIdempotencyKey: "runtime-task-1", RuntimeProfileDigest: DefaultRuntimeProfile().Digest, CertificationID: "cert-1", EndpointIdentity: "endpoint:test", DispatchedAt: 100, ReconcileBy: 200})
	if reserve.Rejection != "" || bind.Rejection != "" || dispatch.Rejection != "" {
		t.Fatalf("test setup rejected: reserve=%q bind=%q dispatch=%q", reserve.Rejection, bind.Rejection, dispatch.Rejection)
	}
	cert := RuntimeCertification{ID: "cert-1", RuntimeIdentity: "hermes:test", ProfileDigest: DefaultRuntimeProfile().Digest}
	command := Command{Type: CommandInvalidateRuntimeCertification, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: RuntimeCertificationInvalidationID(cert), CertificationID: cert.ID, RuntimeIdentity: cert.RuntimeIdentity, RuntimeProfileDigest: cert.ProfileDigest, TaskID: "task-1", RuntimeRunID: "run-1"}
	wrong := command
	wrong.RuntimeRunID = "other-run"
	if tr := Apply(dispatch.State, wrong); tr.Rejection == "" || RuntimeCertificationInvalidated(tr.State, cert) {
		t.Fatalf("mismatched run invalidated certification: %#v", tr)
	}
	wrong = command
	wrong.ActorID = "owner"
	if tr := Apply(dispatch.State, wrong); tr.Rejection == "" || RuntimeCertificationInvalidated(tr.State, cert) {
		t.Fatalf("owner invalidated certification: %#v", tr)
	}
	invalidated := Apply(dispatch.State, command)
	if invalidated.Rejection != "" || !RuntimeCertificationInvalidated(invalidated.State, cert) {
		t.Fatalf("valid violation not recorded: %#v", invalidated)
	}
	if err := ValidateState(invalidated.State); err != nil {
		t.Fatalf("invalidated state failed validation: %v", err)
	}
}

func TestGenericRunReconciliationCannotCompleteConversationTask(t *testing.T) {
	s := conversationWithSend(t)
	s.Nodes = map[string]Node{"owner": {ID: "owner", Status: "paired"}}
	generic := Apply(s, Command{Type: CommandComplete, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "complete-generic-chat", TaskID: "task-1", Outcome: OutcomeCompleted})
	if generic.Rejection == "" || generic.State.Tasks["task-1"].Status == OutcomeCompleted {
		t.Fatalf("generic task completion bypassed conversation evidence: %#v", generic)
	}
	profile := DefaultRuntimeProfile()
	dispatched := Apply(s, Command{Type: CommandDispatchHostRun, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "dispatch-chat", TaskID: "task-1", RuntimeRunID: "run-1", RuntimeIdempotencyKey: "runtime-task-1", RuntimeProfileDigest: profile.Digest, CertificationID: "cert-1", EndpointIdentity: "endpoint:test", DispatchedAt: 100, ReconcileBy: 200})
	if dispatched.Rejection != "" {
		t.Fatalf("dispatch rejected: %q", dispatched.Rejection)
	}
	cancel := Apply(dispatched.State, Command{Type: CommandRequestHostRunCancellation, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "owner", RequestID: "cancel-generic-chat", TaskID: "task-1", RuntimeRunID: "run-1", RuntimeProfileDigest: profile.Digest, ObservedAt: 101})
	if cancel.Rejection == "" || cancel.State.Tasks["task-1"].Status == OutcomeCancelling {
		t.Fatalf("generic cancellation stranded conversation task: %#v", cancel)
	}
	approval := Apply(dispatched.State, Command{Type: CommandRequestRuntimeApproval, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "approval-generic-chat", TaskID: "task-1", RuntimeRunID: "run-1", RuntimeProfileDigest: profile.Digest, RuntimeApprovalID: "approval-1", TargetNodeID: "host", ActionFingerprint: DigestText("hello"), ObservedAt: 101, ExpiresAt: 150})
	if approval.Rejection == "" || approval.State.Tasks["task-1"].Status == OutcomeAwaitingPermission {
		t.Fatalf("generic approval stranded conversation task: %#v", approval)
	}
	tr := Apply(dispatched.State, Command{Type: CommandReconcileHostRun, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", RequestID: "reconcile-chat", TaskID: "task-1", RuntimeRunID: "run-1", RuntimeProfileDigest: profile.Digest, Evidence: EvidenceCompleted, Output: "unverified answer", ObservedAt: 101})
	if tr.Rejection == "" || tr.State.Tasks["task-1"].Status == OutcomeCompleted || len(tr.State.Messages["conversation-1"]) != 1 {
		t.Fatalf("generic reconciliation bypassed conversation evidence: %#v", tr)
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
	dispatched := Apply(sent.State, Command{Type: CommandDispatchHostRun, SpaceID: "space", HostID: "host", Epoch: 1, ActorID: "host", TaskID: "task-1", RuntimeRunID: "runtime-run-1", RuntimeIdempotencyKey: "runtime-task-1", RuntimeProfileDigest: DefaultRuntimeProfile().Digest, CertificationID: "cert-1", EndpointIdentity: "endpoint:test", DispatchedAt: 100, ReconcileBy: 200, RequestID: "dispatch-1"})
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
