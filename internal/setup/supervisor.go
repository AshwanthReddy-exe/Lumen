package setup

import (
	"context"
	"errors"
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
type ServicePlan struct{ Hermes, Host ServiceDefinition }
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
	if err := s.Install(ctx, p); err != nil {
		return err
	}
	return s.Enable(ctx, []ServiceName{ServiceHermes, ServiceHost})
}

type CommandSupervisor struct{ Manager Supervisor }

func (s CommandSupervisor) Install(ctx context.Context, p ServicePlan) error {
	return s.run(ctx, "install", p.Hermes, p.Host)
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
	for _, n := range names {
		if err := s.run(ctx, a.Code, ServiceDefinition{Name: n}); err != nil {
			return nil, err
		}
	}
	return nil, nil
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
