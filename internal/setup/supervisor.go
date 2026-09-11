package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ServiceName string

const (
	ServiceHermes ServiceName = "hermes"
	ServiceHost   ServiceName = "host"
)

type ServiceDefinition struct {
	Name        ServiceName
	Path        string
	Source      string
	Destination string
}
type ServicePlan struct {
	Hermes, Host ServiceDefinition
	// Deprecated fields retained for source compatibility; initialization must use Initializer.
	HostInitialized bool
	Verify          func() error
	Initializer     HostInitializer
}
type HostInitializer interface {
	Initialize(context.Context) error
	Verify(context.Context) error
}
type ServiceState struct {
	Name  ServiceName
	State string
}

func (s ServiceState) Validate() error {
	switch s.State {
	case StateRunning, StateStopped, StateFailed, StateUnknown:
		return nil
	}
	return fmt.Errorf("invalid service state %q", s.State)
}

const (
	StateRunning = "running"
	StateStopped = "stopped"
	StateFailed  = "failed"
	StateUnknown = "unknown"
)

type ActionRequiredError struct {
	Manager   Supervisor
	Service   ServiceName
	Operation string
}

type ValidationError struct{ Field, Value string }

func (e *ValidationError) Error() string { return fmt.Sprintf("invalid %s %q", e.Field, e.Value) }
func validateName(n ServiceName) error {
	if n != ServiceHermes && n != ServiceHost {
		return &ValidationError{"service", string(n)}
	}
	return nil
}
func validateAction(a Action) error {
	switch a.Code {
	case "start", "stop", "restart", "status":
		return nil
	}
	return &ValidationError{"action", a.Code}
}

func (e *ActionRequiredError) Error() string {
	return fmt.Sprintf("action_required: %s %s on %s", e.Manager, e.Operation, e.Service)
}

type SupervisorAPI interface {
	Install(context.Context, ServicePlan) error
	Enable(context.Context, []ServiceName) error
	Control(context.Context, Action, []ServiceName) ([]ServiceState, error)
}

func InstallServices(ctx context.Context, s SupervisorAPI, p ServicePlan) error {
	if s == nil {
		return &ValidationError{"supervisor", "nil"}
	}
	if err := validateName(p.Hermes.Name); err != nil {
		return err
	}
	if err := validateName(p.Host.Name); err != nil {
		return err
	}
	if p.Initializer == nil {
		return errors.New("host initialization not verified")
	}
	if p.Initializer != nil {
		if err := p.Initializer.Initialize(ctx); err != nil {
			return fmt.Errorf("initialize Host: %w", err)
		}
		if err := p.Initializer.Verify(ctx); err != nil {
			return fmt.Errorf("verify Host state: %w", err)
		}
	}
	if err := s.Install(ctx, p); err != nil {
		return err
	}
	return s.Enable(ctx, []ServiceName{ServiceHermes, ServiceHost})
}

type CommandSupervisor struct{ Manager Supervisor }

type CommandRunner interface {
	Run(context.Context, string, ...string) (stdout, stderr []byte, err error)
}
type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	c := exec.CommandContext(ctx, name, args...)
	var ob, eb bytes.Buffer
	c.Stdout = &ob
	c.Stderr = &eb
	err := c.Run()
	return ob.Bytes(), eb.Bytes(), err
}

var commandRunner CommandRunner = execRunner{}

func (s CommandSupervisor) Install(ctx context.Context, p ServicePlan) error {
	for _, d := range []ServiceDefinition{p.Hermes, p.Host} {
		if d.Path == "" && (d.Source == "" || d.Destination == "") {
			return errors.New("service definition path required")
		}
		if err := validateName(d.Name); err != nil {
			return err
		}
		if d.Source != "" {
			if err := copyDefinition(d.Source, d.Destination); err != nil {
				return fmt.Errorf("manager=%s service=%s operation=install: %w", s.Manager, d.Name, err)
			}
			d.Path = d.Destination
		}
		var err error
		switch s.Manager {
		case SupervisorSystemd:
			_, _, err = s.call(ctx, d.Name, "install", "systemctl", "daemon-reload")
		case SupervisorLaunchd:
			err = launchdBootstrap(ctx, d)
		case SupervisorDocker:
			// compose definitions are registered at enable time.
		case SupervisorRunit:
			// placement is installation; start occurs in Enable.
		default:
			return &ActionRequiredError{Manager: s.Manager, Service: d.Name, Operation: "install"}
		}
		if err != nil {
			return fmt.Errorf("manager=%s service=%s operation=install: %w", s.Manager, d.Name, err)
		}
	}
	return nil
}
func copyDefinition(src, dst string) error {
	st, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&0077 != 0 {
		return errors.New("unsafe definition")
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err == nil {
		err = out.Sync()
	}
	out.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
func (s CommandSupervisor) Enable(ctx context.Context, names []ServiceName) error {
	for _, n := range names {
		if err := validateName(n); err != nil {
			return err
		}
		if err := s.run(ctx, "enable", ServiceDefinition{Name: n}); err != nil {
			return fmt.Errorf("enable %s: %w", n, err)
		}
	}
	return nil
}
func (s CommandSupervisor) Control(ctx context.Context, a Action, names []ServiceName) ([]ServiceState, error) {
	if err := validateAction(a); err != nil {
		return nil, err
	}
	states := make([]ServiceState, 0, len(names))
	timeout := ctx
	if a.Code == "stop" || a.Code == "restart" {
		var cancel context.CancelFunc
		timeout, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}
	for _, n := range names {
		if err := validateName(n); err != nil {
			return nil, err
		}
		if a.Code != "status" {
			if err := s.run(timeout, a.Code, ServiceDefinition{Name: n}); err != nil {
				return nil, fmt.Errorf("manager=%s service=%s operation=%s: %w", s.Manager, n, a.Code, err)
			}
		}
		st, err := s.observe(timeout, n)
		if err != nil {
			return nil, err
		}
		states = append(states, st)
	}
	return states, nil
}
func unitName(d ServiceDefinition) string {
	if d.Path != "" {
		return d.Path
	}
	return "lumen-" + string(d.Name) + ".service"
}
func launchdBootstrap(ctx context.Context, d ServiceDefinition) error {
	domain := "system"
	if strings.HasPrefix(d.Path, "gui/") {
		domain = "gui/" + strings.TrimPrefix(d.Path, "gui/")
	}
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, _, err := commandRunner.Run(bounded, "launchctl", "bootstrap", domain, unitName(d))
	return err
}
func (s CommandSupervisor) observe(ctx context.Context, n ServiceName) (ServiceState, error) {
	out, errout, err := s.runOutput(ctx, "status", ServiceDefinition{Name: n})
	text := strings.ToLower(string(out) + " " + string(errout))
	st := StateUnknown
	if strings.Contains(text, "running") || strings.Contains(text, "active (running)") {
		st = StateRunning
	}
	if strings.Contains(text, "stopped") || strings.Contains(text, "inactive") {
		st = StateStopped
	}
	if strings.Contains(text, "failed") || strings.Contains(text, "error") {
		st = StateFailed
	}
	if err != nil {
		return ServiceState{}, fmt.Errorf("manager=%s service=%s operation=status: %w", s.Manager, n, err)
	}
	return ServiceState{Name: n, State: st}, nil
}
func (s CommandSupervisor) run(ctx context.Context, op string, defs ...ServiceDefinition) error {
	for _, d := range defs {
		var name string
		var args []string
		switch s.Manager {
		case SupervisorSystemd:
			name = "systemctl"
			args = []string{op, unitName(d)}
		case SupervisorLaunchd:
			name = "launchctl"
			args = []string{"kickstart", "system/dev.lumen." + string(d.Name)}
			if op == "stop" {
				args = []string{"kill", "SIGTERM", "system/dev.lumen." + string(d.Name)}
			}
		case SupervisorDocker:
			name = "docker"
			actual := op
			if op == "enable" {
				actual = "up"
			}
			args = []string{"compose", "-f", "deploy/docker/compose.yaml", actual, "-d", "lumen-" + string(d.Name)}
		case SupervisorRunit:
			name = "sv"
			if op == "enable" {
				return nil
			}
			args = []string{op, "lumen-" + string(d.Name)}
		default:
			return &ActionRequiredError{Manager: s.Manager, Service: d.Name, Operation: op}
		}
		if _, _, err := s.call(ctx, d.Name, op, name, args...); err != nil {
			return fmt.Errorf("manager=%s service=%s operation=%s: %w", s.Manager, d.Name, op, err)
		}
	}
	return nil
}
func (s CommandSupervisor) call(ctx context.Context, service ServiceName, op, name string, args ...string) ([]byte, []byte, error) {
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, errout, err := commandRunner.Run(bounded, name, args...)
	if err != nil {
		return out, errout, fmt.Errorf("manager=%s service=%s operation=%s: %w", s.Manager, service, op, err)
	}
	return out, errout, nil
}
func (s CommandSupervisor) runOutput(ctx context.Context, op string, d ServiceDefinition) ([]byte, []byte, error) {
	var name string
	var args []string
	switch s.Manager {
	case SupervisorSystemd:
		name = "systemctl"
		args = []string{"status", unitName(d), "--no-pager"}
	case SupervisorLaunchd:
		name = "launchctl"
		args = []string{"print", "system/" + "dev.lumen." + string(d.Name)}
	case SupervisorDocker:
		name = "docker"
		args = []string{"compose", "-f", "deploy/docker/compose.yaml", "ps", string(d.Name)}
	case SupervisorRunit:
		name = "sv"
		args = []string{"status", "lumen-" + string(d.Name)}
	default:
		return nil, nil, &ActionRequiredError{Manager: s.Manager, Service: d.Name, Operation: op}
	}
	return s.call(ctx, d.Name, op, name, args...)
}
