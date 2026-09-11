package setup

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
)

type recordingRunner struct {
	calls [][]string
	err   error
	ctx   context.Context
}

func (r *recordingRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.ctx = ctx
	r.calls = append(r.calls, append([]string{name}, args...))
	return []byte("active (running)"), nil, r.err
}

func TestSupervisorMissingManagerAndStatusError(t *testing.T) {
	old := commandRunner
	defer func() { commandRunner = old }()
	r := &recordingRunner{err: errors.New("missing")}
	commandRunner = r
	_, err := (CommandSupervisor{Manager: "other"}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost})
	var required *ActionRequiredError
	if !errors.As(err, &required) {
		t.Fatalf("want action required, got %v", err)
	}
	commandRunner = r
	_, err = (CommandSupervisor{Manager: SupervisorSystemd}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost})
	if err == nil {
		t.Fatal("status swallowed manager error")
	}
}

func TestSupervisorStatusParsesRunningAndBoundsContext(t *testing.T) {
	old := commandRunner
	defer func() { commandRunner = old }()
	r := &recordingRunner{}
	commandRunner = r
	got, err := (CommandSupervisor{Manager: SupervisorSystemd}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost})
	if err != nil || got[0].State != StateRunning {
		t.Fatalf("got %#v %v", got, err)
	}
	if _, ok := r.ctx.Deadline(); !ok {
		t.Fatal("manager call was not bounded")
	}
}

func TestSupervisorRejectsInvalidInput(t *testing.T) {
	s := CommandSupervisor{Manager: SupervisorSystemd}
	if _, err := s.Control(context.Background(), Action{Code: "shell"}, []ServiceName{ServiceHost}); err == nil {
		t.Fatal("accepted invalid action")
	}
	if _, err := s.Control(context.Background(), Action{Code: "status"}, []ServiceName{"other"}); err == nil {
		t.Fatal("accepted invalid service")
	}
}

func TestSupervisorRestartUsesBoundedContext(t *testing.T) {
	old := commandRunner
	defer func() { commandRunner = old }()
	r := &recordingRunner{}
	commandRunner = r
	if _, err := (CommandSupervisor{Manager: SupervisorSystemd}).Control(context.Background(), Action{Code: "restart"}, []ServiceName{ServiceHost}); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.ctx.Deadline(); !ok {
		t.Fatal("restart was not bounded")
	}
}

type fakeSupervisor struct{ calls []string }

func (f *fakeSupervisor) Install(context.Context, ServicePlan) error {
	f.calls = append(f.calls, "install:hermes", "install:host")
	return nil
}

func TestDefinitionInstallCopiesExactBytes(t *testing.T) {
	d := t.TempDir()
	src := d + "/source"
	dst := d + "/nested/service"
	if err := os.WriteFile(src, []byte("unit"), 0600); err != nil {
		t.Fatal(err)
	}
	s := CommandSupervisor{Manager: SupervisorDocker}
	if err := s.Install(context.Background(), ServicePlan{Hermes: ServiceDefinition{Name: ServiceHermes, Source: src, Destination: dst}, Host: ServiceDefinition{Name: ServiceHost, Path: dst}, Initializer: fakeInitializer{}}); err != nil {
		t.Fatalf("install definition: %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil || string(b) != "unit" {
		t.Fatalf("installed bytes: %q %v", b, err)
	}
}
func (f *fakeSupervisor) Enable(_ context.Context, names []ServiceName) error {
	for _, n := range names {
		f.calls = append(f.calls, "enable:"+string(n))
	}
	return nil
}
func (f *fakeSupervisor) Control(context.Context, Action, []ServiceName) ([]ServiceState, error) {
	return nil, nil
}

func TestInstallEnablesHermesAndHostInOrder(t *testing.T) {
	f := &fakeSupervisor{}
	if err := InstallServices(context.Background(), f, ServicePlan{Host: ServiceDefinition{Name: ServiceHost}, Hermes: ServiceDefinition{Name: ServiceHermes}, Initializer: fakeInitializer{}}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.calls, []string{"install:hermes", "install:host", "enable:hermes", "enable:host"}) {
		t.Fatalf("got %#v", f.calls)
	}
}

type fakeInitializer struct{}

func (fakeInitializer) Initialize(context.Context) error { return nil }
func (fakeInitializer) Verify(context.Context) error     { return nil }
