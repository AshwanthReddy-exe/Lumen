package setup

import (
	"context"
	"errors"
	"strings"
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

func TestRunnerRejectsInvalidRequestBeforeMutation(t *testing.T) {
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	r := Runner{Journal: j, RunStage: func(context.Context, Stage) error { calls++; return nil }, Verify: func(context.Context, Stage) error { return nil }}
	report, err := r.Run(context.Background(), Request{Profile: Profile("invalid")})
	if err == nil || report.Outcome != ActionRequired || calls != 0 {
		t.Fatalf("report=%#v err=%v calls=%d", report, err, calls)
	}
}

func TestRunnerUsesNonZeroRequestBoundDigest(t *testing.T) {
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := Runner{Journal: j, RunStage: func(context.Context, Stage) error { return nil }, Verify: func(context.Context, Stage) error { return nil }}
	if _, err := r.Run(context.Background(), Request{Profile: Development}); err != nil {
		t.Fatal(err)
	}
	if got := j.evidence[0].InputDigest; got == "sha256:"+strings.Repeat("0", 64) || !validDigest(got) {
		t.Fatalf("digest=%q", got)
	}
}

func TestRunnerInitializesHostBeforeRecordingHostStageAndRerunSkipsIt(t *testing.T) {
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	initCalls, verifyCalls := 0, 0
	r := Runner{Journal: j, Initializer: HostInitializerFuncs{
		InitializeFunc: func(context.Context) error { initCalls++; return nil },
		VerifyFunc:     func(context.Context) error { verifyCalls++; return nil },
	}, RunStage: func(context.Context, Stage) error { return nil }, Verify: func(context.Context, Stage) error { return nil }}
	if _, err := r.Run(context.Background(), Request{Profile: Development}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), Request{Profile: Development}); err != nil {
		t.Fatal(err)
	}
	if initCalls != 1 || verifyCalls != 1 {
		t.Fatalf("init=%d verify=%d", initCalls, verifyCalls)
	}
}

func TestRunnerReturnsRedactedStageError(t *testing.T) {
	j, err := NewJournal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := Runner{Journal: j, RunStage: func(context.Context, Stage) error { return errors.New("open /Users/owner/private/token: secret") }, Verify: func(context.Context, Stage) error { return nil }}
	report, err := r.Run(context.Background(), Request{Profile: Development})
	if err == nil || report.Outcome != ActionRequired || strings.Contains(err.Error(), "/Users/") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}
