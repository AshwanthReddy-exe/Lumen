package host

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/conversation"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

var dockerIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var imageIDPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type dockerChatCertifier struct {
	config  Config
	runtime hermes.Adapter
}

func newDockerChatCertifier(config Config, runtime hermes.Adapter) conversation.Certifier {
	return &dockerChatCertifier{config: config, runtime: runtime}
}

type chatProbeEvidence struct {
	Status            string `json:"status"`
	ImageID           string `json:"imageId"`
	ConfigDigest      string `json:"configDigest"`
	ProviderCalls     int    `json:"providerCalls"`
	RejectedToolCalls int    `json:"rejectedToolCalls"`
}

type dockerChatInspection struct {
	ID    string `json:"Id"`
	Image string `json:"Image"`
	State struct {
		Running   bool   `json:"Running"`
		PID       int    `json:"Pid"`
		StartedAt string `json:"StartedAt"`
	} `json:"State"`
	Config struct {
		User       string   `json:"User"`
		Entrypoint []string `json:"Entrypoint"`
		Cmd        []string `json:"Cmd"`
		Env        []string `json:"Env"`
	} `json:"Config"`
	HostConfig struct {
		ReadonlyRootfs bool `json:"ReadonlyRootfs"`
		Privileged     bool `json:"Privileged"`
		PortBindings   map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"PortBindings"`
		Tmpfs map[string]string `json:"Tmpfs"`
	} `json:"HostConfig"`
	Mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
}

func (c *dockerChatCertifier) Certify(ctx context.Context, _ space.State, profile space.RuntimeProfile) (space.RuntimeCertification, error) {
	deny := func() (space.RuntimeCertification, error) {
		return space.RuntimeCertification{}, conversation.ErrRuntimeProfileUnverified
	}
	if !dockerIDPattern.MatchString(c.config.ChatContainerID) || c.config.HermesProfile != hermes.ProfileDevelopment {
		return deny()
	}
	u, err := url.Parse(c.config.ChatBaseURL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return deny()
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return deny()
	}
	probeBytes, err := readRestrictedSecret(c.config.ChatProbePath)
	if err != nil || len(probeBytes) > 4096 {
		return deny()
	}
	var probe chatProbeEvidence
	decoder := json.NewDecoder(bytes.NewReader(probeBytes))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&probe) != nil || decoder.Decode(new(any)) != io.EOF || probe.Status != "container_probe_passed" || !imageIDPattern.MatchString(probe.ImageID) || probe.ProviderCalls != 4 || probe.RejectedToolCalls != 3 {
		return deny()
	}
	configBytes, err := os.ReadFile(c.config.ChatConfigPath)
	if err != nil || len(configBytes) > 4096 || probe.ConfigDigest != space.DigestText(string(configBytes)) {
		return deny()
	}
	key, err := readRestrictedSecret(c.config.ChatBearerPath)
	if err != nil || len(bytes.TrimSpace(key)) == 0 {
		return deny()
	}
	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(inspectCtx, "docker", "inspect", "--type", "container", c.config.ChatContainerID).Output()
	if err != nil || len(output) > 128<<10 {
		return deny()
	}
	var containers []dockerChatInspection
	if json.Unmarshal(output, &containers) != nil || len(containers) != 1 {
		return deny()
	}
	d := containers[0]
	if d.ID != c.config.ChatContainerID || d.Image != probe.ImageID || !d.State.Running || d.State.PID <= 0 || d.State.StartedAt == "" || d.Config.User != "65532:65532" || !d.HostConfig.ReadonlyRootfs || d.HostConfig.Privileged {
		return deny()
	}
	if _, err := time.Parse(time.RFC3339Nano, d.State.StartedAt); err != nil {
		return deny()
	}
	if len(d.Config.Entrypoint) != 1 || d.Config.Entrypoint[0] != "/src/.venv/bin/hermes" || len(d.Config.Cmd) != 4 || d.Config.Cmd[0] != "gateway" || d.Config.Cmd[1] != "run" || d.Config.Cmd[2] != "--no-supervise" || d.Config.Cmd[3] != "--force" {
		return deny()
	}
	if len(d.Mounts) != 1 || d.Mounts[0].Source != c.config.ChatConfigPath || d.Mounts[0].Destination != "/var/lib/hermes/config.yaml" || d.Mounts[0].RW {
		return deny()
	}
	if len(d.HostConfig.Tmpfs) != 1 || d.HostConfig.Tmpfs["/var/lib/hermes"] != "rw,uid=65532,gid=65532,mode=0700" || len(d.HostConfig.PortBindings) != 1 || len(d.HostConfig.PortBindings["8642/tcp"]) != 1 {
		return deny()
	}
	binding := d.HostConfig.PortBindings["8642/tcp"][0]
	if binding.HostIP != "127.0.0.1" || binding.HostPort != u.Port() {
		return deny()
	}
	env := map[string]string{}
	for _, entry := range d.Config.Env {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || name == "" {
			return deny()
		}
		if _, duplicate := env[name]; duplicate {
			return deny()
		}
		env[name] = value
	}
	for name, want := range map[string]string{"HERMES_SAFE_MODE": "1", "LUMEN_CHAT_ZERO_TOOL": "1", "HERMES_HOME": "/var/lib/hermes", "API_SERVER_ENABLED": "true", "API_SERVER_HOST": "0.0.0.0", "API_SERVER_PORT": "8642", "API_SERVER_KEY": string(bytes.TrimSpace(key))} {
		if env[name] != want {
			return deny()
		}
	}
	// The qualified image fixes its inherited environment. Reject any container
	// override outside the launch contract, especially provider routing variables.
	imageOutput, err := exec.CommandContext(inspectCtx, "docker", "image", "inspect", probe.ImageID).Output()
	if err != nil || len(imageOutput) > 128<<10 {
		return deny()
	}
	var images []struct {
		Config struct {
			Env []string `json:"Env"`
		} `json:"Config"`
	}
	if json.Unmarshal(imageOutput, &images) != nil || len(images) != 1 {
		return deny()
	}
	allowed := map[string]string{}
	baseNames := map[string]bool{"PATH": true, "LANG": true, "GPG_KEY": true, "PYTHON_VERSION": true, "PYTHON_SHA256": true}
	for _, entry := range images[0].Config.Env {
		name, value, ok := strings.Cut(entry, "=")
		_, duplicate := allowed[name]
		if !ok || !baseNames[name] || duplicate {
			return deny()
		}
		allowed[name] = value
	}
	if len(allowed) != len(baseNames) {
		return deny()
	}
	for name, value := range map[string]string{"HERMES_SAFE_MODE": "1", "LUMEN_CHAT_ZERO_TOOL": "1", "HERMES_HOME": "/var/lib/hermes", "TERMINAL_CWD": "/var/lib/hermes", "API_SERVER_ENABLED": "true", "API_SERVER_HOST": "0.0.0.0", "API_SERVER_PORT": "8642", "API_SERVER_KEY": string(bytes.TrimSpace(key)), "API_SERVER_MODEL_NAME": "synthetic-model", "HERMES_INFERENCE_PROVIDER": "custom", "HERMES_INFERENCE_MODEL": "synthetic-model", "OPENAI_API_KEY": "synthetic-only", "OPENROUTER_API_KEY": "synthetic-only"} {
		allowed[name] = value
	}
	providerURL := env["OPENAI_BASE_URL"]
	provider, err := url.Parse(providerURL)
	if err != nil || providerURL != c.config.ChatProviderURL || provider.Scheme != "http" || provider.Hostname() != "host.docker.internal" || provider.Path != "/v1" || provider.RawQuery != "" || provider.Fragment != "" || provider.User != nil || env["OPENROUTER_BASE_URL"] != providerURL {
		return deny()
	}
	providerPort, err := strconv.Atoi(provider.Port())
	if err != nil || providerPort < 1 || providerPort > 65535 {
		return deny()
	}
	allowed["OPENAI_BASE_URL"] = providerURL
	allowed["OPENROUTER_BASE_URL"] = providerURL
	if len(env) != len(allowed) {
		return deny()
	}
	for name, want := range allowed {
		if env[name] != want {
			return deny()
		}
	}
	health, err := c.runtime.Health(ctx)
	if err != nil || health.Status != "ok" {
		return deny()
	}
	observer, ok := c.runtime.(interface {
		VerifiedConversationEndpointIdentity(context.Context) (string, error)
	})
	if !ok {
		return deny()
	}
	endpoint, err := observer.VerifiedConversationEndpointIdentity(ctx)
	if err != nil || endpoint == "" {
		return deny()
	}
	process := "docker:" + d.ID + ":" + strconv.Itoa(d.State.PID) + ":" + d.State.StartedAt
	limits := space.RuntimeProfileLimits{Version: 1, MaxTurns: profile.MaxTurns, MaxMessages: profile.MaxMessages, MaxContextBytes: profile.MaxContextBytes, MaxInputTokens: profile.MaxInputTokens, MaxOutputTokens: profile.MaxOutputTokens, MaxTotalTokens: profile.MaxTotalTokens, DeadlineSeconds: profile.DeadlineSeconds}
	id := space.DigestText(process + probe.ImageID + probe.ConfigDigest + endpoint + profile.Digest)
	return space.RuntimeCertification{ID: id, RuntimeIdentity: "docker:" + d.ID, EndpointIdentity: endpoint, ArtifactDigest: probe.ImageID, ProcessIdentity: process, HermesVersion: "0.21.1", PluginIdentity: "lumen-chat-zero-tool-patch", PluginCommit: "2237be355906fbe6065ce1815711eee52b2d646e", ProfileDigest: profile.Digest, ConfigDigest: probe.ConfigDigest, EffectiveToolsets: []string{}, MemoryRead: false, MemoryWrite: false, Limits: limits, Evidence: space.DigestText(string(probeBytes)), ExpiresAt: time.Now().Truncate(time.Minute).Add(time.Minute).Unix()}, nil
}
