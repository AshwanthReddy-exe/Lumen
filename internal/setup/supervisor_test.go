package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func TestDockerSupervisorUsesComposeServiceNamesAndOptions(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &recordingRunner{}
	commandRunner = r
	if _, err := (CommandSupervisor{Manager: SupervisorDocker}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHermes}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"docker", "compose", "-f", "deploy/docker/compose.yaml", "ps", "--status", "running", "-q", "lumen-hermes"}; !reflect.DeepEqual(r.calls[0], want) {
		t.Fatalf("status call=%#v, want %#v", r.calls[0], want)
	}
	r.calls = nil
	if _, err := (CommandSupervisor{Manager: SupervisorDocker}).Control(context.Background(), Action{Code: "restart"}, []ServiceName{ServiceHost}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"docker", "compose", "-f", "deploy/docker/compose.yaml", "restart", "lumen-host"}; !reflect.DeepEqual(r.calls[0], want) {
		t.Fatalf("restart call=%#v, want %#v", r.calls[0], want)
	}
}

func TestDockerSupervisorUsesConfiguredComposeProjectAndFiles(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &recordingRunner{}
	commandRunner = r
	s := CommandSupervisor{Manager: SupervisorDocker, Compose: ComposeConfig{
		Project: "lumen-check",
		Files:   []string{"deploy/docker/compose.yaml", "deploy/docker/compose.milestone1.yaml"},
	}}
	if _, err := s.Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHermes}); err != nil {
		t.Fatal(err)
	}
	want := []string{"docker", "compose", "-p", "lumen-check", "-f", "deploy/docker/compose.yaml", "-f", "deploy/docker/compose.milestone1.yaml", "ps", "--status", "running", "-q", "lumen-hermes"}
	if !reflect.DeepEqual(r.calls[0], want) {
		t.Fatalf("status call=%#v, want %#v", r.calls[0], want)
	}
}

func TestDockerSupervisorReadsComposeContextFromEnvironment(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &recordingRunner{}
	commandRunner = r
	t.Setenv("COMPOSE_PROJECT_NAME", "lumen-env-check")
	t.Setenv("COMPOSE_FILE", "deploy/docker/compose.yaml:deploy/docker/compose.milestone1.yaml")
	if _, err := (CommandSupervisor{Manager: SupervisorDocker}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHermes}); err != nil {
		t.Fatal(err)
	}
	want := []string{"docker", "compose", "-p", "lumen-env-check", "-f", "deploy/docker/compose.yaml", "-f", "deploy/docker/compose.milestone1.yaml", "ps", "--status", "running", "-q", "lumen-hermes"}
	if !reflect.DeepEqual(r.calls[0], want) {
		t.Fatalf("status call=%#v, want %#v", r.calls[0], want)
	}
}

func TestDockerSupervisorRejectsUnsafeComposeContext(t *testing.T) {
	for name, cfg := range map[string]ComposeConfig{
		"project separator": {Project: "lumen/check"},
		"project prefix":    {Project: "-lumen-check"},
		"path":              {Files: []string{"../outside.yaml"}},
	} {
		t.Run(name, func(t *testing.T) {
			old := commandRunner
			t.Cleanup(func() { commandRunner = old })
			r := &recordingRunner{}
			commandRunner = r
			_, err := (CommandSupervisor{Manager: SupervisorDocker, Compose: cfg}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost})
			if err == nil {
				t.Fatal("accepted unsafe Compose context")
			}
			if len(r.calls) != 0 {
				t.Fatalf("ran Docker command after rejecting context: %#v", r.calls)
			}
		})
	}
}

func TestDockerSupervisorBootStatusResolvesContainerThroughCompose(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &queuedOutputRunner{outputs: [][]byte{[]byte("container-id\n"), []byte("unless-stopped\n")}}
	commandRunner = r
	s := CommandSupervisor{Manager: SupervisorDocker, Compose: ComposeConfig{
		Project: "lumen-check",
		Files:   []string{"deploy/docker/compose.yaml", "deploy/docker/compose.milestone1.yaml"},
	}}
	ready, err := s.BootStatus(context.Background(), []ServiceName{ServiceHost})
	if err != nil || !ready {
		t.Fatalf("ready=%v err=%v calls=%#v", ready, err, r.calls)
	}
	want := [][]string{
		{"docker", "compose", "-p", "lumen-check", "-f", "deploy/docker/compose.yaml", "-f", "deploy/docker/compose.milestone1.yaml", "ps", "-q", "lumen-host"},
		{"docker", "inspect", "--format", "{{.HostConfig.RestartPolicy.Name}}", "container-id"},
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls=%#v, want %#v", r.calls, want)
	}
}

func TestSupervisorBootStatusUsesLiveEnableObservation(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &bootStatusRunner{}
	commandRunner = r
	ready, err := (CommandSupervisor{Manager: SupervisorSystemd}).BootStatus(context.Background(), []ServiceName{ServiceHost})
	if err != nil || !ready || !reflect.DeepEqual(r.calls, [][]string{{"systemctl", "is-enabled", "lumen-host.service"}}) {
		t.Fatalf("ready=%v err=%v calls=%#v", ready, err, r.calls)
	}
}

func TestSupervisorBootStatusUsesDockerRestartPolicy(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &queuedOutputRunner{outputs: [][]byte{[]byte("container-id\n"), []byte("unless-stopped\n")}}
	commandRunner = r
	ready, err := (CommandSupervisor{Manager: SupervisorDocker}).BootStatus(context.Background(), []ServiceName{ServiceHost})
	want := [][]string{{"docker", "compose", "-f", "deploy/docker/compose.yaml", "ps", "-q", "lumen-host"}, {"docker", "inspect", "--format", "{{.HostConfig.RestartPolicy.Name}}", "container-id"}}
	if err != nil || !ready || !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("ready=%v err=%v calls=%#v", ready, err, r.calls)
	}
}

func TestSupervisorBootStatusRejectsAmbiguousRunitBootState(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &bootStatusOutputRunner{stdout: []byte("run: lumen-host: (pid 1) 10s\n")}
	commandRunner = r
	ready, err := (CommandSupervisor{Manager: SupervisorRunit}).BootStatus(context.Background(), []ServiceName{ServiceHost})
	if err == nil || ready || len(r.calls) != 0 {
		t.Fatalf("ready=%v err=%v calls=%#v", ready, err, r.calls)
	}
}

func TestSupervisorBootStatusRejectsAmbiguousLaunchdBootState(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &bootStatusRunner{}
	commandRunner = r
	ready, err := (CommandSupervisor{Manager: SupervisorLaunchd}).BootStatus(context.Background(), []ServiceName{ServiceHost})
	if err == nil || ready || len(r.calls) != 0 {
		t.Fatalf("ready=%v err=%v calls=%#v", ready, err, r.calls)
	}
}

type bootStatusRunner struct{ calls [][]string }

func (r *bootStatusRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return []byte("enabled\n"), nil, nil
}

type bootStatusOutputRunner struct {
	calls  [][]string
	stdout []byte
}

func (r *bootStatusOutputRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return r.stdout, nil, nil
}

type queuedOutputRunner struct {
	calls   [][]string
	outputs [][]byte
}

func (r *queuedOutputRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if len(r.outputs) == 0 {
		return nil, nil, nil
	}
	out := r.outputs[0]
	r.outputs = r.outputs[1:]
	return out, nil, nil
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

func TestDefinitionInstallAcceptsTrackedPublicSystemdUnits(t *testing.T) {
	for _, name := range []string{"lumen-hermes.service", "lumen-host.service"} {
		src := filepath.Join("..", "..", "deploy", "systemd", name)
		dst := filepath.Join(t.TempDir(), name)
		if err := copyDefinition(src, dst); err != nil {
			t.Fatalf("copy tracked %s: %v", name, err)
		}
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

func TestInstallServicesDoesNotInitializeHostAfterHostInitialized(t *testing.T) {
	f := &fakeSupervisor{}
	initializer := &countingInitializer{}
	if err := InstallServices(context.Background(), f, ServicePlan{
		Host: setupDefinition(ServiceHost), Hermes: setupDefinition(ServiceHermes), Initializer: initializer,
	}); err != nil {
		t.Fatal(err)
	}
	if initializer.initializeCalls != 0 {
		t.Fatalf("InstallServices initialized Host %d times", initializer.initializeCalls)
	}
	if initializer.verifyCalls != 1 {
		t.Fatalf("InstallServices verify calls = %d, want 1", initializer.verifyCalls)
	}
}

func TestInstallServicesRequiresVerifierBeforeSupervisorCalls(t *testing.T) {
	f := &fakeSupervisor{}
	if err := InstallServices(context.Background(), f, ServicePlan{
		Host: setupDefinition(ServiceHost), Hermes: setupDefinition(ServiceHermes),
	}); err == nil {
		t.Fatal("accepted missing Host verifier")
	}
	if len(f.calls) != 0 {
		t.Fatalf("supervisor calls before verifier failure = %#v", f.calls)
	}
}

func setupDefinition(name ServiceName) ServiceDefinition { return ServiceDefinition{Name: name} }

type countingInitializer struct{ initializeCalls, verifyCalls int }

func (i *countingInitializer) Initialize(context.Context) error {
	i.initializeCalls++
	return nil
}
func (i *countingInitializer) Verify(context.Context) error {
	i.verifyCalls++
	return nil
}

type fakeInitializer struct{}

func (fakeInitializer) Initialize(context.Context) error { return nil }
func (fakeInitializer) Verify(context.Context) error     { return nil }
