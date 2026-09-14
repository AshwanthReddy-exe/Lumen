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
	// Services is the exact locally owned service set. Empty preserves the
	// original combined topology for callers that predate topology support.
	Services []ServiceDefinition
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
	services := p.serviceDefinitions()
	if len(services) == 0 {
		return &ValidationError{"services", "empty"}
	}
	for _, service := range services {
		if err := validateName(service.Name); err != nil {
			return err
		}
	}
	if p.Initializer == nil {
		return errors.New("host initialization not verified")
	}
	if err := p.Initializer.Verify(ctx); err != nil {
		return fmt.Errorf("verify Host state: %w", err)
	}
	if err := s.Install(ctx, p); err != nil {
		return err
	}
	names := make([]ServiceName, 0, len(services))
	for _, service := range services {
		names = append(names, service.Name)
	}
	return s.Enable(ctx, names)
}

func (p ServicePlan) serviceDefinitions() []ServiceDefinition {
	if len(p.Services) != 0 {
		return p.Services
	}
	return []ServiceDefinition{p.Hermes, p.Host}
}

// ComposeConfig identifies the project and file set used by a Docker
// supervisor. An empty value uses the repository's base Compose file and the
// default Compose project selected by Docker.
type ComposeConfig struct {
	Project string
	Files   []string
}

func validateComposeBinding(supervisor Supervisor, project string) error {
	if project == "" {
		return nil
	}
	if supervisor != SupervisorDocker {
		return &ValidationError{"compose project", project}
	}
	return validateComposeProject(project)
}

type CommandSupervisor struct {
	Manager  Supervisor
	Topology Topology
	Compose  ComposeConfig
}

const defaultComposeFile = "deploy/docker/compose.yaml"
const externalComposeFile = "deploy/docker/compose.external.yaml"

func (s CommandSupervisor) composeArgs(operation string, services ...string) ([]string, error) {
	c := s.Compose
	if c.Project == "" {
		c.Project = os.Getenv("COMPOSE_PROJECT_NAME")
	}
	if s.Topology == TopologyExternal {
		if len(c.Files) == 0 {
			c.Files = []string{externalComposeFile}
		} else if len(c.Files) != 1 || c.Files[0] != externalComposeFile {
			return nil, &ValidationError{"compose file", strings.Join(c.Files, string(os.PathListSeparator))}
		}
	}
	if len(c.Files) == 0 {
		if raw := os.Getenv("COMPOSE_FILE"); raw != "" {
			c.Files = strings.Split(raw, string(os.PathListSeparator))
		} else {
			c.Files = []string{defaultComposeFile}
		}
	}
	if err := validateComposeProject(c.Project); err != nil {
		return nil, err
	}
	args := []string{"compose"}
	if c.Project != "" {
		args = append(args, "-p", c.Project)
	}
	for _, file := range c.Files {
		if err := validateComposeFile(file); err != nil {
			return nil, err
		}
		args = append(args, "-f", file)
	}
	args = append(args, operation)
	args = append(args, services...)
	return args, nil
}

func validateComposeProject(project string) error {
	if project == "" {
		return nil
	}
	if len(project) > 63 {
		return &ValidationError{"compose project", project}
	}
	for i, r := range project {
		lowerOrDigit := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if i == 0 && !lowerOrDigit {
			return &ValidationError{"compose project", project}
		}
		if !lowerOrDigit && r != '_' && r != '-' {
			return &ValidationError{"compose project", project}
		}
	}
	return nil
}

func validateComposeFile(file string) error {
	if file == "" || filepath.IsAbs(file) || filepath.Clean(file) != file || file == "." {
		return &ValidationError{"compose file", file}
	}
	for _, part := range strings.Split(file, string(filepath.Separator)) {
		if part == ".." {
			return &ValidationError{"compose file", file}
		}
	}
	return nil
}

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
	for _, d := range p.serviceDefinitions() {
		if s.Manager != SupervisorDocker && d.Path == "" && (d.Source == "" || d.Destination == "") {
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
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || st.Mode()&0022 != 0 {
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

// BootStatus observes whether the supervisor has enabled every selected
// service. Managers without a stable enabled-state query fail closed instead
// of inferring boot readiness from service status or setup history.
func (s CommandSupervisor) BootStatus(ctx context.Context, names []ServiceName) (bool, error) {
	if err := s.Manager.Validate(); err != nil {
		return false, err
	}
	if len(names) == 0 {
		return false, &ValidationError{"services", "empty"}
	}
	if s.Manager == SupervisorRunit {
		return false, &ActionRequiredError{Manager: s.Manager, Service: names[0], Operation: "boot-status"}
	}
	if s.Manager == SupervisorDocker {
		for _, name := range names {
			compose, err := s.composeArgs("ps", "-q", "lumen-"+string(name))
			if err != nil {
				return false, err
			}
			out, _, err := s.call(ctx, name, "boot-status", "docker", compose...)
			if err != nil {
				return false, err
			}
			ids := strings.Fields(string(out))
			if len(ids) != 1 {
				return false, nil
			}
			out, _, err = s.call(ctx, name, "boot-status", "docker", "inspect", "--format", "{{.HostConfig.RestartPolicy.Name}}", ids[0])
			if err != nil {
				return false, err
			}
			policy := strings.TrimSpace(string(out))
			if policy != "always" && policy != "unless-stopped" {
				return false, nil
			}
		}
		return true, nil
	}
	if s.Manager == SupervisorLaunchd {
		return false, &ActionRequiredError{Manager: s.Manager, Service: names[0], Operation: "boot-status"}
	}
	if s.Manager != SupervisorSystemd {
		return false, &ActionRequiredError{Manager: s.Manager, Service: names[0], Operation: "boot-status"}
	}
	for _, name := range names {
		if err := validateName(name); err != nil {
			return false, err
		}
		out, _, err := s.call(ctx, name, "is-enabled", "systemctl", "is-enabled", unitName(ServiceDefinition{Name: name}))
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(string(out)) != "enabled" {
			return false, nil
		}
	}
	return true, nil
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
	if s.Manager == SupervisorDocker {
		if err != nil {
			return ServiceState{}, fmt.Errorf("manager=%s service=%s operation=status: %w", s.Manager, n, err)
		}
		if strings.TrimSpace(string(out)) != "" {
			return ServiceState{Name: n, State: StateRunning}, nil
		}
		return ServiceState{Name: n, State: StateStopped}, nil
	}
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
		var err error
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
			args, err = s.composeArgs(actual, "lumen-"+string(d.Name))
			if err != nil {
				return err
			}
			if op == "enable" {
				args, err = s.composeArgs("up", "-d", "lumen-"+string(d.Name))
				if err != nil {
					return err
				}
			}
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
		var err error
		args, err = s.composeArgs("ps", "--status", "running", "-q", "lumen-"+string(d.Name))
		if err != nil {
			return nil, nil, err
		}
	case SupervisorRunit:
		name = "sv"
		args = []string{"status", "lumen-" + string(d.Name)}
	default:
		return nil, nil, &ActionRequiredError{Manager: s.Manager, Service: d.Name, Operation: op}
	}
	return s.call(ctx, d.Name, op, name, args...)
}
