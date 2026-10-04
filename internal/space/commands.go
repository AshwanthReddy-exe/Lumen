package space

// ConversationCommand is an alias for the canonical Space command envelope.
// Conversation commands use the same idempotency and audit path as every
// other state-changing Space command.
type ConversationCommand = Command

const (
	CommandConversationCreate   = CommandCreateConversation
	CommandConversationSend     = CommandSendConversation
	CommandConversationComplete = CommandCompleteConversation
)
