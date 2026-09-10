package setup

import (
	"context"
	"errors"
	"fmt"
)

// Runner executes the durable setup stages in order. Callers provide the
// platform-specific mutation and verification; the journal is the authority.
type SetupRunner struct {
	Journal   *Journal
	Stages    []Stage
	RunStage  func(context.Context, Stage) error
	Verify    func(context.Context, Stage) error
	InputHash func(Stage) string
}

func (r SetupRunner) Run(ctx context.Context, _ Request) (Report, error) {
	if r.Journal == nil || r.RunStage == nil || r.Verify == nil {
		return Report{Outcome: ActionRequired, Actions: []Action{{Code: "setup_unavailable"}}}, errors.New("setup runner is incomplete")
	}
	stages := r.Stages
	if len(stages) == 0 {
		stages = stageOrder
	}
	for {
		stage := r.Journal.Next()
		if stage == Validated {
			return Report{Outcome: Ready, Stage: Validated}, nil
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
		if err := r.RunStage(ctx, stage); err != nil {
			return Report{Outcome: ActionRequired, Stage: stage}, err
		}
		if err := r.Verify(ctx, stage); err != nil {
			return Report{Outcome: ActionRequired, Stage: stage}, err
		}
		digest := "sha256:" + fmt.Sprintf("%064x", 0)
		if r.InputHash != nil {
			digest = r.InputHash(stage)
		}
		if err := r.Journal.Record(StageEvidence{Stage: stage, InputDigest: digest}); err != nil {
			return Report{Outcome: ActionRequired, Stage: stage}, err
		}
	}
}

// Runner is retained as the public name requested by the setup contract.
type Runner = SetupRunner
