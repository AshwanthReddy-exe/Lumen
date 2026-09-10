package setup

import (
	"context"
	"errors"
	"fmt"
)

var ErrIsolationUnavailable = errors.New("hardened isolation unavailable")

type PlanResult struct {
	Profile             Profile
	Platform            Platform
	Architecture        string
	Supervisor          Supervisor
	SupervisorAvailable bool
	SetupDir            string
	LumenVersion        string
	HermesVersion       string
	HermesAdopted       bool
	IsolationReady      bool
	Outcome             Outcome
	Actions             []Action
	NextStage           Stage
}

type Probe interface {
	GOOS() string
	GOARCH() string
	TermuxPrefix() string
	Supervisor() (Supervisor, bool)
	SetupDir() string
	InstalledVersions() (lumen, hermes string, adopted bool)
	HardenedIsolation() bool
}

func Plan(ctx context.Context, req Request, probe Probe) (PlanResult, error) {
	check := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	select {
	case <-ctx.Done():
		return PlanResult{}, ctx.Err()
	default:
	}
	if probe == nil {
		return PlanResult{}, errors.New("nil setup probe")
	}
	if req.Profile != Development && req.Profile != PersonalAlpha && req.Profile != Hardened {
		return PlanResult{}, fmt.Errorf("unsupported profile %q", req.Profile)
	}
	goos := probe.GOOS()
	if err := check(); err != nil {
		return PlanResult{}, err
	}
	arch := probe.GOARCH()
	if err := check(); err != nil {
		return PlanResult{}, err
	}
	prefix := probe.TermuxPrefix()
	if err := check(); err != nil {
		return PlanResult{}, err
	}
	platform, err := detectPlatform(goos, arch, prefix)
	if err != nil {
		return PlanResult{}, err
	}
	sup, available := probe.Supervisor()
	if err := check(); err != nil {
		return PlanResult{}, err
	}
	setupDir := probe.SetupDir()
	if err := check(); err != nil {
		return PlanResult{}, err
	}
	lumenVersion, hermesVersion, adopted := probe.InstalledVersions()
	if err := check(); err != nil {
		return PlanResult{}, err
	}
	isolation := probe.HardenedIsolation()
	if err := check(); err != nil {
		return PlanResult{}, err
	}
	result := PlanResult{Profile: req.Profile, Platform: platform, Architecture: arch, Supervisor: sup, SupervisorAvailable: available, SetupDir: setupDir, LumenVersion: lumenVersion, HermesVersion: hermesVersion, HermesAdopted: adopted, IsolationReady: isolation, Outcome: Ready, NextStage: Detected}
	if req.Profile == Hardened && !result.IsolationReady {
		return PlanResult{}, ErrIsolationUnavailable
	}
	if !available {
		result.Outcome = ActionRequired
		result.Actions = []Action{{Code: "supervisor_unavailable"}}
	}
	return result, nil
}

func detectPlatform(goos, arch, prefix string) (Platform, error) {
	if arch != "arm64" && arch != "amd64" {
		return "", fmt.Errorf("unsupported architecture %q", arch)
	}
	if goos == "android" || (goos == "linux" && containsTermux(prefix)) {
		return PlatformTermux, nil
	}
	switch goos {
	case "darwin":
		return PlatformMacOS, nil
	case "linux":
		return PlatformLinux, nil
	default:
		return "", fmt.Errorf("unsupported platform %q", goos)
	}
}
func containsTermux(prefix string) bool {
	return len(prefix) > 0 && (prefix == "/data/data/com.termux/files/usr" || len(prefix) > 12 && prefix[len(prefix)-12:] == "/com.termux/files/usr")
}
