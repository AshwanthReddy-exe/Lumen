package setup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

type ServiceName string

const (
	ServiceHermes ServiceName = "hermes"
	ServiceHost   ServiceName = "host"
)

type ServiceDefinition struct {
	Name ServiceName
	Path string
}
type ServicePlan struct {
	Hermes, Host    ServiceDefinition
	HostInitialized bool
	Verify          func() error
}
type ServiceState struct {
	Name  ServiceName
	State string
}
type SupervisorAPI interface {
	Install(context.Context, ServicePlan) error
	Enable(context.Context, []ServiceName) error
	Control(context.Context, Action, []ServiceName) ([]ServiceState, error)
}

func InstallServices(ctx context.Context, s SupervisorAPI, p ServicePlan) error {
	if s == nil || p.Hermes.Name != ServiceHermes || p.Host.Name != ServiceHost {
		return errors.New("invalid service plan")
	}
	if !p.HostInitialized || p.Verify == nil {
		return errors.New("host initialization not verified")
	}
	if err := p.Verify(); err != nil {
		return fmt.Errorf("verify Host state: %w", err)
	}
	if err := s.Install(ctx, p); err != nil {
		return err
	}
	return s.Enable(ctx, []ServiceName{ServiceHermes, ServiceHost})
}

type CommandSupervisor struct{ Manager Supervisor }

func (s CommandSupervisor) Install(ctx context.Context, p ServicePlan) error {
	for _, d := range []ServiceDefinition{p.Hermes, p.Host} {
		if d.Path == "" {
			return errors.New("service definition path required")
		}
		var err error
		switch s.Manager {
		case SupervisorSystemd:
			err = exec.CommandContext(ctx, "systemctl", "enable", d.Path).Run()
		case SupervisorLaunchd:
			err = exec.CommandContext(ctx, "launchctl", "bootstrap", d.Path).Run()
		case SupervisorDocker:
			err = exec.CommandContext(ctx, "docker", "compose", "-f", d.Path, "config").Run()
		case SupervisorRunit:
			err = exec.CommandContext(ctx, "sv", "up", d.Path).Run()
		default:
			return errors.New("action_required: supervisor unavailable")
		}
		if err != nil {
			return fmt.Errorf("install %s: %w", d.Name, err)
		}
	}
	return nil
}
func (s CommandSupervisor) Enable(ctx context.Context, names []ServiceName) error {
	for _, n := range names {
		if err := s.run(ctx, "enable", ServiceDefinition{Name: n}); err != nil {
			return err
		}
	}
	return nil
}
func (s CommandSupervisor) Control(ctx context.Context, a Action, names []ServiceName) ([]ServiceState, error) {
	if a.Code != "start" && a.Code != "stop" && a.Code != "restart" && a.Code != "status" {
		return nil, errors.New("invalid service action")
	}
	states := make([]ServiceState, 0, len(names))
	for _, n := range names {
		if n != ServiceHermes && n != ServiceHost {
			return nil, errors.New("invalid service name")
		}
		if err := s.run(ctx, a.Code, ServiceDefinition{Name: n}); err != nil {
			return nil, fmt.Errorf("%s %s: %w", a.Code, n, err)
		}
		states = append(states, ServiceState{Name: n, State: a.Code})
	}
	return states, nil
}
func (s CommandSupervisor) run(ctx context.Context, op string, defs ...ServiceDefinition) error {
	for _, d := range defs {
		var name string
		var args []string
		switch s.Manager {
		case SupervisorSystemd:
			name = "systemctl"
			args = []string{op, "lumen-" + string(d.Name) + ".service"}
		case SupervisorLaunchd:
			name = "launchctl"
			args = []string{op, "dev.lumen." + string(d.Name) + ".plist"}
		case SupervisorDocker:
			name = "docker"
			args = []string{"compose", op, "lumen-" + string(d.Name)}
		case SupervisorRunit:
			name = "sv"
			args = []string{op, "lumen-" + string(d.Name)}
		default:
			return errors.New("supervisor unavailable")
		}
		if err := exec.CommandContext(ctx, name, args...).Run(); err != nil {
			return err
		}
	}
	return nil
}
