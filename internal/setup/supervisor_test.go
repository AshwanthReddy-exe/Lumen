package setup

import (
	"context"
	"reflect"
	"testing"
)

type fakeSupervisor struct{ calls []string }

func (f *fakeSupervisor) Install(context.Context, ServicePlan) error {
	f.calls = append(f.calls, "install:hermes", "install:host")
	return nil
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
	if err := InstallServices(context.Background(), f, ServicePlan{Host: ServiceDefinition{Name: ServiceHost}, Hermes: ServiceDefinition{Name: ServiceHermes}}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.calls, []string{"install:hermes", "install:host", "enable:hermes", "enable:host"}) {
		t.Fatalf("got %#v", f.calls)
	}
}
