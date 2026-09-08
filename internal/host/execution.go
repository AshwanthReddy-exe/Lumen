package host

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
	"github.com/AshwanthReddy-exe/Lumen/internal/store"
)

var errRuntimeUnavailable = errors.New("Hermes runtime unavailable")

// ErrCreateAmbiguous lets a runtime adapter distinguish a transport outcome
// that may have created a run from a proven pre-dispatch rejection.
var ErrCreateAmbiguous = hermes.ErrCreateAmbiguous

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

type RuntimeApprovalRequest struct{ Command space.Command }

type CancelRequest struct{ Command space.Command }

type configuredAdapter struct {
	client *hermes.Client
	err    error
}

func configuredRuntime(c Config) (hermes.Adapter, error) {
	client, err := buildHermesClient(c)
	if err != nil {
		return nil, err
	}
	return &configuredAdapter{client: client}, nil
}

func buildHermesClient(c Config) (*hermes.Client, error) {
	profile := c.HermesProfile
	if profile == "" {
		profile = hermes.ProfileHardened
	}
	bearer, err := readRestrictedSecret(c.HermesBearerPath)
	if err != nil {
		return nil, err
	}
	cfg := hermes.Config{BaseURL: c.HermesBaseURL, ProfileMode: profile, MaxResponseBytes: 8 << 20, MaxEventBytes: 1 << 20, MaxEventStreamBytes: 8 << 20, RequestTimeout: 10 * time.Second, EventTimeout: 30 * time.Second, BearerToken: strings.TrimSpace(string(bearer))}
	if profile == hermes.ProfileHardened {
		caPEM, err := readRestrictedSecret(c.HermesCAPath)
		if err != nil {
			return nil, err
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("invalid Hermes CA file")
		}
		certPEM, err := readRestrictedSecret(c.HermesClientCertPath)
		if err != nil {
			return nil, err
		}
		keyPEM, err := readRestrictedSecret(c.HermesClientKeyPath)
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		cfg.TLS = hermes.TLSConfig{RootCAs: roots, ClientCertificate: cert, ServerCertPin: c.HermesServerPin}
	}
	return hermes.New(cfg)
}

func readRestrictedSecret(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("Hermes secret path is required")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("invalid Hermes secret descriptor")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return nil, errors.New("Hermes secret must be an owner-only regular file")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint32(os.Getuid()) != stat.Uid {
		return nil, errors.New("Hermes secret must be owned by the current user")
	}
	return io.ReadAll(f)
}

func (a *configuredAdapter) get() (*hermes.Client, error) {
	if a.client == nil {
		return nil, a.err
	}
	return a.client, nil
}
func (a *configuredAdapter) Capabilities(ctx context.Context) (hermes.Capabilities, error) {
	c, err := a.get()
	if err != nil {
		return hermes.Capabilities{}, err
	}
	return c.Capabilities(ctx)
}
func (a *configuredAdapter) Health(ctx context.Context) (hermes.Health, error) {
	c, err := a.get()
	if err != nil {
		return hermes.Health{}, err
	}
	return c.Health(ctx)
}
func (a *configuredAdapter) CreateRun(ctx context.Context, in hermes.CreateRunRequest, key string) (hermes.Run, error) {
	c, err := a.get()
	if err != nil {
		return hermes.Run{}, err
	}
	return c.CreateRun(ctx, in, key)
}
func (a *configuredAdapter) RunStatus(ctx context.Context, id string) (hermes.Run, error) {
	c, err := a.get()
	if err != nil {
		return hermes.Run{}, err
	}
	return c.RunStatus(ctx, id)
}
func (a *configuredAdapter) Events(ctx context.Context, id string) ([]hermes.Event, error) {
	c, err := a.get()
	if err != nil {
		return nil, err
	}
	return c.Events(ctx, id)
}
func (a *configuredAdapter) EventsStream(ctx context.Context, id string) (hermes.EventIterator, error) {
	c, err := a.get()
	if err != nil {
		return nil, err
	}
	return c.EventsStream(ctx, id)
}
func (a *configuredAdapter) ResolveApproval(ctx context.Context, id, decision string) error {
	c, err := a.get()
	if err != nil {
		return err
	}
	return c.ResolveApproval(ctx, id, decision)
}
func (a *configuredAdapter) Steer(ctx context.Context, id string, in hermes.SteerRequest) error {
	c, err := a.get()
	if err != nil {
		return err
	}
	return c.Steer(ctx, id, in)
}
func (a *configuredAdapter) Stop(ctx context.Context, id string) (hermes.Run, error) {
	c, err := a.get()
	if err != nil {
		return hermes.Run{}, err
	}
	return c.Stop(ctx, id)
}

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
	state, err := s.state.Read()
	if err != nil {
		return control.Response{Error: "state unavailable"}
	}
	if runtimeApprovalID := strings.TrimSpace(args["runtime_approval_id"]); runtimeApprovalID != "" {
		requestID, parseErr := parseArgument(args, "request_id")
		if parseErr != nil {
			return control.Response{Error: parseErr.Error()}
		}
		taskID, parseErr := parseArgument(args, "task_id")
		if parseErr != nil {
			return control.Response{Error: parseErr.Error()}
		}
		target, parseErr := parseArgument(args, "target_node_id")
		if parseErr != nil {
			return control.Response{Error: parseErr.Error()}
		}
		fingerprint, parseErr := parseArgument(args, "action_fingerprint")
		if parseErr != nil {
			return control.Response{Error: parseErr.Error()}
		}
		runID, parseErr := parseArgument(args, "runtime_run_id")
		if parseErr != nil {
			return control.Response{Error: parseErr.Error()}
		}
		observed, parseErr := parseIntArgument(args, "observed_at")
		if parseErr != nil {
			return control.Response{Error: parseErr.Error()}
		}
		run := state.HostRuns[taskID]
		tr, callErr := s.ResolveRuntimeApproval(ctx, RuntimeApprovalRequest{Command: space.Command{Type: space.CommandResolveRuntimeApproval, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: requestID, TaskID: taskID, RuntimeRunID: runID, RuntimeProfileDigest: run.RuntimeProfileDigest, RuntimeApprovalID: runtimeApprovalID, TargetNodeID: target, ActionFingerprint: fingerprint, Decision: decision, ObservedAt: observed}})
		return responseForTransition(tr, callErr)
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
	if decision == "deny" {
		tr, callErr := s.ResolveApproval(ctx, ApprovalRequest{Command: space.Command{Type: space.CommandApprove, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: requestID, TaskID: taskID, TargetNodeID: target, ActionFingerprint: fingerprint, ApprovalID: approvalID, ApprovedAt: approvedAt, ExpiresAt: expiresAt, Decision: "deny"}})
		return responseForTransition(tr, callErr)
	}
	tr, callErr := s.ResolveApproval(ctx, ApprovalRequest{Command: space.Command{Type: space.CommandApprove, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.OwnerID, RequestID: requestID, TaskID: taskID, TargetNodeID: target, ActionFingerprint: fingerprint, ApprovalID: approvalID, ApprovedAt: approvedAt, ExpiresAt: expiresAt}, Runtime: hermes.CreateRunRequest{Input: args["input"], SessionID: args["session_id"], Instructions: args["instructions"]}, RuntimeProfileDigest: args["runtime_profile_digest"], ReconcileBy: by})
	return responseForTransition(tr, callErr)
}

type executor struct {
	service    *Service
	runtime    hermes.Adapter
	options    executionOptions
	ctx        context.Context
	cancel     context.CancelFunc
	seenMu     sync.Mutex
	seen       map[string]map[string]struct{}
	consumerMu sync.Mutex
	consumers  map[string]struct{}
	closed     bool
	wg         sync.WaitGroup
}

func newExecutor(s *Service, runtime hermes.Adapter, options executionOptions) *executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &executor{service: s, runtime: runtime, options: options, ctx: ctx, cancel: cancel, seen: make(map[string]map[string]struct{}), consumers: make(map[string]struct{})}
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
	tr, err := s.apply(req.Submit)
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		return tr, err
	}
	if tr.Replayed {
		return tr, nil
	}
	return s.startOrFail(ctx, req, tr)
}

func (s *Service) ResolveApproval(ctx context.Context, req ApprovalRequest) (space.Transition, error) {
	tr, err := s.apply(req.Command)
	if err != nil || tr.Rejection != "" || tr.Receipt.Outcome != space.OutcomeQueued {
		return tr, err
	}
	if tr.Replayed {
		return tr, nil
	}
	state, readErr := s.state.Read()
	if readErr != nil {
		return tr, readErr
	}
	task := state.Tasks[req.Command.TaskID]
	return s.startOrFail(ctx, ExecuteRequest{Submit: space.Command{Type: space.CommandSubmit, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: task.CommandID, TaskID: task.ID, OriginNodeID: task.OriginNodeID, TargetNodeID: task.TargetNodeID, CapabilityID: task.CapabilityID, Action: task.Action, ActionFingerprint: task.ActionFingerprint}, Runtime: req.Runtime, RuntimeProfileDigest: req.RuntimeProfileDigest, ReconcileBy: req.ReconcileBy}, tr)
}

func (s *Service) ResolveRuntimeApproval(ctx context.Context, req RuntimeApprovalRequest) (space.Transition, error) {
	tr, err := s.apply(req.Command)
	if err != nil || tr.Rejection != "" {
		return tr, err
	}
	state, readErr := s.state.Read()
	if readErr != nil {
		return tr, readErr
	}
	approval, approvalOK := state.RuntimeApprovals[req.Command.RuntimeApprovalID]
	if !approvalOK || approval.Decision == "" {
		return tr, errors.New("runtime approval decision unavailable")
	}
	if approval.DeliveryState == "delivered" {
		return tr, nil
	}
	if s.executor == nil || s.executor.options.now().Unix() >= approval.ExpiresAt {
		return tr, errors.New("runtime approval expired")
	}
	run, ok := state.HostRuns[req.Command.TaskID]
	if !ok || s.executor == nil || s.executor.runtime == nil {
		return tr, errors.New("runtime approval mapping unavailable")
	}
	forwardErr := s.executor.runtime.ResolveApproval(ctx, run.RuntimeRunID, approval.Decision)
	deliveryState := "delivered"
	if forwardErr != nil {
		deliveryState = "uncertain"
	}
	delivery := space.Command{Type: space.CommandRecordRuntimeApproval, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "runtime-approval-delivery:" + approval.ID + ":" + approval.Decision + ":" + deliveryState, TaskID: approval.TaskID, RuntimeRunID: approval.RuntimeRunID, RuntimeProfileDigest: run.RuntimeProfileDigest, RuntimeApprovalID: approval.ID, Decision: approval.Decision, DeliveryState: deliveryState}
	delivery.TargetNodeID = approval.TargetNodeID
	delivery.ActionFingerprint = approval.ActionFingerprint
	dispatched, applyErr := s.apply(delivery)
	if applyErr != nil || dispatched.Rejection != "" {
		if applyErr == nil {
			applyErr = errors.New(dispatched.Rejection)
		}
		if forwardErr != nil {
			return tr, errors.Join(forwardErr, applyErr)
		}
		return tr, applyErr
	}
	if forwardErr != nil {
		return dispatched, forwardErr
	}
	return dispatched, nil
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
			if persistErr := s.persistEvidence(req.Command.TaskID, run, evidence, "stop"); persistErr != nil {
				if terminal, readErr := s.taskTerminal(req.Command.TaskID); readErr == nil && terminal {
					return tr, nil
				}
				reconcileErr := s.reconcileAfterStop(ctx, req.Command.TaskID)
				if reconcileErr != nil {
					return tr, errors.Join(persistErr, reconcileErr)
				}
			}
		} else {
			return tr, s.reconcileAfterStop(ctx, req.Command.TaskID)
		}
	} else {
		return tr, s.reconcileAfterStop(ctx, req.Command.TaskID)
	}
	return tr, nil
}

func (s *Service) taskTerminal(taskID string) (bool, error) {
	state, err := s.state.Read()
	if err != nil {
		return false, err
	}
	switch state.Tasks[taskID].Status {
	case space.OutcomeCompleted, space.OutcomeFailed, space.OutcomeCancelled, space.OutcomeUnknown:
		return true, nil
	default:
		return false, nil
	}
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
	intent, err := s.apply(space.Command{Type: space.CommandCreateHostRun, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "create:" + req.Submit.RequestID, TaskID: req.Submit.TaskID, RuntimeIdempotencyKey: req.Submit.RequestID, RuntimeProfileDigest: digest, DispatchedAt: now, ReconcileBy: by})
	if err != nil || intent.Rejection != "" {
		return queued, fmt.Errorf("persist create intent: %w", errOrRejection(err, intent.Rejection))
	}
	run, err := runtime.CreateRun(ctx, req.Runtime, req.Submit.RequestID)
	if err != nil || run.RunID == "" {
		if err == nil {
			err = errors.New("Hermes create returned no run ID")
		}
		if !errors.Is(err, hermes.ErrCreateRejected) {
			return s.finishQueued(req.Submit, space.OutcomeUnknown, errOrRejection(err, "ambiguous_create"))
		}
		return s.failQueued(req.Submit, err)
	}
	dispatch := space.Command{Type: space.CommandDispatchHostRun, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "dispatch:" + req.Submit.RequestID, TaskID: req.Submit.TaskID, RuntimeRunID: run.RunID, RuntimeIdempotencyKey: req.Submit.RequestID, RuntimeProfileDigest: digest, DispatchedAt: now, ReconcileBy: by}
	dispatched, err := s.apply(dispatch)
	if err != nil || dispatched.Rejection != "" {
		if err == nil && dispatched.Rejection != "" {
			return s.finishQueued(req.Submit, space.OutcomeUnknown, errors.New(dispatched.Rejection))
		}
		return queued, fmt.Errorf("persist dispatch: %w", errOrRejection(err, dispatched.Rejection))
	}
	s.executor.startConsumer(req.Submit.TaskID, space.HostRun{TaskID: req.Submit.TaskID, RuntimeRunID: run.RunID, RuntimeProfileDigest: digest, HostEpoch: state.Epoch, DispatchedAt: now, ReconcileBy: by})
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
		case space.OutcomeQueued, space.OutcomeCreating, space.OutcomeAwaitingPermission, space.OutcomeDispatched, space.OutcomeRunning, space.OutcomeCancelling:
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
		switch state.Tasks[taskID].Status {
		case space.OutcomeUnknown, space.OutcomeAwaitingPermission, space.OutcomeDispatched, space.OutcomeRunning, space.OutcomeCancelling:
			run := run
			s.executor.startConsumer(taskID, run)
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
	for {
		terminal, retryApproval := e.consumeOnce(taskID, run)
		if terminal {
			e.forgetSeen(taskID)
			return
		}
		if retryApproval {
			timer := time.NewTimer(5 * time.Millisecond)
			select {
			case <-timer.C:
			case <-e.ctx.Done():
				timer.Stop()
				return
			}
			continue
		}
		e.reconcile(taskID, run)
		return
	}
}

func (e *executor) consumeOnce(taskID string, run space.HostRun) (terminal, retryApproval bool) {
	streamCtx := e.ctx
	remaining := time.Duration(run.ReconcileBy-e.options.now().Unix()) * time.Second
	if remaining <= 0 {
		return e.service.persistEvidence(taskID, run, space.EvidenceUnavailable, "expired") == nil, false
	}
	var cancel context.CancelFunc
	streamCtx, cancel = context.WithTimeout(e.ctx, remaining)
	defer cancel()
	process := func(i int, event hermes.Event) (bool, bool) {
		if event.ID != "" && e.isSeen(taskID, event.ID) {
			return false, false
		}
		if approval, ok := normalizeApproval(event); ok {
			if err := e.persistRuntimeApprovalRetry(streamCtx, taskID, run, approval); err != nil {
				return false, true
			}
			if event.ID != "" {
				e.markSeen(taskID, event.ID)
			}
			return false, false
		}
		if evidence, ok := normalizeEvent(event); ok {
			persistErr := e.service.persistEvidence(taskID, run, evidence, fmt.Sprintf("event:%s:%d", event.ID, i))
			if persistErr == nil && event.ID != "" {
				e.markSeen(taskID, event.ID)
			}
			if persistErr == nil && (evidence == space.EvidenceCompleted || evidence == space.EvidenceFailed || evidence == space.EvidenceCancelled) {
				return true, false
			}
		}
		return false, false
	}
	if streamAdapter, ok := e.runtime.(hermes.StreamAdapter); ok {
		stream, err := streamAdapter.EventsStream(streamCtx, run.RuntimeRunID)
		if err != nil {
			return false, false
		}
		defer stream.Close()
		for i := 0; ; i++ {
			event, nextErr := stream.Next(streamCtx)
			if nextErr != nil {
				if errors.Is(nextErr, context.DeadlineExceeded) {
					return e.service.persistEvidence(taskID, run, space.EvidenceUnavailable, "stream-deadline") == nil, false
				}
				return false, false
			}
			terminal, retryApproval := process(i, event)
			if terminal || retryApproval {
				return terminal, retryApproval
			}
		}
	}
	events, err := e.runtime.Events(streamCtx, run.RuntimeRunID)
	if errors.Is(err, context.DeadlineExceeded) {
		return e.service.persistEvidence(taskID, run, space.EvidenceUnavailable, "stream-deadline") == nil, false
	}
	if err != nil && len(events) == 0 {
		return false, false
	}
	for i, event := range events {
		terminal, retryApproval := process(i, event)
		if terminal || retryApproval {
			return terminal, retryApproval
		}
	}
	return false, false
}

func (e *executor) isSeen(taskID, id string) bool {
	e.seenMu.Lock()
	defer e.seenMu.Unlock()
	_, ok := e.seen[taskID][id]
	return ok
}

func (e *executor) markSeen(taskID, id string) {
	e.seenMu.Lock()
	defer e.seenMu.Unlock()
	if e.seen[taskID] == nil {
		e.seen[taskID] = make(map[string]struct{})
	}
	e.seen[taskID][id] = struct{}{}
}

func (e *executor) forgetSeen(taskID string) {
	e.seenMu.Lock()
	delete(e.seen, taskID)
	e.seenMu.Unlock()
}

func (e *executor) persistRuntimeApprovalRetry(ctx context.Context, taskID string, run space.HostRun, approval runtimeApprovalEvidence) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if err := e.service.persistRuntimeApproval(taskID, run, approval); err == nil {
			return nil
		} else {
			last = err
		}
		if attempt < 2 {
			timer := time.NewTimer(5 * time.Millisecond)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}
	return last
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
	if evidence == space.EvidenceUnavailable && observed < run.ReconcileBy {
		observed = run.ReconcileBy
	}
	command := space.Command{Type: space.CommandReconcileHostRun, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: fmt.Sprintf("reconcile:%s:%s", taskID, suffix), TaskID: taskID, RuntimeRunID: run.RuntimeRunID, RuntimeProfileDigest: run.RuntimeProfileDigest, Evidence: evidence, ObservedAt: observed}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		tr, applyErr := s.apply(command)
		if applyErr == nil && tr.Rejection == "" {
			return nil
		}
		last = applyErr
		if last == nil {
			last = errors.New(tr.Rejection)
		}
		if attempt < 2 {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return last
}

func (e *executor) startConsumer(taskID string, run space.HostRun) {
	e.consumerMu.Lock()
	if e.closed {
		e.consumerMu.Unlock()
		return
	}
	if _, exists := e.consumers[taskID]; exists {
		e.consumerMu.Unlock()
		return
	}
	e.consumers[taskID] = struct{}{}
	e.wg.Add(1)
	e.consumerMu.Unlock()
	go func() {
		defer e.wg.Done()
		defer func() {
			e.consumerMu.Lock()
			delete(e.consumers, taskID)
			e.consumerMu.Unlock()
		}()
		e.consume(taskID, run)
	}()
}

func (e *executor) waitConsumers() { e.wg.Wait() }

func (e *executor) shutdown() {
	e.consumerMu.Lock()
	e.closed = true
	e.cancel()
	e.consumerMu.Unlock()
	e.wg.Wait()
}

func (e *executor) reconcile(taskID string, run space.HostRun) {
	e.reconcileContext(e.ctx, taskID, run)
}

func (e *executor) reconcileContext(ctx context.Context, taskID string, run space.HostRun) {
	sequence := 0
	for {
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				if e.service.persistEvidence(taskID, run, space.EvidenceUnavailable, "unavailable") == nil {
					e.forgetSeen(taskID)
				}
			}
			return
		}
		remaining := time.Duration(run.ReconcileBy-e.options.now().Unix()) * time.Second
		if remaining <= 0 {
			if e.service.persistEvidence(taskID, run, space.EvidenceUnavailable, "unavailable") == nil {
				e.forgetSeen(taskID)
			}
			return
		}
		statusCtx, cancel := context.WithTimeout(ctx, remaining)
		state, err := e.runtime.RunStatus(statusCtx, run.RuntimeRunID)
		cancel()
		if err == nil {
			if evidence, ok := evidenceForRunStatus(state.Status); ok {
				sequence++
				if persistErr := e.service.persistEvidence(taskID, run, evidence, fmt.Sprintf("status:%d", sequence)); persistErr == nil && evidence != space.EvidenceRunning {
					e.forgetSeen(taskID)
					return
				}
			}
		}
		wait := e.options.pollInterval
		if remaining < wait {
			wait = remaining
		}
		timer := time.NewTimer(wait)
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
	deadline := time.Unix(run.ReconcileBy, 0).Sub(s.executor.options.now())
	if deadline <= 0 {
		deadline = time.Nanosecond
	}
	bounded, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	s.executor.reconcileContext(bounded, taskID, run)
	state, err := s.state.Read()
	if err != nil {
		return err
	}
	switch state.Tasks[taskID].Status {
	case space.OutcomeCompleted, space.OutcomeFailed, space.OutcomeCancelled, space.OutcomeUnknown:
		return nil
	default:
		if s.executor.options.now().Unix() >= run.ReconcileBy {
			if persistErr := s.persistEvidence(taskID, run, space.EvidenceUnavailable, "unavailable"); persistErr == nil {
				return nil
			}
		}
		return errors.New("bounded reconciliation did not reach a durable terminal outcome")
	}
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
	var payload map[string]json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(string(event.Data)))
	var trailing any
	if decoder.Decode(&payload) == nil && decoder.Decode(&trailing) == io.EOF {
		var status string
		if raw, ok := payload["status"]; ok && json.Unmarshal(raw, &status) == nil {
			return evidenceForRunStatus(status)
		}
	}
	return "", false
}

type runtimeApprovalEvidence struct {
	ID                string
	ActorNodeID       string
	TargetNodeID      string
	ActionFingerprint string
	ExpiresAt         int64
}

func normalizeApproval(event hermes.Event) (runtimeApprovalEvidence, bool) {
	var payload map[string]json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(string(event.Data)))
	var trailing any
	if decoder.Decode(&payload) != nil || decoder.Decode(&trailing) != io.EOF {
		return runtimeApprovalEvidence{}, false
	}
	var status, id, actor, target, fingerprint string
	var expires int64
	if json.Unmarshal(payload["status"], &status) != nil || status != "awaiting_approval" || json.Unmarshal(payload["approval_id"], &id) != nil || json.Unmarshal(payload["target_node_id"], &target) != nil || json.Unmarshal(payload["action_fingerprint"], &fingerprint) != nil || json.Unmarshal(payload["expires_at"], &expires) != nil {
		return runtimeApprovalEvidence{}, false
	}
	if raw, ok := payload["actor_node_id"]; ok && json.Unmarshal(raw, &actor) != nil {
		return runtimeApprovalEvidence{}, false
	}
	return runtimeApprovalEvidence{ID: id, ActorNodeID: actor, TargetNodeID: target, ActionFingerprint: fingerprint, ExpiresAt: expires}, true
}

func (s *Service) persistRuntimeApproval(taskID string, run space.HostRun, approval runtimeApprovalEvidence) error {
	state, err := s.state.Read()
	if err != nil {
		return err
	}
	task := state.Tasks[taskID]
	if approval.ActorNodeID != "" && approval.ActorNodeID != state.HostID {
		return errors.New("runtime approval actor mismatch")
	}
	if approval.TargetNodeID != task.TargetNodeID || approval.ActionFingerprint != task.ActionFingerprint || approval.ExpiresAt <= s.executor.options.now().Unix() {
		return errors.New("runtime approval binding mismatch")
	}
	tr, err := s.apply(space.Command{Type: space.CommandRequestRuntimeApproval, SpaceID: state.SpaceID, HostID: state.HostID, Epoch: state.Epoch, ActorID: state.HostID, RequestID: "runtime-approval:" + approval.ID, TaskID: taskID, RuntimeRunID: run.RuntimeRunID, RuntimeProfileDigest: run.RuntimeProfileDigest, RuntimeApprovalID: approval.ID, TargetNodeID: approval.TargetNodeID, ActionFingerprint: approval.ActionFingerprint, ExpiresAt: approval.ExpiresAt, ObservedAt: s.executor.options.now().Unix()})
	if err != nil {
		return err
	}
	if tr.Rejection != "" {
		if tr.Rejection == "runtime_approval_exists" {
			state, readErr := s.state.Read()
			if readErr == nil {
				existing, exists := state.RuntimeApprovals[approval.ID]
				if exists && existing.TaskID == taskID && existing.RuntimeRunID == run.RuntimeRunID && existing.TargetNodeID == approval.TargetNodeID && existing.ActionFingerprint == approval.ActionFingerprint && existing.ExpiresAt == approval.ExpiresAt {
					return nil
				}
			}
		}
		return errors.New(tr.Rejection)
	}
	return nil
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
