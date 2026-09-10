package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// Runner executes the durable setup stages in order. Callers provide the
// platform-specific mutation and verification; the journal is the authority.
type SetupRunner struct {
	Journal     *Journal
	Stages      []Stage
	Probe       Probe
	Plan        *PlanResult
	Initializer HostInitializer
	RunStage    func(context.Context, Stage) error
	Verify      func(context.Context, Stage) error
	InputHash   func(Stage) string
}

// HostInitializerFuncs adapts the concrete create-once Host lifecycle without
// making the platform-independent setup package depend on the Host package.
type HostInitializerFuncs struct {
	InitializeFunc func(context.Context) error
	VerifyFunc     func(context.Context) error
}

func (f HostInitializerFuncs) Initialize(ctx context.Context) error {
	if f.InitializeFunc == nil {
		return errors.New("Host initializer is incomplete")
	}
	return f.InitializeFunc(ctx)
}
func (f HostInitializerFuncs) Verify(ctx context.Context) error {
	if f.VerifyFunc == nil {
		return errors.New("Host initializer is incomplete")
	}
	return f.VerifyFunc(ctx)
}

func (r SetupRunner) Run(ctx context.Context, req Request) (Report, error) {
	if req.Profile != Development && req.Profile != PersonalAlpha && req.Profile != Hardened {
		return Report{Outcome: ActionRequired, Profile: req.Profile, Actions: []Action{{Code: "invalid_profile"}}}, errors.New("invalid setup profile")
	}
	if r.Journal == nil || r.Verify == nil || (r.RunStage == nil && r.Initializer == nil) {
		return Report{Outcome: ActionRequired, Actions: []Action{{Code: "setup_unavailable"}}}, errors.New("setup runner is incomplete")
	}
	if r.Probe != nil {
		planned, err := Plan(ctx, req, r.Probe)
		if err != nil {
			return Report{Outcome: ActionRequired, Profile: req.Profile, Actions: []Action{{Code: "plan_unavailable"}}}, err
		}
		r.Plan = &planned
	}
	if r.Plan != nil {
		if r.Plan.Profile != "" && r.Plan.Profile != req.Profile {
			return Report{Outcome: ActionRequired, Profile: req.Profile, Actions: []Action{{Code: "plan_mismatch"}}}, errors.New("setup plan profile mismatch")
		}
		if r.Plan.NextStage != "" && !validStage(r.Plan.NextStage) {
			return Report{Outcome: ActionRequired, Profile: req.Profile, Actions: []Action{{Code: "invalid_stage_plan"}}}, errors.New("setup plan has invalid next stage")
		}
	}
	stages := r.Stages
	if len(stages) == 0 {
		stages = stageOrder
	}
	if err := validateStageSelection(stages); err != nil {
		return Report{Outcome: ActionRequired, Profile: req.Profile, Actions: []Action{{Code: "invalid_stage_plan"}}}, err
	}
	for {
		stage := r.Journal.Next()
		if stage == Validated {
			return r.readyReport(req), nil
		}
		found := false
		for _, s := range stages {
			if s == stage {
				found = true
				break
			}
		}
		if !found {
			return Report{Outcome: ActionRequired, Stage: stage}, fmt.Errorf("stage %s is not selected", stage)
		}
		if r.Initializer != nil && stage == HostInitialized {
			if err := r.Initializer.Initialize(ctx); err != nil {
				return stageFailure(stage, err)
			}
			if err := r.Initializer.Verify(ctx); err != nil {
				return stageFailure(stage, err)
			}
		} else if r.RunStage != nil {
			if err := r.RunStage(ctx, stage); err != nil {
				return stageFailure(stage, err)
			}
		}
		if err := r.Verify(ctx, stage); err != nil {
			return stageFailure(stage, err)
		}
		digest := requestDigest(req, stage, r.Plan)
		if r.InputHash != nil {
			digest = r.InputHash(stage)
		}
		if !validDigest(digest) || digest == "sha256:"+fmt.Sprintf("%064x", 0) {
			return stageFailure(stage, ErrInvalidDigest)
		}
		if err := r.Journal.Record(StageEvidence{Stage: stage, InputDigest: digest}); err != nil {
			return stageFailure(stage, err)
		}
	}
}

func validateStageSelection(stages []Stage) error {
	if len(stages) == 0 {
		return errors.New("empty stage plan")
	}
	seen := make(map[Stage]bool, len(stages))
	for i, stage := range stages {
		if !validStage(stage) || seen[stage] {
			return errors.New("invalid stage plan")
		}
		seen[stage] = true
		if i > 0 && stageOrderIndex(stage) != stageOrderIndex(stages[i-1])+1 {
			return errors.New("stage plan is not ordered")
		}
	}
	return nil
}
func stageOrderIndex(stage Stage) int {
	for i, s := range stageOrder {
		if s == stage {
			return i
		}
	}
	return -1
}
func requestDigest(req Request, stage Stage, p *PlanResult) string {
	material := string(req.Profile) + "\x00" + string(stage)
	if p != nil {
		material += "\x00" + p.LumenVersion + "\x00" + p.HermesVersion + "\x00" + string(p.Platform) + "\x00" + p.Architecture
	}
	h := sha256.Sum256([]byte(material))
	return "sha256:" + hex.EncodeToString(h[:])
}
func (r SetupRunner) readyReport(req Request) Report {
	out := Report{Outcome: Ready, Stage: Validated, Profile: req.Profile}
	if r.Plan != nil {
		out.Platform, out.LumenVersion, out.HermesVersion = r.Plan.Platform, r.Plan.LumenVersion, r.Plan.HermesVersion
	}
	return out
}
func stageFailure(stage Stage, err error) (Report, error) {
	return Report{Outcome: ActionRequired, Stage: stage, Actions: []Action{{Code: "stage_failed", Detail: string(stage)}}}, stageError{stage: stage, cause: err}
}

type stageError struct {
	stage Stage
	cause error
}

func (e stageError) Error() string { return fmt.Sprintf("setup stage %s failed", e.stage) }
func (e stageError) Unwrap() error { return e.cause }

// Runner is retained as the public name requested by the setup contract.
type Runner = SetupRunner
