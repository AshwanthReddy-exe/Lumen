package space

import (
	"unicode/utf8"
)

func createConversation(s State, c Command) Transition {
	if reason := conversationContext(s, c, true); reason != "" {
		return reject(s, c, reason)
	}
	if c.ConversationID == "" || c.SurfaceID == "" {
		return reject(s, c, "invalid_identifier")
	}
	if _, exists := s.Conversations[c.ConversationID]; exists {
		return reject(s, c, "conversation_already_exists")
	}
	if _, exists := s.Surfaces[c.SurfaceID]; !exists {
		return reject(s, c, "surface_unknown")
	}
	personaID := c.PersonaID
	if personaID == "" {
		personaID = DefaultPersona().ID
	}
	persona, exists := s.Personas[personaID]
	if !exists || persona.Status != PersonaActive {
		return reject(s, c, "persona_unknown")
	}
	if s.Conversations == nil {
		s.Conversations = map[string]Conversation{}
	}
	if s.Messages == nil {
		s.Messages = map[string][]Message{}
	}
	s.Conversations[c.ConversationID] = Conversation{
		ID: c.ConversationID, OwnerID: s.OwnerID, SurfaceID: c.SurfaceID,
		PersonaID: personaID, Status: ConversationActive,
		CreatedAt: c.CreatedAt, UpdatedAt: c.CreatedAt, NextSequence: 1,
	}
	s.Messages[c.ConversationID] = []Message{}
	return accepted(s, c, OutcomeApplied, c.ConversationID)
}

func sendConversation(s State, c Command) Transition {
	if reason := conversationContext(s, c, false); reason != "" {
		return reject(s, c, reason)
	}
	if c.CreatedAt <= 0 || c.ReconcileBy <= c.CreatedAt {
		return reject(s, c, "invalid_deadline")
	}
	conversation, exists := s.Conversations[c.ConversationID]
	if !exists {
		return reject(s, c, "conversation_unknown")
	}
	if conversation.OwnerID != c.ActorID || conversation.Status != ConversationActive {
		return reject(s, c, "conversation_unavailable")
	}
	if c.SurfaceID == "" {
		return reject(s, c, "invalid_identifier")
	}
	if c.SurfaceID != conversation.SurfaceID {
		return reject(s, c, "surface_mismatch")
	}
	if c.TaskID == "" {
		return reject(s, c, "invalid_identifier")
	}
	if !utf8.ValidString(c.Content) {
		return reject(s, c, "invalid_utf8")
	}
	if len(c.Content) > MaxChatMessageBytes {
		return reject(s, c, "message_too_large")
	}
	messages := s.Messages[c.ConversationID]
	if conversation.NextSequence == 0 || conversation.NextSequence != uint64(len(messages)+1) {
		return reject(s, c, "sequence_conflict")
	}
	if conversation.SizeBytes+len(c.Content) > MaxConversationBytes {
		return reject(s, c, "conversation_too_large")
	}
	if _, exists := s.Tasks[c.TaskID]; exists {
		return reject(s, c, "task_already_exists")
	}
	if s.HostCreates == nil {
		s.HostCreates = map[string]HostCreate{}
	}
	if s.Tasks == nil {
		s.Tasks = map[string]Task{}
	}
	runtimeKey := c.RuntimeIdempotencyKey
	if runtimeKey == "" {
		runtimeKey = "conversation:" + c.TaskID
	}
	for _, intent := range s.HostCreates {
		if intent.IdempotencyKey == runtimeKey {
			return reject(s, c, "runtime_idempotency_collision")
		}
	}
	profileDigest := c.RuntimeProfileDigest
	if profileDigest == "" {
		profileDigest = DefaultRuntimeProfile().Digest
	}
	message := Message{
		ID: userMessageID(c.TaskID), ConversationID: c.ConversationID,
		Sequence: conversation.NextSequence, Role: MessageUser,
		AuthorID: c.ActorID, SurfaceID: conversation.SurfaceID,
		Content: c.Content, ContentDigest: DigestText(c.Content),
		CreatedAt: c.CreatedAt, TaskID: c.TaskID,
	}
	if s.Messages == nil {
		s.Messages = map[string][]Message{}
	}
	s.Messages[c.ConversationID] = append(messages, message)
	conversation.NextSequence++
	conversation.SizeBytes += len(c.Content)
	conversation.UpdatedAt = c.CreatedAt
	s.Conversations[c.ConversationID] = conversation
	s.Tasks[c.TaskID] = Task{
		ID: c.TaskID, CommandID: commandID(c), OriginNodeID: c.ActorID,
		TargetNodeID: s.HostID, CapabilityID: "conversation.chat/respond",
		Action: "respond", ActionFingerprint: DigestText(c.Content),
		HostEpoch: s.Epoch, Status: OutcomeQueued,
	}
	s.HostCreates[c.TaskID] = HostCreate{
		TaskID: c.TaskID, IdempotencyKey: runtimeKey,
		RuntimeProfileDigest: profileDigest, HostEpoch: s.Epoch,
		CreatedAt: c.CreatedAt, ReconcileBy: c.ReconcileBy,
	}
	return accepted(s, c, OutcomeQueued, c.TaskID)
}

func completeConversation(s State, c Command) Transition {
	if reason := conversationContext(s, c, false); reason != "" {
		return reject(s, c, reason)
	}
	if c.TaskID == "" {
		return reject(s, c, "invalid_identifier")
	}
	task, exists := s.Tasks[c.TaskID]
	if !exists {
		return reject(s, c, "task_unknown")
	}
	if c.ActorID != s.HostID {
		return reject(s, c, "unauthorized_actor")
	}
	if c.Outcome != OutcomeCompleted && c.Outcome != OutcomeFailed && c.Outcome != OutcomeUnknown {
		return reject(s, c, "invalid_completion_outcome")
	}
	if !utf8.ValidString(c.Output) {
		return reject(s, c, "invalid_utf8")
	}
	if len(c.Output) > MaxChatMessageBytes {
		return reject(s, c, "message_too_large")
	}
	conversationID, ok := conversationForTask(s, c.TaskID)
	if !ok {
		return reject(s, c, "conversation_unknown")
	}
	if task.CapabilityID != "conversation.chat/respond" {
		return reject(s, c, "conversation_capability_mismatch")
	}
	// Terminal observations are deliberately monotonic. A late success cannot
	// turn an uncertain or failed task into a success, and vice versa.
	if task.Status == OutcomeCompleted || task.Status == OutcomeFailed || task.Status == OutcomeUnknown {
		return accepted(s, c, task.Status, c.TaskID)
	}
	if task.Status != OutcomeQueued && task.Status != OutcomeCreating && task.Status != OutcomeDispatched && task.Status != OutcomeRunning {
		return reject(s, c, "invalid_task_state")
	}
	task.Status = c.Outcome
	if c.Outcome == OutcomeUnknown {
		task.TerminalReason = "unknown_outcome"
	} else if c.Outcome == OutcomeFailed {
		task.TerminalReason = "runtime_failed"
	}
	if c.Outcome == OutcomeCompleted {
		task.Output = c.Output
		task.OutputTruncated = c.OutputTruncated
		conversation := s.Conversations[conversationID]
		messages := s.Messages[conversationID]
		if conversation.NextSequence == 0 || conversation.NextSequence != uint64(len(messages)+1) {
			return reject(s, c, "sequence_conflict")
		}
		if conversation.SizeBytes+len(c.Output) > MaxConversationBytes {
			return reject(s, c, "conversation_too_large")
		}
		assistantID := AssistantMessageID(c.TaskID)
		for _, message := range messages {
			if message.ID == assistantID {
				s.Tasks[c.TaskID] = task
				return accepted(s, c, c.Outcome, c.TaskID)
			}
		}
		s.Messages[conversationID] = append(messages, Message{
			ID: assistantID, ConversationID: conversationID,
			Sequence: conversation.NextSequence, Role: MessageAssistant,
			AuthorID: s.HostID, SurfaceID: conversation.SurfaceID,
			Content: c.Output, ContentDigest: DigestText(c.Output),
			CreatedAt: c.CreatedAt, TaskID: c.TaskID,
		})
		conversation.NextSequence++
		conversation.SizeBytes += len(c.Output)
		conversation.UpdatedAt = c.CreatedAt
		s.Conversations[conversationID] = conversation
	}
	s.Tasks[c.TaskID] = task
	return accepted(s, c, c.Outcome, c.TaskID)
}

func conversationContext(s State, c Command, ownerActor bool) string {
	if c.SpaceID != s.SpaceID {
		return "invalid_space"
	}
	if c.HostID != s.HostID || c.Epoch == 0 || c.Epoch != s.Epoch {
		return "stale_host_epoch"
	}
	if ownerActor {
		if c.ActorID != s.OwnerID {
			return "unauthorized_actor"
		}
	} else if c.ActorID == "" {
		return "unauthorized_actor"
	}
	return ""
}

func conversationForTask(s State, taskID string) (string, bool) {
	for conversationID, messages := range s.Messages {
		for _, message := range messages {
			if message.TaskID == taskID && message.Role == MessageUser {
				return conversationID, true
			}
		}
	}
	return "", false
}
