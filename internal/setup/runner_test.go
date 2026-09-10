package setup

import (
	"context"
	"errors"
	"testing"
)

func TestRunnerResumesWithoutRepeatingCompletedStages(t *testing.T) {
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	fail := true
	r := Runner{Journal: j, RunStage: func(_ context.Context, s Stage) error {
		calls++
		if s == ServicesStarted && fail {
			fail = false
			return errors.New("interrupt")
		}
		return nil
	}, Verify: func(context.Context, Stage) error { return nil }}
	if _, err := r.Run(context.Background(), Request{Profile: Development}); err == nil {
		t.Fatal("expected interruption")
	}
	report, err := r.Run(context.Background(), Request{Profile: Development})
	if err != nil || report.Outcome != Ready || calls != len(stageOrder) {
		t.Fatalf("report=%#v err=%v calls=%d", report, err, calls)
	}
}
