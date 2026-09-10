package setup

import "fmt"

type Outcome string

const (
	Ready          Outcome = "ready"
	Degraded       Outcome = "degraded"
	ActionRequired Outcome = "action_required"
)

func (o Outcome) Validate() error {
	switch o {
	case Ready, Degraded, ActionRequired:
		return nil
	default:
		return fmt.Errorf("invalid outcome %q", o)
	}
}

type Stage string

const (
	Detected           Stage = "detected"
	ArtifactsReady     Stage = "artifacts_ready"
	DirectoriesReady   Stage = "directories_ready"
	CredentialsReady   Stage = "credentials_ready"
	ConfigurationReady Stage = "configuration_ready"
	HostInitialized    Stage = "host_initialized"
	ServicesInstalled  Stage = "services_installed"
	ServicesStarted    Stage = "services_started"
	Validated          Stage = "validated"
)

type Profile string

const (
	Development   Profile = "development"
	PersonalAlpha Profile = "personal-alpha"
	Hardened      Profile = "hardened"
)

type Request struct {
	Profile Profile `json:"profile"`
}

type Platform string

const (
	PlatformMacOS  Platform = "macos"
	PlatformLinux  Platform = "linux"
	PlatformTermux Platform = "android-termux"
)

type Supervisor string

const (
	SupervisorLaunchd Supervisor = "launchd"
	SupervisorSystemd Supervisor = "systemd"
	SupervisorRunit   Supervisor = "runit"
	SupervisorDocker  Supervisor = "docker"
)

type StageEvidence struct {
	Stage       Stage  `json:"stage"`
	InputDigest string `json:"inputDigest"`
	CompletedAt int64  `json:"completedAt"`
}

type Action struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

type Report struct {
	Outcome          Outcome           `json:"outcome"`
	Stage            Stage             `json:"stage"`
	Profile          Profile           `json:"profile,omitempty"`
	Platform         Platform          `json:"platform,omitempty"`
	LumenVersion     string            `json:"lumenVersion,omitempty"`
	HermesVersion    string            `json:"hermesVersion,omitempty"`
	States           map[string]string `json:"states,omitempty"`
	Actions          []Action          `json:"actions,omitempty"`
	AvailableInBlock int               `json:"availableInBlock,omitempty"`
}
