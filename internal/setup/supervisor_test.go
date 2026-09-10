package setup

import (
	"context"
	"os"
	"reflect"
	"testing"
)

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
	if err := s.Install(context.Background(), ServicePlan{Hermes: ServiceDefinition{Name: ServiceHermes, Source: src, Destination: dst}, Host: ServiceDefinition{Name: ServiceHost, Path: dst}, HostInitialized: true, Verify: func() error { return nil }}); err != nil { /* docker may be absent; copy is still asserted */
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
	if err := InstallServices(context.Background(), f, ServicePlan{Host: ServiceDefinition{Name: ServiceHost}, Hermes: ServiceDefinition{Name: ServiceHermes}, HostInitialized: true, Verify: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.calls, []string{"install:hermes", "install:host", "enable:hermes", "enable:host"}) {
		t.Fatalf("got %#v", f.calls)
	}
}
