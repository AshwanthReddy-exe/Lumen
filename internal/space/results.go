package space

// AssistantMessageID is stable for a task and therefore safe to derive again
// when completion observations are duplicated or reordered.
func AssistantMessageID(taskID string) string { return "assistant:" + taskID }

func userMessageID(taskID string) string { return "user:" + taskID }
