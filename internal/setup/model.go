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

type Action struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

type Report struct {
	Outcome          Outcome  `json:"outcome"`
	Stage            Stage    `json:"stage"`
	Actions          []Action `json:"actions,omitempty"`
	AvailableInBlock int      `json:"availableInBlock,omitempty"`
}
