package conversation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.logger = logger
		}
	}
}

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
	logger    *slog.Logger
	now       func() time.Time
}

func NewService(state StateStore, runtime hermes.Adapter, certifier Certifier, options ...Option) *Service {
	s := &Service{state: state, runtime: runtime, certifier: certifier, logger: slog.Default(), now: time.Now}
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

type MemoryRequest struct {
	RequestID, MemoryID, Text string
	CreatedAt                 int64
}

type Memory struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	AcceptedAt int64  `json:"accepted_at"`
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
		s.logger.Info("lumen_conversation", "event", "intent_replayed", "outcome", task.Status)
		return queued, nil
	}
	s.logger.Info("lumen_conversation", "event", "intent_persisted", "input_bytes", len(req.Input))
	state, err = s.state.Read()
	if err != nil {
		return queued, err
	}
	profile, ok := state.RuntimeProfiles[space.DefaultRuntimeProfile().ID]
	if !ok {
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	if s.certifier == nil {
		s.logger.Warn("lumen_conversation", "event", "runtime_blocked", "reason", "runtime_profile_unverified")
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	sessionID := "lumen-session:" + req.ConversationID
	reserved, reserveErr := s.apply(space.Command{Type: space.CommandReserveRuntimeSession, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "reserve-session:" + req.RequestID, ConversationID: req.ConversationID, HermesSessionID: sessionID, RuntimeProfileDigest: profile.Digest})
	if reserveErr != nil || reserved.Rejection != "" {
		return s.fail(queued, state, taskID, errOrRejection(reserveErr, reserved.Rejection))
	}
	state, err = s.state.Read()
	if err != nil {
		return queued, err
	}
	s.logger.Info("lumen_conversation", "event", "certification_started", "profile", profile.ID)
	cert, certErr := s.certifier.Certify(ctx, state, profile)
	if certErr != nil || !certificationMatches(cert, profile, s.now()) || space.RuntimeCertificationInvalidated(state, cert) || !s.endpointMatches(ctx, cert) {
		s.logger.Warn("lumen_conversation", "event", "runtime_blocked", "reason", "runtime_profile_unverified")
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
	s.logger.Info("lumen_conversation", "event", "context_projected", "history_messages", len(projection.Messages), "context_bytes", len(projection.Context), "instructions_bytes", len(projection.Instructions))
	fresh, freshErr := s.certifier.Certify(ctx, state, profile)
	if freshErr != nil || !reflect.DeepEqual(fresh, cert) || !certificationMatches(fresh, profile, s.now()) || !s.endpointMatches(ctx, fresh) {
		s.logger.Warn("lumen_conversation", "event", "runtime_blocked", "reason", "runtime_profile_unverified")
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	bound, err := s.apply(space.Command{Type: space.CommandBindRuntimeSession, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "session:" + req.RequestID, ConversationID: req.ConversationID, RuntimeIdentity: cert.RuntimeIdentity, HermesSessionID: sessionID, RuntimeProfileDigest: profile.Digest})
	if err != nil || bound.Rejection != "" {
		return queued, errOrRejection(err, bound.Rejection)
	}
	if s.runtime == nil {
		s.logger.Warn("lumen_conversation", "event", "runtime_blocked", "reason", "runtime_profile_unverified")
		return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
	}
	s.logger.Info("lumen_conversation", "event", "hermes_request_started", "input_bytes", len(projection.Input), "history_messages", len(projection.Messages), "context_bytes", len(projection.Context), "profile", profile.ID)
	run, createErr := s.runtime.CreateRun(ctx, hermes.CreateRunRequest{Input: projection.Input, Instructions: projection.Instructions, SessionID: sessionID, ConversationHistory: mustJSON(projection.Messages)}, req.RequestID)
	if createErr != nil || run.RunID == "" {
		if createErr == nil {
			createErr = errors.New("Hermes create returned no run ID")
		}
		if errors.Is(createErr, hermes.ErrCreateRejected) {
			s.logger.Warn("lumen_conversation", "event", "hermes_request_rejected", "error_class", "create_rejected")
			return s.fail(queued, state, taskID, createErr)
		}
		s.logger.Warn("lumen_conversation", "event", "hermes_request_uncertain", "error_class", "create_ambiguous")
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", createErr)
	}
	s.logger.Info("lumen_conversation", "event", "hermes_request_accepted")
	dispatched, err := s.apply(space.Command{Type: space.CommandDispatchHostRun, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "dispatch:" + req.RequestID, TaskID: taskID, RuntimeRunID: run.RunID, RuntimeIdempotencyKey: req.RequestID, RuntimeProfileDigest: profile.Digest, CertificationID: cert.ID, EndpointIdentity: cert.EndpointIdentity, DispatchedAt: created, ReconcileBy: by})
	if err != nil || dispatched.Rejection != "" {
		return queued, errOrRejection(err, dispatched.Rejection)
	}
	return s.reconcileRun(ctx, queued, state, taskID, run, cert, profile, by)
}

func (s *Service) reconcileRun(ctx context.Context, queued space.Transition, state space.State, taskID string, run hermes.Run, cert space.RuntimeCertification, profile space.RuntimeProfile, by int64) (space.Transition, error) {
	if s.runtime == nil {
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("runtime unavailable for conversation reconciliation"))
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
	events, eventsErr := s.runtime.Events(reconcileCtx, run.RunID)
	s.logger.Info("lumen_conversation", "event", "hermes_evidence_received", "status_class", terminalOutcome(status.Status), "events", len(events))
	for _, event := range events {
		if !allowedChatEvent(event) {
			invalidated, invalidateErr := s.apply(space.Command{Type: space.CommandInvalidateRuntimeCertification, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: space.RuntimeCertificationInvalidationID(cert), CertificationID: cert.ID, RuntimeIdentity: cert.RuntimeIdentity, RuntimeProfileDigest: profile.Digest, TaskID: taskID, RuntimeRunID: run.RunID})
			_, _ = s.runtime.Stop(ctx, run.RunID)
			if invalidateErr != nil || invalidated.Rejection != "" {
				return s.fail(queued, state, taskID, errors.Join(ErrRuntimeProfileUnverified, errOrRejection(invalidateErr, invalidated.Rejection)))
			}
			return s.fail(queued, state, taskID, ErrRuntimeProfileUnverified)
		}
	}
	terminalEvent := false
	terminal := terminalOutcome(status.Status)
	for _, event := range events {
		if eventStatus, output, ok := conversationEvent(event); ok {
			if outcome := terminalOutcome(eventStatus); terminal != "" && outcome != terminal {
				return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("conflicting runtime terminal evidence"))
			} else {
				terminal = outcome
			}
			if status.Output != "" && output != "" && status.Output != output {
				return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("conflicting runtime output evidence"))
			}
			status.Status = eventStatus
			if output != "" {
				status.Output = output
			}
			terminalEvent = true
		}
	}
	if eventsErr != nil && !errors.Is(eventsErr, hermes.ErrEventStreamDisconnected) {
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", eventsErr)
	}
	if errors.Is(eventsErr, hermes.ErrEventStreamDisconnected) && !terminalEvent {
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("runtime event stream ended without terminal evidence"))
	}
	if s.now().Unix() >= by {
		return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("conversation reconciliation deadline reached"))
	}
	switch status.Status {
	case "completed", "succeeded", "success":
		conversation, ok := state.Conversations[conversationForTask(state, taskID)]
		if !ok || !utf8.ValidString(status.Output) || len(status.Output) > space.MaxChatMessageBytes || conversation.SizeBytes+len(status.Output) > space.MaxConversationBytes {
			return s.complete(queued, state, taskID, space.OutcomeUnknown, "", errors.New("runtime output exceeds conversation bounds"))
		}
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
	if len(event.Data) > 0 {
		if data := bytes.TrimSpace(event.Data); len(data) == 0 || data[0] != '{' {
			return false
		}
		decoder := json.NewDecoder(bytes.NewReader(event.Data))
		start, err := decoder.Token()
		if err != nil || start != json.Delim('{') {
			return false
		}
		seen := map[string]bool{}
		expected := canonicalChatStatus(event.Type)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return false
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return false
			}
			seen[name] = true
			var value string
			if decoder.Decode(&value) != nil {
				return false
			}
			switch name {
			case "status", "event", "type":
				if canonicalChatStatus(value) != expected {
					return false
				}
			case "output":
			default:
				return false
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return false
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			return false
		}
	}
	return true
}

func terminalOutcome(status string) space.Outcome {
	status = strings.TrimPrefix(strings.ToLower(status), "run.")
	switch status {
	case "completed", "succeeded", "success":
		return space.OutcomeCompleted
	case "failed", "error", "cancelled", "canceled":
		return space.OutcomeFailed
	default:
		return ""
	}
}

func allowedChatStatus(value string) bool {
	return canonicalChatStatus(value) != ""
}

func canonicalChatStatus(value string) string {
	switch strings.TrimPrefix(strings.ToLower(value), "run.") {
	case "created", "started", "running", "progress":
		return strings.TrimPrefix(strings.ToLower(value), "run.")
	case "completed", "succeeded", "success":
		return "completed"
	case "failed", "error":
		return "failed"
	case "cancelled", "canceled":
		return "cancelled"
	default:
		return ""
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

func (s *Service) SaveMemory(_ context.Context, req MemoryRequest) (space.Transition, error) {
	state, err := s.state.Read()
	if err != nil {
		return space.Transition{}, err
	}
	created := req.CreatedAt
	if created <= 0 {
		created = s.now().Unix()
	}
	return s.apply(space.Command{Type: space.CommandSaveMemory, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: req.RequestID, MemoryID: req.MemoryID, Content: req.Text, CreatedAt: created})
}

func (s *Service) DeleteMemory(_ context.Context, req MemoryRequest) (space.Transition, error) {
	state, err := s.state.Read()
	if err != nil {
		return space.Transition{}, err
	}
	return s.apply(space.Command{Type: space.CommandDeleteMemory, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: req.RequestID, MemoryID: req.MemoryID})
}

func (s *Service) ListMemory(_ context.Context) ([]Memory, error) {
	state, err := s.state.Read()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(state.ContextRecords))
	for id := range state.ContextRecords {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]Memory, 0)
	for _, id := range ids {
		record := state.ContextRecords[id]
		if record.Namespace != "user.memory/v1" || record.SchemaVersion != 1 || record.Provenance != "owner" || record.Classification != "private" || record.AcceptedAt <= 0 || record.RetentionUntil != 0 && record.RetentionUntil <= s.now().Unix() || record.Digest != space.DigestText(string(record.Payload)) {
			continue
		}
		var payload map[string]string
		if json.Unmarshal(record.Payload, &payload) != nil || len(payload) != 1 || payload["text"] == "" {
			continue
		}
		result = append(result, Memory{ID: id, Text: payload["text"], AcceptedAt: record.AcceptedAt})
	}
	return result, nil
}

func (s *Service) apply(command space.Command) (space.Transition, error) {
	return s.state.Update(func(state space.State) space.Transition { return space.Apply(state, command) })
}

func (s *Service) fail(queued space.Transition, state space.State, taskID string, cause error) (space.Transition, error) {
	return s.complete(queued, state, taskID, space.OutcomeFailed, "", cause)
}

func (s *Service) complete(queued space.Transition, state space.State, taskID string, outcome space.Outcome, output string, cause error) (space.Transition, error) {
	tr, err := s.apply(space.Command{Type: space.CommandCompleteConversation, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: fmt.Sprintf("complete:%s:%s", taskID, outcome), TaskID: taskID, Outcome: outcome, Output: output, CreatedAt: s.now().Unix(), TerminalReason: func() string {
		if cause != nil && errors.Is(cause, ErrRuntimeProfileUnverified) {
			return "runtime_profile_unverified"
		}
		return ""
	}()})
	if err != nil {
		return tr, errors.Join(cause, err)
	}
	if tr.Rejection != "" {
		s.logger.Error("lumen_conversation", "event", "terminal_persistence_failed", "reason", "state_rejected")
		return tr, errors.Join(cause, errors.New(tr.Rejection))
	}
	s.logger.Info("lumen_conversation", "event", "terminal_persisted", "outcome", outcome, "output_bytes", len(output), "reason", func() string {
		if errors.Is(cause, ErrRuntimeProfileUnverified) {
			return "runtime_profile_unverified"
		}
		if cause != nil {
			return "runtime_error"
		}
		return ""
	}())
	return tr, cause
}

func publicOverrides(instructions, session, provider, model, profile string) bool {
	return instructions != "" || session != "" || provider != "" || model != "" || profile != ""
}

func certificationMatches(cert space.RuntimeCertification, profile space.RuntimeProfile, now time.Time) bool {
	limits := cert.Limits
	return cert.ID != "" && cert.RuntimeIdentity != "" && cert.EndpointIdentity != "" && cert.ArtifactDigest != "" && cert.ProcessIdentity != "" && cert.HermesVersion != "" && cert.PluginIdentity != "" && cert.PluginCommit != "" && cert.ConfigDigest != "" && cert.Evidence != "" && profile.Digest == space.RuntimeProfileDigest(profile) && len(profile.AllowedFeatureSet) == 0 && !profile.MemoryRead && !profile.MemoryWrite && cert.ProfileDigest == profile.Digest && len(cert.EffectiveToolsets) == 0 && !cert.MemoryRead && !cert.MemoryWrite && cert.ExpiresAt > now.Unix() && limits.Version == 1 && limits.MaxTurns == profile.MaxTurns && limits.MaxMessages == profile.MaxMessages && limits.MaxContextBytes == profile.MaxContextBytes && limits.MaxInputTokens == profile.MaxInputTokens && limits.MaxOutputTokens == profile.MaxOutputTokens && limits.DeadlineSeconds == profile.DeadlineSeconds
}

func (s *Service) endpointMatches(ctx context.Context, cert space.RuntimeCertification) bool {
	observer, ok := s.runtime.(interface {
		VerifiedConversationEndpointIdentity(context.Context) (string, error)
	})
	if !ok {
		return false
	}
	identity, err := observer.VerifiedConversationEndpointIdentity(ctx)
	return err == nil && identity == cert.EndpointIdentity
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
		status, output := strings.TrimPrefix(strings.ToLower(payload.Status), "run."), payload.Output
		return status, output, isTerminalStatus(status)
	}
	return status, "", isTerminalStatus(status)
}
