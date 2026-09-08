package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

var errRuntimeUnavailable = errors.New("Hermes runtime unavailable")

// ErrCreateAmbiguous lets a runtime adapter distinguish a transport outcome
// that may have created a run from a proven pre-dispatch rejection.
var ErrCreateAmbiguous = errors.New("Hermes create outcome is ambiguous")

type executionOptions struct {
	now             func() time.Time
	pollInterval    time.Duration
	reconcileWindow time.Duration
}

type ExecutionOption func(*executionOptions)

func WithClock(now func() time.Time) ExecutionOption {
	return func(o *executionOptions) {
		if now != nil {
			o.now = now
		}
	}
}

func WithExecutionTiming(pollInterval, reconcileWindow time.Duration) ExecutionOption {
	return func(o *executionOptions) {
		if pollInterval > 0 {
			o.pollInterval = pollInterval
		}
		if reconcileWindow > 0 {
			o.reconcileWindow = reconcileWindow
		}
	}
}

type ExecuteRequest struct {
	Submit               space.Command
	Runtime              hermes.CreateRunRequest
	RuntimeProfileDigest string
	ReconcileBy          int64
}

type ApprovalRequest struct {
	Command              space.Command
	Runtime              hermes.CreateRunRequest
	RuntimeProfileDigest string
	ReconcileBy          int64
}

type CancelRequest struct{ Command space.Command }

func (s *Service) handleTaskSubmit(ctx context.Context, args map[string]string) control.Response {
	state, err := s.state.Read()
	if err != nil {
		return control.Response{Error: "state unavailable"}
	}
	requestID, err := parseArgument(args, "request_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	taskID, err := parseArgument(args, "task_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	origin, err := parseArgument(args, "origin_node_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	target, err := parseArgument(args, "target_node_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	capability, err := parseArgument(args, "capability_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	action, err := parseArgument(args, "action")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	fingerprint, err := parseArgument(args, "action_fingerprint")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	by, err := optionalIntArgument(args, "reconcile_by")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	tr, err := s.SubmitTask(ctx, ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: requestID, TaskID: taskID, OriginNodeID: origin, TargetNodeID: target, CapabilityID: capability, Action: action, ActionFingerprint: fingerprint}, Runtime: hermes.CreateRunRequest{Input: args["input"], SessionID: args["session_id"], Instructions: args["instructions"]}, RuntimeProfileDigest: args["runtime_profile_digest"], ReconcileBy: by})
	return responseForTransition(tr, err)
}

func (s *Service) handleTaskShow(args map[string]string) control.Response {
	taskID, err := parseArgument(args, "task_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	state, err := s.state.Read()
	if err != nil {
		return control.Response{Error: "state unavailable"}
	}
	task, ok := state.Tasks[taskID]
	if !ok {
		return control.Response{Error: "task_unknown"}
	}
	return control.Response{OK: true, Data: map[string]any{"task": task, "run": state.HostRuns[taskID]}}
}

func (s *Service) handleTaskCancel(ctx context.Context, args map[string]string) control.Response {
	requestID, err := parseArgument(args, "request_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	taskID, err := parseArgument(args, "task_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	state, err := s.state.Read()
	if err != nil {
		return control.Response{Error: "state unavailable"}
	}
	run := state.HostRuns[taskID]
	tr, callErr := s.CancelTask(ctx, CancelRequest{Command: space.Command{Type: space.CommandRequestHostRunCancellation, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: requestID, TaskID: taskID, RuntimeRunID: run.RuntimeRunID, RuntimeProfileDigest: run.RuntimeProfileDigest, ObservedAt: s.executor.options.now().Unix()}})
	return responseForTransition(tr, callErr)
}

func (s *Service) handleApprovalResolve(ctx context.Context, args map[string]string) control.Response {
	decision, err := parseArgument(args, "decision")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	if decision != "once" && decision != "deny" {
		return control.Response{Error: hermes.ErrInvalidApproval.Error()}
	}
	if decision == "deny" {
		return control.Response{Error: "approval denial is not supported for a task without a runtime"}
	}
	state, err := s.state.Read()
	if err != nil {
		return control.Response{Error: "state unavailable"}
	}
	requestID, err := parseArgument(args, "request_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	taskID, err := parseArgument(args, "task_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	target, err := parseArgument(args, "target_node_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	fingerprint, err := parseArgument(args, "action_fingerprint")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	approvalID, err := parseArgument(args, "approval_id")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	approvedAt, err := parseIntArgument(args, "approved_at")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	expiresAt, err := parseIntArgument(args, "expires_at")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	by, err := optionalIntArgument(args, "reconcile_by")
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	tr, callErr := s.ResolveApproval(ctx, ApprovalRequest{Command: space.Command{Type: space.CommandApprove, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: requestID, TaskID: taskID, TargetNodeID: target, ActionFingerprint: fingerprint, ApprovalID: approvalID, ApprovedAt: approvedAt, ExpiresAt: expiresAt}, Runtime: hermes.CreateRunRequest{Input: args["input"], SessionID: args["session_id"], Instructions: args["instructions"]}, RuntimeProfileDigest: args["runtime_profile_digest"], ReconcileBy: by})
	return responseForTransition(tr, callErr)
}

type executor struct {
	service *Service
	runtime hermes.Adapter
	options executionOptions
	ctx     context.Context
	cancel  context.CancelFunc
	seenMu  sync.Mutex
	seen    map[string]map[string]struct{}
}

func newExecutor(s *Service, runtime hermes.Adapter, options executionOptions) *executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &executor{service: s, runtime: runtime, options: options, ctx: ctx, cancel: cancel, seen: make(map[string]map[string]struct{})}
}

func NewWithRuntime(c Config, runtime hermes.Adapter, options ...ExecutionOption) (*Service, error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(c.DataDir, "initialized")); err != nil {
		return nil, fmt.Errorf("state unavailable: %w", err)
	}
	state, err := store.Open(filepath.Join(c.DataDir, "state.json"), filepath.Join(c.DataDir, "state.key"))
	if err != nil {
		return nil, fmt.Errorf("state unavailable: %w", err)
	}
	if _, err := state.Read(); err != nil {
		_ = state.Close()
		return nil, fmt.Errorf("state unavailable: %w", err)
	}
	opts := executionOptions{now: time.Now, pollInterval: 250 * time.Millisecond, reconcileWindow: 30 * time.Second}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	s := &Service{cfg: c, state: state, ready: make(chan struct{}), stop: make(chan struct{})}
	s.executor = newExecutor(s, runtime, opts)
	s.recoverInFlight()
	return s, nil
}

func (s *Service) SubmitTask(ctx context.Context, req ExecuteRequest) (space.Transition, error) {
	state, readErr := s.state.Read()
	if readErr != nil {
		return space.Transition{}, readErr
	}
	_, replay := state.Commands[req.Submit.RequestID]
	tr, err := s.apply(req.Submit)
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		return tr, err
	}
	if replay {
		return tr, nil
	}
	return s.startOrFail(ctx, req, tr)
}

func (s *Service) ResolveApproval(ctx context.Context, req ApprovalRequest) (space.Transition, error) {
	stateBefore, readErr := s.state.Read()
	if readErr != nil {
		return space.Transition{}, readErr
	}
	_, replay := stateBefore.Commands[req.Command.RequestID]
	tr, err := s.apply(req.Command)
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		return tr, err
	}
	if replay {
		return tr, nil
	}
	state, readErr := s.state.Read()
	if readErr != nil {
		return tr, readErr
	}
	task := state.Tasks[req.Command.TaskID]
	return s.startOrFail(ctx, ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: req.Command.RequestID, TaskID: task.ID, OriginNodeID: task.OriginNodeID, TargetNodeID: task.TargetNodeID, CapabilityID: task.CapabilityID, Action: task.Action, ActionFingerprint: task.ActionFingerprint}, Runtime: req.Runtime, RuntimeProfileDigest: req.RuntimeProfileDigest, ReconcileBy: req.ReconcileBy}, tr)
}

func (s *Service) CancelTask(ctx context.Context, req CancelRequest) (space.Transition, error) {
	tr, err := s.apply(req.Command)
	if err != nil || tr.Rejection != "" {
		return tr, err
	}
	if s.executor == nil || s.executor.runtime == nil {
		return tr, s.reconcileAfterStop(ctx, req.Command.TaskID)
	}
	run, ok := s.hostRun(req.Command.TaskID)
	if !ok {
		return tr, errors.New("dispatch mapping unavailable")
	}
	stopped, stopErr := s.executor.runtime.Stop(ctx, run.RuntimeRunID)
	if stopErr == nil {
		if evidence, ok := evidenceForRunStatus(stopped.Status); ok {
			_ = s.persistEvidence(req.Command.TaskID, run, evidence, "stop")
		} else {
			return tr, s.reconcileAfterStop(ctx, req.Command.TaskID)
		}
	} else {
		return tr, s.reconcileAfterStop(ctx, req.Command.TaskID)
	}
	return tr, nil
}

func (s *Service) startOrFail(ctx context.Context, req ExecuteRequest, queued space.Transition) (space.Transition, error) {
	if s.executor == nil || s.executor.runtime == nil {
		return s.failQueued(req.Submit, errRuntimeUnavailable)
	}
	runtime := s.executor.runtime
	if _, err := runtime.Capabilities(ctx); err != nil {
		return s.failQueued(req.Submit, err)
	}
	if _, err := runtime.Health(ctx); err != nil {
		return s.failQueued(req.Submit, err)
	}
	state, err := s.state.Read()
	if err != nil {
		return queued, err
	}
	run, err := runtime.CreateRun(ctx, req.Runtime, req.Submit.RequestID)
	if err != nil || run.RunID == "" {
		if err == nil {
			err = errors.New("Hermes create returned no run ID")
		}
		if errors.Is(err, ErrCreateAmbiguous) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return s.finishQueued(req.Submit, space.OutcomeUnknown, err)
		}
		return s.failQueued(req.Submit, err)
	}
	digest := req.RuntimeProfileDigest
	if digest == "" {
		digest = "sha256:hermes-default"
	}
	now := s.executor.options.now().Unix()
	by := req.ReconcileBy
	if by <= now {
		by = now + int64(s.executor.options.reconcileWindow/time.Second)
		if by <= now {
			by = now + 1
		}
	}
	dispatch := space.Command{Type: space.CommandDispatchHostRun, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "dispatch:" + req.Submit.RequestID, TaskID: req.Submit.TaskID, RuntimeRunID: run.RunID, RuntimeProfileDigest: digest, DispatchedAt: now, ReconcileBy: by}
	dispatched, err := s.apply(dispatch)
	if err != nil || dispatched.Rejection != "" {
		return queued, fmt.Errorf("persist dispatch: %w", errOrRejection(err, dispatched.Rejection))
	}
	go s.executor.consume(req.Submit.TaskID, space.HostRun{TaskID: req.Submit.TaskID, RuntimeRunID: run.RunID, RuntimeProfileDigest: digest, HostEpoch: state.Epoch, DispatchedAt: now, ReconcileBy: by})
	return queued, nil
}

func (s *Service) failQueued(req space.Command, cause error) (space.Transition, error) {
	return s.finishQueued(req, space.OutcomeFailed, cause)
}

func (s *Service) finishQueued(req space.Command, outcome space.Outcome, cause error) (space.Transition, error) {
	failed := req
	failed.Type = space.CommandComplete
	failed.RequestID = "failed:" + req.RequestID
	failed.ActorID = req.TargetNodeID
	failed.Outcome = outcome
	tr, err := s.apply(failed)
	if err != nil {
		return tr, errors.Join(cause, err)
	}
	if tr.Rejection != "" {
		return tr, errors.Join(cause, errors.New(tr.Rejection))
	}
	return tr, nil
}

func (s *Service) apply(command space.Command) (space.Transition, error) {
	return s.state.Update(func(state space.State) space.Transition { return space.Apply(state, command) })
}

// ApplyCommand is the durable Host authority boundary used by node/control
// adapters. Callers cannot mutate canonical state without going through the
// Space transition validator and the encrypted store commit.
func (s *Service) ApplyCommand(command space.Command) (space.Transition, error) {
	return s.apply(command)
}

func (s *Service) Task(taskID string) (space.Task, bool, error) {
	state, err := s.state.Read()
	if err != nil {
		return space.Task{}, false, err
	}
	task, ok := state.Tasks[taskID]
	return task, ok, nil
}

func (s *Service) recoverInFlight() {
	state, err := s.state.Read()
	if err != nil || state.SpaceID == "" || state.HostID == "" || state.Epoch == 0 {
		return
	}
	inFlight := false
	for _, task := range state.Tasks {
		switch task.Status {
		case space.OutcomeQueued, space.OutcomeDispatched, space.OutcomeRunning, space.OutcomeCancelling:
			inFlight = true
		}
	}
	if !inFlight {
		return
	}
	requestID := fmt.Sprintf("restart:%d:%d", state.Epoch, s.executor.options.now().UnixNano())
	tr, err := s.apply(space.Command{Type: space.CommandRecoverAfterRestart, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: requestID})
	if err != nil || tr.Rejection != "" || s.executor.runtime == nil {
		return
	}
	state, err = s.state.Read()
	if err != nil {
		return
	}
	for taskID, run := range state.HostRuns {
		if state.Tasks[taskID].Status == space.OutcomeUnknown {
			run := run
			go s.executor.reconcile(taskID, run)
		}
	}
}

func (s *Service) hostRun(taskID string) (space.HostRun, bool) {
	state, err := s.state.Read()
	if err != nil {
		return space.HostRun{}, false
	}
	run, ok := state.HostRuns[taskID]
	return run, ok
}

func (e *executor) consume(taskID string, run space.HostRun) {
	events, err := e.runtime.Events(e.ctx, run.RuntimeRunID)
	terminal := false
	for i, event := range events {
		if event.ID != "" && !e.markSeen(taskID, event.ID) {
			continue
		}
		if evidence, ok := normalizeEvent(event); ok {
			if evidence == space.EvidenceCompleted || evidence == space.EvidenceFailed || evidence == space.EvidenceCancelled {
				terminal = true
			}
			_ = e.service.persistEvidence(taskID, run, evidence, fmt.Sprintf("event:%s:%d", event.ID, i))
		}
	}
	if err != nil || !terminal {
		e.reconcile(taskID, run)
	}
}

func (e *executor) markSeen(taskID, id string) bool {
	e.seenMu.Lock()
	defer e.seenMu.Unlock()
	if e.seen[taskID] == nil {
		e.seen[taskID] = make(map[string]struct{})
	}
	if _, ok := e.seen[taskID][id]; ok {
		return false
	}
	e.seen[taskID][id] = struct{}{}
	return true
}

func (s *Service) persistEvidence(taskID string, run space.HostRun, evidence space.EvidenceOutcome, suffix string) error {
	state, err := s.state.Read()
	if err != nil {
		return err
	}
	observed := s.executor.options.now().Unix()
	if observed < run.DispatchedAt {
		observed = run.DispatchedAt
	}
	_, err = s.apply(space.Command{Type: space.CommandReconcileHostRun, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: fmt.Sprintf("reconcile:%s:%s", taskID, suffix), TaskID: taskID, RuntimeRunID: run.RuntimeRunID, RuntimeProfileDigest: run.RuntimeProfileDigest, Evidence: evidence, ObservedAt: observed})
	return err
}

func (e *executor) reconcile(taskID string, run space.HostRun) {
	e.reconcileContext(e.ctx, taskID, run)
}

func (e *executor) reconcileContext(ctx context.Context, taskID string, run space.HostRun) {
	deadline := time.Unix(run.ReconcileBy, 0)
	sequence := 0
	for {
		if ctx.Err() != nil {
			return
		}
		state, err := e.runtime.RunStatus(ctx, run.RuntimeRunID)
		if err == nil {
			if evidence, ok := evidenceForRunStatus(state.Status); ok {
				sequence++
				if evidence != space.EvidenceRunning || e.options.now().Before(deadline) {
					if persistErr := e.service.persistEvidence(taskID, run, evidence, fmt.Sprintf("status:%d", sequence)); persistErr == nil && evidence != space.EvidenceRunning {
						return
					}
				}
			}
		}
		if !e.options.now().Before(deadline) {
			_ = e.service.persistEvidence(taskID, run, space.EvidenceUnavailable, "unavailable")
			return
		}
		timer := time.NewTimer(e.options.pollInterval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (s *Service) reconcileAfterStop(ctx context.Context, taskID string) error {
	run, ok := s.hostRun(taskID)
	if !ok || s.executor == nil || s.executor.runtime == nil {
		return errors.New("dispatch mapping unavailable")
	}
	deadline := time.Until(time.Unix(run.ReconcileBy, 0))
	if deadline <= 0 {
		deadline = time.Nanosecond
	}
	bounded, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	s.executor.reconcileContext(bounded, taskID, run)
	return nil
}

func evidenceForRunStatus(status string) (space.EvidenceOutcome, bool) {
	switch strings.ToLower(status) {
	case "queued", "started", "running", "stopping":
		return space.EvidenceRunning, true
	case "completed":
		return space.EvidenceCompleted, true
	case "failed":
		return space.EvidenceFailed, true
	case "cancelled", "canceled":
		return space.EvidenceCancelled, true
	default:
		return "", false
	}
}

func normalizeEvent(event hermes.Event) (space.EvidenceOutcome, bool) {
	if evidence, ok := evidenceForRunStatus(event.Type); ok {
		return evidence, true
	}
	if dot := strings.LastIndexByte(event.Type, '.'); dot >= 0 {
		if evidence, ok := evidenceForRunStatus(event.Type[dot+1:]); ok {
			return evidence, true
		}
	}
	var payload struct {
		Status string `json:"status"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(event.Data)))
	decoder.DisallowUnknownFields()
	var trailing any
	if decoder.Decode(&payload) == nil && decoder.Decode(&trailing) == io.EOF {
		return evidenceForRunStatus(payload.Status)
	}
	return "", false
}

func errOrRejection(err error, rejection string) error {
	if err != nil {
		return err
	}
	if rejection != "" {
		return errors.New(rejection)
	}
	return errors.New("operation rejected")
}

func parseArgument(args map[string]string, key string) (string, error) {
	v := strings.TrimSpace(args[key])
	if v == "" {
		return "", fmt.Errorf("missing argument %s", key)
	}
	return v, nil
}

func parseIntArgument(args map[string]string, key string) (int64, error) {
	v, err := parseArgument(args, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func optionalIntArgument(args map[string]string, key string) (int64, error) {
	if strings.TrimSpace(args[key]) == "" {
		return 0, nil
	}
	return strconv.ParseInt(strings.TrimSpace(args[key]), 10, 64)
}

func responseForTransition(tr space.Transition, err error) control.Response {
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	if tr.Rejection != "" {
		return control.Response{Error: tr.Rejection, Data: tr.Receipt}
	}
	return control.Response{OK: true, Data: tr.Receipt}
}
