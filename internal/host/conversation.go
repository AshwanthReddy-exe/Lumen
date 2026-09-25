package host

import (
	"context"
	"errors"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/conversation"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

func (s *Service) conversationAPI() (*conversation.Service, error) {
	s.conversationMu.Lock()
	defer s.conversationMu.Unlock()
	if s.conversation != nil {
		return s.conversation, nil
	}
	if _, err := s.state.MigrateState(); err != nil {
		return nil, err
	}
	s.conversation = conversation.NewService(s.state, s.executor.options.conversationRuntime, s.executor.options.certifier)
	return s.conversation, nil
}

func (s *Service) ConversationCreate(ctx context.Context, req conversation.CreateRequest) (space.Transition, error) {
	api, err := s.conversationAPI()
	if err != nil {
		return space.Transition{}, err
	}
	return api.Create(ctx, req)
}

func (s *Service) ConversationSend(ctx context.Context, req conversation.SendRequest) (space.Transition, error) {
	api, err := s.conversationAPI()
	if err != nil {
		return space.Transition{}, err
	}
	return api.Send(ctx, req)
}

func (s *Service) ConversationShow(ctx context.Context, conversationID string) (conversation.View, error) {
	api, err := s.conversationAPI()
	if err != nil {
		return conversation.View{}, err
	}
	return api.Show(ctx, conversationID)
}

func (s *Service) PreferenceSet(ctx context.Context, req conversation.PreferenceRequest) (space.Transition, error) {
	api, err := s.conversationAPI()
	if err != nil {
		return space.Transition{}, err
	}
	return api.SetPreference(ctx, req)
}

func (s *Service) MemorySave(ctx context.Context, req conversation.MemoryRequest) (space.Transition, error) {
	api, err := s.conversationAPI()
	if err != nil {
		return space.Transition{}, err
	}
	return api.SaveMemory(ctx, req)
}

func (s *Service) MemoryDelete(ctx context.Context, req conversation.MemoryRequest) (space.Transition, error) {
	api, err := s.conversationAPI()
	if err != nil {
		return space.Transition{}, err
	}
	return api.DeleteMemory(ctx, req)
}

func (s *Service) MemoryList(ctx context.Context) ([]conversation.Memory, error) {
	api, err := s.conversationAPI()
	if err != nil {
		return nil, err
	}
	return api.ListMemory(ctx)
}

func conversationError(err error, tr space.Transition) error {
	if err != nil {
		return err
	}
	if tr.Rejection != "" {
		return errors.New(tr.Rejection)
	}
	return nil
}

func (s *Service) handleConversationCreate(ctx context.Context, args map[string]string) control.Response {
	if !validConversationArguments("conversation create", args) {
		return control.Response{Error: "invalid conversation arguments"}
	}
	tr, err := s.ConversationCreate(ctx, conversation.CreateRequest{RequestID: args["request_id"], ConversationID: args["conversation_id"], SurfaceID: args["surface_id"]})
	return responseForTransition(tr, conversationError(err, tr))
}

func (s *Service) handleConversationSend(ctx context.Context, args map[string]string) control.Response {
	if !validConversationArguments("conversation send", args) {
		return control.Response{Error: "invalid conversation arguments"}
	}
	tr, err := s.ConversationSend(ctx, conversation.SendRequest{RequestID: args["request_id"], ConversationID: args["conversation_id"], SurfaceID: args["surface_id"], Input: args["input"], TaskID: args["task_id"]})
	return responseForTransition(tr, conversationError(err, tr))
}

func (s *Service) handleConversationShow(ctx context.Context, args map[string]string) control.Response {
	if !validConversationArguments("conversation show", args) {
		return control.Response{Error: "invalid conversation arguments"}
	}
	view, err := s.ConversationShow(ctx, args["conversation_id"])
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	return control.Response{OK: true, Data: view}
}

func (s *Service) handlePreferenceSet(ctx context.Context, args map[string]string) control.Response {
	if !validConversationArguments("preference set", args) {
		return control.Response{Error: "invalid preference arguments"}
	}
	tr, err := s.PreferenceSet(ctx, conversation.PreferenceRequest{RequestID: args["request_id"], Name: args["name"], Value: args["value"]})
	return responseForTransition(tr, conversationError(err, tr))
}

func (s *Service) handleMemorySave(ctx context.Context, args map[string]string) control.Response {
	if !validConversationArguments("memory save", args) {
		return control.Response{Error: "invalid memory arguments"}
	}
	tr, err := s.MemorySave(ctx, conversation.MemoryRequest{RequestID: args["request_id"], MemoryID: args["memory_id"], Text: args["text"]})
	return responseForTransition(tr, conversationError(err, tr))
}

func (s *Service) handleMemoryDelete(ctx context.Context, args map[string]string) control.Response {
	if !validConversationArguments("memory delete", args) {
		return control.Response{Error: "invalid memory arguments"}
	}
	tr, err := s.MemoryDelete(ctx, conversation.MemoryRequest{RequestID: args["request_id"], MemoryID: args["memory_id"]})
	return responseForTransition(tr, conversationError(err, tr))
}

func (s *Service) handleMemoryList(ctx context.Context, args map[string]string) control.Response {
	if !validConversationArguments("memory list", args) {
		return control.Response{Error: "invalid memory arguments"}
	}
	items, err := s.MemoryList(ctx)
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	return control.Response{OK: true, Data: items}
}

func validConversationArguments(command string, args map[string]string) bool {
	allowed := map[string]map[string]bool{
		"conversation create": {"request_id": true, "conversation_id": true, "surface_id": true},
		"conversation send":   {"request_id": true, "conversation_id": true, "surface_id": true, "input": true, "task_id": true},
		"conversation show":   {"conversation_id": true},
		"preference set":      {"request_id": true, "name": true, "value": true},
		"memory save":         {"request_id": true, "memory_id": true, "text": true},
		"memory list":         {},
		"memory delete":       {"request_id": true, "memory_id": true},
	}
	set, ok := allowed[command]
	if !ok {
		return false
	}
	for key := range args {
		if !set[key] {
			return false
		}
	}
	for _, key := range map[string][]string{"conversation create": {"request_id", "conversation_id", "surface_id"}, "conversation send": {"request_id", "conversation_id", "surface_id", "input"}, "conversation show": {"conversation_id"}, "preference set": {"request_id", "name", "value"}, "memory save": {"request_id", "memory_id", "text"}, "memory list": {}, "memory delete": {"request_id", "memory_id"}}[command] {
		if args[key] == "" {
			return false
		}
	}
	return true
}
