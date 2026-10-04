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

func TestLaunchdStatusUsesOnlyTheStateField(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	commandRunner = &sequenceRunner{steps: []runnerStep{{stdout: []byte("state = waiting\npath = /Users/alice/running/lumen-host\n")}}}
	got, err := (CommandSupervisor{Manager: SupervisorLaunchd}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost})
	if err != nil || got[0].State != StateStopped {
		t.Fatalf("stopped LaunchAgent with 'running' in path: got %#v, err %v", got, err)
	}

	commandRunner = &sequenceRunner{steps: []runnerStep{{stdout: []byte("state = running\npath = /Users/alice/lumen-host\n")}}}
	got, err = (CommandSupervisor{Manager: SupervisorLaunchd}).Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost})
	if err != nil || got[0].State != StateRunning {
		t.Fatalf("running LaunchAgent: got %#v, err %v", got, err)
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

func TestDockerExternalSupervisorForcesStandaloneComposeFile(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &recordingRunner{}
	commandRunner = r
	t.Setenv("COMPOSE_FILE", "deploy/docker/compose.yaml")
	s := CommandSupervisor{Manager: SupervisorDocker, Topology: TopologyExternal}
	if _, err := s.Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost}); err != nil {
		t.Fatal(err)
	}
	want := []string{"docker", "compose", "-f", "deploy/docker/compose.external.yaml", "ps", "--status", "running", "-q", "lumen-host"}
	if !reflect.DeepEqual(r.calls[0], want) {
		t.Fatalf("status call=%#v, want %#v", r.calls[0], want)
	}
}

func TestDockerExternalSupervisorRejectsCombinedComposeFile(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &recordingRunner{}
	commandRunner = r
	s := CommandSupervisor{Manager: SupervisorDocker, Topology: TopologyExternal, Compose: ComposeConfig{
		Files: []string{"deploy/docker/compose.yaml"},
	}}
	if _, err := s.Control(context.Background(), Action{Code: "status"}, []ServiceName{ServiceHost}); err == nil {
		t.Fatal("accepted combined Compose file for external topology")
	}
	if len(r.calls) != 0 {
		t.Fatalf("ran Docker command after rejecting combined Compose file: %#v", r.calls)
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

func TestDockerSupervisorBootStatusRejectsOnFailurePolicy(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &queuedOutputRunner{outputs: [][]byte{[]byte("container-id\n"), []byte("on-failure\n")}}
	commandRunner = r

	ready, err := (CommandSupervisor{Manager: SupervisorDocker}).BootStatus(context.Background(), []ServiceName{ServiceHost})
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("on-failure does not restart a container after the Docker daemon restarts")
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

func TestSupervisorBootStatusRequiresExactLaunchdEnabledEntry(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	for _, tc := range []struct {
		output string
		ready  bool
	}{
		{"disabled services = {\n\t\"dev.lumen.host\" => enabled\n}", true},
		{"disabled services = {\n\t\"dev.lumen.host\" => disabled\n}", false},
		{"disabled services = {\n\t\"dev.lumen.host-old\" => enabled\n}", false},
	} {
		r := &queuedOutputRunner{outputs: [][]byte{[]byte(tc.output)}}
		commandRunner = r
		ready, err := (CommandSupervisor{Manager: SupervisorLaunchd}).BootStatus(context.Background(), []ServiceName{ServiceHost})
		if err != nil || ready != tc.ready || !reflect.DeepEqual(r.calls, [][]string{{"launchctl", "print-disabled", launchdDomain()}}) {
			t.Fatalf("output=%q ready=%v err=%v calls=%#v", tc.output, ready, err, r.calls)
		}
	}
}

func TestLaunchdBootstrapResumesOnlyTheSameDefinition(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	path := filepath.Join(t.TempDir(), "dev.lumen.host.plist")
	if err := os.WriteFile(path, []byte("plist"), 0600); err != nil {
		t.Fatal(err)
	}
	target := launchdTarget(ServiceHost)
	t.Run("same path reloads the verified definition", func(t *testing.T) {
		r := &sequenceRunner{steps: []runnerStep{{stdout: []byte("path = " + path)}, {}, {}}}
		commandRunner = r
		if err := launchdBootstrap(context.Background(), ServiceDefinition{Name: ServiceHost, Path: path}); err != nil || len(r.calls) != 3 {
			t.Fatalf("err=%v calls=%#v", err, r.calls)
		}
		want := [][]string{{"launchctl", "print", target}, {"launchctl", "bootout", target}, {"launchctl", "bootstrap", launchdDomain(), path}}
		if !reflect.DeepEqual(r.calls, want) {
			t.Fatalf("calls=%#v want=%#v", r.calls, want)
		}
	})
	t.Run("same path reload failure is not success", func(t *testing.T) {
		r := &sequenceRunner{steps: []runnerStep{{stdout: []byte("path = " + path)}, {}, {err: errors.New("bootstrap failed")}}}
		commandRunner = r
		if err := launchdBootstrap(context.Background(), ServiceDefinition{Name: ServiceHost, Path: path}); err == nil || len(r.calls) != 3 {
			t.Fatalf("err=%v calls=%#v", err, r.calls)
		}
	})
	t.Run("already loaded from the committed auto-load path", func(t *testing.T) {
		alternate := filepath.Join(t.TempDir(), "dev.lumen.host.plist")
		if err := os.WriteFile(alternate, []byte("committed plist"), 0600); err != nil {
			t.Fatal(err)
		}
		r := &sequenceRunner{steps: []runnerStep{{stdout: []byte("path = " + alternate)}, {}, {}}}
		commandRunner = r
		definition := ServiceDefinition{Name: ServiceHost, Path: path, AlternatePath: alternate}
		if err := launchdBootstrap(context.Background(), definition); err != nil || len(r.calls) != 3 {
			t.Fatalf("err=%v calls=%#v", err, r.calls)
		}
		want := [][]string{{"launchctl", "print", target}, {"launchctl", "bootout", target}, {"launchctl", "bootstrap", launchdDomain(), path}}
		if !reflect.DeepEqual(r.calls, want) {
			t.Fatalf("calls=%#v want=%#v", r.calls, want)
		}
	})
	t.Run("bootout uncertainty fails closed", func(t *testing.T) {
		alternate := filepath.Join(t.TempDir(), "staged.plist")
		if err := os.WriteFile(alternate, []byte("staged"), 0600); err != nil {
			t.Fatal(err)
		}
		r := &sequenceRunner{steps: []runnerStep{{stdout: []byte("path = " + alternate)}, {err: errors.New("timeout")}}}
		commandRunner = r
		if err := launchdBootstrap(context.Background(), ServiceDefinition{Name: ServiceHost, Path: path, AlternatePath: alternate}); err == nil || len(r.calls) != 2 {
			t.Fatalf("err=%v calls=%#v", err, r.calls)
		}
	})
	t.Run("failed bootstrap is not accepted from a matching path", func(t *testing.T) {
		r := &sequenceRunner{steps: []runnerStep{
			{err: errors.New("print unavailable")},
			{err: errors.New("bootstrap timeout")},
		}}
		commandRunner = r
		if err := launchdBootstrap(context.Background(), ServiceDefinition{Name: ServiceHost, Path: path}); err == nil || len(r.calls) != 2 {
			t.Fatalf("err=%v calls=%#v", err, r.calls)
		}
	})
	t.Run("failed published transition retries from no loaded job", func(t *testing.T) {
		published := filepath.Join(t.TempDir(), "published.plist")
		if err := os.WriteFile(published, []byte("published"), 0600); err != nil {
			t.Fatal(err)
		}
		stage := filepath.Join(t.TempDir(), "stage.plist")
		if err := os.WriteFile(stage, []byte("stage"), 0600); err != nil {
			t.Fatal(err)
		}
		definition := ServiceDefinition{Name: ServiceHost, Path: published, AlternatePath: stage}
		r := &sequenceRunner{steps: []runnerStep{
			{stdout: []byte("path = " + stage)}, {}, {err: errors.New("bootstrap interrupted")},
			{err: errors.New("not loaded")}, {},
		}}
		commandRunner = r
		if err := launchdBootstrap(context.Background(), definition); err == nil {
			t.Fatal("failed staged-to-published transition reported success")
		}
		if err := launchdBootstrap(context.Background(), definition); err != nil {
			t.Fatalf("retry after interrupted transition: %v", err)
		}
		want := [][]string{
			{"launchctl", "print", target}, {"launchctl", "bootout", target}, {"launchctl", "bootstrap", launchdDomain(), published},
			{"launchctl", "print", target}, {"launchctl", "bootstrap", launchdDomain(), published},
		}
		if !reflect.DeepEqual(r.calls, want) {
			t.Fatalf("calls=%#v want=%#v", r.calls, want)
		}
	})
	t.Run("collision is never booted out or replaced", func(t *testing.T) {
		r := &sequenceRunner{steps: []runnerStep{{stdout: []byte("path = /other/dev.lumen.host.plist")}}}
		commandRunner = r
		if err := launchdBootstrap(context.Background(), ServiceDefinition{Name: ServiceHost, Path: path}); err == nil || len(r.calls) != 1 {
			t.Fatalf("err=%v calls=%#v", err, r.calls)
		}
	})
	t.Run("new registration is user scoped", func(t *testing.T) {
		r := &sequenceRunner{steps: []runnerStep{{err: errors.New("not loaded")}, {}}}
		commandRunner = r
		if err := launchdBootstrap(context.Background(), ServiceDefinition{Name: ServiceHost, Path: path}); err != nil {
			t.Fatal(err)
		}
		want := [][]string{{"launchctl", "print", target}, {"launchctl", "bootstrap", launchdDomain(), path}}
		if !reflect.DeepEqual(r.calls, want) {
			t.Fatalf("calls=%#v want=%#v", r.calls, want)
		}
	})
}

func TestLaunchdLifecycleUsesCurrentUserDomain(t *testing.T) {
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	r := &queuedOutputRunner{outputs: [][]byte{[]byte("state = running"), []byte("state = running"), []byte("state = running")}}
	commandRunner = r
	s := CommandSupervisor{Manager: SupervisorLaunchd}
	for _, action := range []string{"start", "stop", "restart"} {
		if _, err := s.Control(context.Background(), Action{Code: action}, []ServiceName{ServiceHost}); err != nil {
			t.Fatalf("%s: %v", action, err)
		}
	}
	target := launchdTarget(ServiceHost)
	want := [][]string{
		{"launchctl", "kickstart", target}, {"launchctl", "print", target},
		{"launchctl", "kill", "SIGTERM", target}, {"launchctl", "print", target},
		{"launchctl", "kickstart", "-k", target}, {"launchctl", "print", target},
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Fatalf("calls=%#v want=%#v", r.calls, want)
	}
}

type runnerStep struct {
	stdout []byte
	err    error
}

type sequenceRunner struct {
	calls [][]string
	steps []runnerStep
}

func (r *sequenceRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	step := runnerStep{}
	if len(r.steps) > 0 {
		step, r.steps = r.steps[0], r.steps[1:]
	}
	return step.stdout, nil, step.err
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

func TestDefinitionInstallAcceptsTrackedPublicSystemdUnitContents(t *testing.T) {
	for _, name := range []string{"lumen-hermes.service", "lumen-host.service"} {
		tracked := filepath.Join("..", "..", "deploy", "systemd", name)
		body, err := os.ReadFile(tracked)
		if err != nil {
			t.Fatal(err)
		}
		src := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(src, body, 0600); err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(t.TempDir(), name)
		if err := copyDefinition(src, dst); err != nil {
			t.Fatalf("copy tracked %s contents: %v", name, err)
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
