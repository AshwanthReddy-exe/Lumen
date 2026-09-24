package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

var (
	ErrPublicRuntimeOverride    = errors.New("public conversation runtime override rejected")
	ErrRuntimeProfileUnverified = errors.New("runtime profile unverified")
)

type StateStore interface {
	Read() (space.State, error)
	Update(func(space.State) space.Transition) (space.Transition, error)
}

type Certifier interface {
	Certify(context.Context, space.State, space.RuntimeProfile) (space.RuntimeCertification, error)
}

type Option func(*Service)

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

type Service struct {
	state     StateStore
	runtime   hermes.Adapter
	certifier Certifier
	now       func() time.Time
}

func NewService(state StateStore, runtime hermes.Adapter, certifier Certifier, options ...Option) *Service {
	s := &Service{state: state, runtime: runtime, certifier: certifier, now: time.Now}
	for _, option := range options {
		if option != nil {
			option(s)
		}
	}
	return s
}

type CreateRequest struct {
	RequestID, ConversationID, SurfaceID                           string
	Instructions, SessionID, Provider, Model, RuntimeProfileDigest string
	CreatedAt                                                      int64
}

type SendRequest struct {
	RequestID, ConversationID, SurfaceID, Input, TaskID            string
	Instructions, SessionID, Provider, Model, RuntimeProfileDigest string
	CreatedAt, ReconcileBy                                         int64
}

type PreferenceRequest struct {
	RequestID, Name, Value string
	CreatedAt              int64
}

type View struct {
	Conversation space.Conversation `json:"conversation"`
	Messages     []space.Message    `json:"messages"`
}

func (s *Service) Create(_ context.Context, req CreateRequest) (space.Transition, error) {
	if publicOverrides(req.Instructions, req.SessionID, req.Provider, req.Model, req.RuntimeProfileDigest) {
		return space.Transition{}, ErrPublicRuntimeOverride
	}
	state, err := s.state.Read()
	if err != nil {
		return space.Transition{}, err
	}
	created := req.CreatedAt
	if created <= 0 {
		created = s.now().Unix()
	}
	if _, ok := state.Surfaces[req.SurfaceID]; !ok {
		registered, registerErr := s.apply(space.Command{Type: space.CommandRegisterSurface, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: "surface:" + req.SurfaceID, SurfaceID: req.SurfaceID})
		if registerErr != nil || registered.Rejection != "" {
			return registered, errOrRejection(registerErr, registered.Rejection)
		}
	}
	return s.apply(space.Command{Type: space.CommandCreateConversation, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: req.RequestID, ConversationID: req.ConversationID, SurfaceID: req.SurfaceID, CreatedAt: created})
}

func (s *Service) Send(ctx context.Context, req SendRequest) (space.Transition, error) {
	if publicOverrides(req.Instructions, req.SessionID, req.Provider, req.Model, req.RuntimeProfileDigest) {
		return space.Transition{}, ErrPublicRuntimeOverride
	}
	state, err := s.state.Read()
	if err != nil {
		return space.Transition{}, err
	}
	created := req.CreatedAt
	if created <= 0 {
		created = s.now().Unix()
	}
	by := req.ReconcileBy
	if by <= created {
		by = created + 30
	}
	taskID := req.TaskID
	if taskID == "" {
		taskID = "conversation:" + req.RequestID
	}
	queued, err := s.apply(space.Command{Type: space.CommandSendConversation, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: req.RequestID, ConversationID: req.ConversationID, SurfaceID: req.SurfaceID, TaskID: taskID, Content: req.Input, RuntimeIdempotencyKey: req.RequestID, CreatedAt: created, ReconcileBy: by})
	if err != nil || queued.Rejection != "" {
		return queued, errOrRejection(err, queued.Rejection)
	}
	if queued.Replayed {
		current, readErr := s.state.Read()
		if readErr != nil {
			return queued, readErr
		}
		task, ok := current.Tasks[queued.Receipt.SubjectID]
		if !ok {
			return queued, errors.New("replayed conversation task is missing")
		}
		queued.Receipt.Outcome = task.Status
		return queued, nil
	}
	state, err = s.state.Read()
	if err != nil {
		return queued, err
	}
	profile, ok := state.RuntimeProfiles[space.DefaultRuntimeProfile().ID]
	if !ok {
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	if s.certifier == nil {
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	cert, certErr := s.certifier.Certify(ctx, state, profile)
	if certErr != nil || !certificationMatches(cert, profile, s.now()) {
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	persona, err := CompilePersona(state, req.ConversationID)
	if err != nil {
		return s.fail(queued, state, taskID, err)
	}
	projection, err := Project(state, req.ConversationID, persona, profile, s.now().Unix())
	if err != nil {
		return s.fail(queued, state, taskID, err)
	}
	sessionID := "lumen-session:" + req.ConversationID
	bound, err := s.apply(space.Command{Type: space.CommandBindRuntimeSession, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "session:" + req.RequestID, ConversationID: req.ConversationID, RuntimeIdentity: cert.RuntimeIdentity, HermesSessionID: sessionID, RuntimeProfileDigest: profile.Digest})
	if err != nil || bound.Rejection != "" {
		return queued, errOrRejection(err, bound.Rejection)
	}
	if s.runtime == nil {
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	run, createErr := s.runtime.CreateRun(ctx, hermes.CreateRunRequest{Input: projection.Input, Instructions: projection.Instructions, SessionID: sessionID, ConversationHistory: mustJSON(projection.Messages)}, req.RequestID)
	if createErr != nil || run.RunID == "" {
		if createErr == nil {
			createErr = errors.New("Hermes create returned no run ID")
		}
		if errors.Is(createErr, hermes.ErrCreateRejected) {
			return s.fail(queued, state, taskID, createErr)
		}
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", createErr)
	}
	dispatched, err := s.apply(space.Command{Type: space.CommandDispatchHostRun, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "dispatch:" + req.RequestID, TaskID: taskID, RuntimeRunID: run.RunID, RuntimeIdempotencyKey: req.RequestID, RuntimeProfileDigest: profile.Digest, DispatchedAt: created, ReconcileBy: by})
	if err != nil || dispatched.Rejection != "" {
		return queued, errOrRejection(err, dispatched.Rejection)
	}
	reconcileCtx, cancel := context.WithTimeout(ctx, time.Unix(by, 0).Sub(s.now()))
	defer cancel()
	if s.now().Unix() >= by {
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("conversation reconciliation deadline reached"))
	}
	status, statusErr := s.runtime.RunStatus(reconcileCtx, run.RunID)
	if statusErr != nil || status.RunID != run.RunID {
		if statusErr == nil {
			statusErr = errors.New("Hermes status run mismatch")
		}
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", statusErr)
	}
	if !isTerminalStatus(status.Status) {
		events, eventsErr := s.runtime.Events(reconcileCtx, run.RunID)
		if eventsErr == nil {
			for _, event := range events {
				if !allowedChatEvent(event) {
					_, _ = s.runtime.Stop(ctx, run.RunID)
					return s.fail(queued, state, taskID, errors.New("runtime emitted an uncertified event"))
				}
				if eventStatus, output, ok := conversationEvent(event); ok {
					status.Status, status.Output = eventStatus, output
					if isTerminalStatus(status.Status) {
						break
					}
				}
			}
		}
	}
	if s.now().Unix() >= by {
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("conversation reconciliation deadline reached"))
	}
	switch status.Status {
	case "completed", "succeeded", "success":
		return s.complete(queued, state, taskID, space.OutcomeCompleted, status.Output, nil)
	case "failed", "error", "cancelled", "canceled":
		return s.complete(queued, state, taskID, space.OutcomeFailed, "", nil)
	default:
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("Hermes completion remains uncertain"))
	}
}

func allowedChatEvent(event hermes.Event) bool {
	if !allowedChatStatus(event.Type) {
		return false
	}
	var payload struct {
		Status string `json:"status"`
		Event  string `json:"event"`
		Type   string `json:"type"`
	}
	if len(event.Data) > 0 && json.Unmarshal(event.Data, &payload) != nil {
		return false
	}
	for _, value := range []string{payload.Status, payload.Event, payload.Type} {
		if value != "" && !allowedChatStatus(value) {
			return false
		}
	}
	return true
}

func allowedChatStatus(value string) bool {
	switch strings.ToLower(value) {
	case "created", "started", "running", "progress", "completed", "succeeded", "success", "failed", "error", "cancelled", "canceled", "run.created", "run.started", "run.running", "run.progress", "run.completed", "run.failed", "run.cancelled":
		return true
	default:
		return false
	}
}

func (s *Service) Show(_ context.Context, conversationID string) (View, error) {
	state, err := s.state.Read()
	if err != nil {
		return View{}, err
	}
	conversation, ok := state.Conversations[conversationID]
	if !ok {
		return View{}, errors.New("conversation_unknown")
	}
	messages := append([]space.Message(nil), state.Messages[conversationID]...)
	return View{Conversation: conversation, Messages: messages}, nil
}

func (s *Service) SetPreference(_ context.Context, req PreferenceRequest) (space.Transition, error) {
	state, err := s.state.Read()
	if err != nil {
		return space.Transition{}, err
	}
	created := req.CreatedAt
	if created <= 0 {
		created = s.now().Unix()
	}
	return s.apply(space.Command{Type: space.CommandSetPreference, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: req.RequestID, PreferenceName: req.Name, Content: req.Value, CreatedAt: created})
}

func (s *Service) apply(command space.Command) (space.Transition, error) {
	return s.state.Update(func(state space.State) space.Transition { return space.Apply(state, command) })
}

func (s *Service) fail(queued space.Transition, state space.State, taskID string, cause error) (space.Transition, error) {
	return s.complete(queued, state, taskID, space.OutcomeFailed, "", cause)
}

func (s *Service) complete(queued space.Transition, state space.State, taskID string, outcome space.Outcome, output string, cause error) (space.Transition, error) {
	tr, err := s.apply(space.Command{Type: space.CommandCompleteConversation, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: fmt.Sprintf("complete:%s:%s", taskID, outcome), TaskID: taskID, Outcome: outcome, Output: output, TerminalReason: func() string {
		if cause != nil && errors.Is(cause, ErrRuntimeProfileUnverified) {
			return "runtime_profile_unverified"
		}
		return ""
	}()})
	if err != nil {
		return tr, errors.Join(cause, err)
	}
	if tr.Rejection != "" {
		return tr, errors.Join(cause, errors.New(tr.Rejection))
	}
	return tr, cause
}

func publicOverrides(instructions, session, provider, model, profile string) bool {
	return instructions != "" || session != "" || provider != "" || model != "" || profile != ""
}

func certificationMatches(cert space.RuntimeCertification, profile space.RuntimeProfile, now time.Time) bool {
	limits := cert.Limits
	return cert.ID != "" && cert.RuntimeIdentity != "" && cert.EndpointIdentity != "" && cert.HermesVersion != "" && cert.PluginIdentity != "" && cert.PluginCommit != "" && cert.ConfigDigest != "" && cert.Evidence != "" && profile.Digest == space.RuntimeProfileDigest(profile) && len(profile.AllowedFeatureSet) == 0 && !profile.MemoryRead && !profile.MemoryWrite && cert.ProfileDigest == profile.Digest && len(cert.EffectiveToolsets) == 0 && !cert.MemoryRead && !cert.MemoryWrite && cert.ExpiresAt > now.Unix() && limits.Version == 1 && limits.MaxTurns == profile.MaxTurns && limits.MaxMessages == profile.MaxMessages && limits.MaxContextBytes == profile.MaxContextBytes && limits.MaxInputTokens == profile.MaxInputTokens && limits.MaxOutputTokens == profile.MaxOutputTokens && limits.MaxTotalTokens == profile.MaxTotalTokens && limits.DeadlineSeconds == profile.DeadlineSeconds
}

func errOrRejection(err error, rejection string) error {
	if err != nil {
		return err
	}
	if rejection != "" {
		return errors.New(rejection)
	}
	return nil
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func isTerminalStatus(status string) bool {
	switch status {
	case "completed", "succeeded", "success", "failed", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func conversationEvent(event hermes.Event) (string, string, bool) {
	status := event.Type
	if dot := strings.LastIndexByte(status, '.'); dot >= 0 {
		status = status[dot+1:]
	}
	status = strings.ToLower(status)
	var payload struct {
		Status string `json:"status"`
		Output string `json:"output"`
	}
	if json.Unmarshal(event.Data, &payload) == nil && payload.Status != "" {
		status, output := strings.ToLower(payload.Status), payload.Output
		return status, output, isTerminalStatus(status)
	}
	return status, "", isTerminalStatus(status)
}
