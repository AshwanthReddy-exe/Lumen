package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "doctor", "init", "status", "shutdown":
		if len(args) != 1 {
			return usage()
		}
	case "serve":
		if len(args) != 1 && (len(args) != 2 || (args[1] != "--manual" && args[1] != "--setup-staging")) {
			return usage()
		}
	case "task":
		if len(args) < 2 || (args[1] != "submit" && args[1] != "show" && args[1] != "cancel") {
			return usage()
		}
	case "approval":
		if len(args) < 2 || args[1] != "resolve" {
			return usage()
		}
	case "conversation":
		if len(args) < 2 || (args[1] != "create" && args[1] != "send" && args[1] != "show") {
			return usage()
		}
	case "preference":
		if len(args) < 2 || args[1] != "set" {
			return usage()
		}
	case "memory":
		if len(args) < 2 || (args[1] != "save" && args[1] != "list" && args[1] != "delete") {
			return usage()
		}
	case "runtime":
		if len(args) != 2 || args[1] != "inspect" {
			return usage()
		}
	default:
		return usage()
	}
	c, err := host.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration unavailable: %v\n", err)
		return 3
	}
	switch args[0] {
	case "doctor":
		return doctor(c)
	case "init":
		if err := host.Initialize(c); err != nil {
			fmt.Fprintf(os.Stderr, "initialization unavailable: %v\n", err)
			return 3
		}
		return 0
	case "serve":
		return serve(c, len(args) == 2 && args[1] == "--manual", len(args) == 2 && args[1] == "--setup-staging")
	case "status", "shutdown":
		return call(c, args[0])
	case "runtime":
		return call(c, "runtime inspect")
	default:
		arguments, ok := parseArguments(args[2:])
		if !ok {
			return usage()
		}
		if !validPublicArguments(args[0]+" "+args[1], arguments) {
			return usage()
		}
		return callWithArguments(c, args[0]+" "+args[1], arguments)
	}
}
func usage() int {
	fmt.Fprintln(os.Stderr, "usage: lumen-host doctor|init|serve [--manual|--setup-staging]|status|runtime inspect|task submit|task show|task cancel|approval resolve|conversation create|conversation send|conversation show|preference set|memory save|memory list|memory delete|shutdown")
	return 2
}

func doctor(c host.Config) int {
	profile := "hardened-capable"
	if c.HermesProfile == hermes.ProfileDevelopment {
		profile = "development"
	} else if isTermux() {
		profile = "personal-alpha"
	}
	report := map[string]any{
		"status":             "ok",
		"platform":           platformName(),
		"deployment_profile": profile,
		"host_initialized":   fileExists(filepath.Join(c.DataDir, "initialized")),
		"hermes_profile":     c.HermesProfile,
		"hermes_configured":  c.HermesBaseURL != "",
	}
	if err := writeJSON(os.Stdout, report); err != nil {
		fmt.Fprintf(os.Stderr, "output unavailable: %v\n", err)
		return 3
	}
	return 0
}

func serve(c host.Config, manual, staging bool) int {
	if !launchActivationAllowedFor(c, os.Getenv("LUMEN_REQUIRE_VALIDATED_SETUP"), manual, staging, runtime.GOOS) {
		fmt.Fprintln(os.Stderr, "service activation denied: validated setup required on this platform; use serve --manual only for an intentional foreground run")
		return 78
	}
	s, err := host.New(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup unavailable: %v\n", err)
		return 3
	}
	if err := s.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "service unavailable: %v\n", err)
		return 3
	}
	fmt.Println("ready")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); s.Shutdown() }()
	if err := s.Wait(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "runtime incompatibility: %v\n", err)
		return 4
	}
	return 0
}

func launchActivationAllowedFor(c host.Config, mode string, manual, staging bool, goos string) bool {
	if mode == "" {
		if goos != "darwin" || manual {
			return true
		}
		if !staging {
			return false
		}
	} else if mode != "1" {
		return false
	}
	if c.DataDir == "" || !filepath.IsAbs(c.DataDir) || filepath.Clean(c.DataDir) != c.DataDir || c.SocketPath != filepath.Join(c.DataDir, "host.sock") || c.CredentialPath != filepath.Join(c.DataDir, "operator.credential") {
		return false
	}
	journal, err := setup.NewJournal(filepath.Join(c.DataDir, "setup"))
	if err != nil {
		return false
	}
	next := journal.Next()
	if mode == "1" || journal.IsValidated() {
		if !journal.IsValidatedForBinding() {
			return false
		}
	} else if next != setup.ServicesInstalled && next != setup.ServicesStarted && next != setup.Validated {
		return false
	}
	binding, ok := journal.Binding()
	if !ok || binding.Topology != setup.TopologyExternal || binding.Supervisor != setup.SupervisorLaunchd {
		return false
	}
	wantProfile := hermes.ProfileHardened
	if binding.Profile == setup.Development {
		wantProfile = hermes.ProfileDevelopment
	}
	if c.HermesProfile != wantProfile {
		return false
	}
	endpointDigest, err := setup.EndpointOriginDigest(c.HermesBaseURL)
	if err != nil || endpointDigest != binding.EndpointOriginDigest {
		return false
	}
	if !setup.ReferenceDigestsMatch(binding.ReferenceDigests, setup.DigestReferences(map[string]string{
		"credential":  c.HermesBearerPath,
		"ca":          c.HermesCAPath,
		"client_cert": c.HermesClientCertPath,
		"client_key":  c.HermesClientKeyPath,
		"server_pin":  c.HermesServerPin,
	})) {
		return false
	}
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil || filepath.Clean(binding.ArtifactPaths["lumen"]) != executable {
		return false
	}
	contents, err := os.ReadFile(executable)
	if err != nil {
		return false
	}
	digests, err := setup.EffectiveReleaseDigests(binding, c.DataDir)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(contents)
	return digests["lumen"] == "sha256:"+hex.EncodeToString(digest[:])
}

func isTermux() bool {
	return strings.Contains(os.Getenv("PREFIX"), "/com.termux/")
}

func platformName() string {
	if isTermux() {
		return "android-termux"
	}
	return runtime.GOOS + "/" + runtime.GOARCH
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
func call(c host.Config, cmd string) int {
	return callWithArguments(c, cmd, nil)
}
func callWithArguments(c host.Config, cmd string, arguments map[string]string) int {
	r, err := control.Call(c.SocketPath, c.CredentialPath, control.Request{Command: cmd, Arguments: arguments})
	if err != nil {
		fmt.Fprintf(os.Stderr, "host unavailable: %v\n", err)
		return 3
	}
	if !r.OK || r.Error != "" {
		fmt.Fprintln(os.Stderr, r.Error)
		return 3
	}
	if r.Data != nil {
		if err := writeJSON(os.Stdout, r.Data); err != nil {
			fmt.Fprintf(os.Stderr, "output unavailable: %v\n", err)
			return 3
		}
	}
	return 0
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func parseArguments(args []string) (map[string]string, bool) {
	arguments := make(map[string]string)
	for i := 0; i < len(args); i++ {
		key := strings.TrimPrefix(args[i], "--")
		if key == args[i] || key == "" {
			return nil, false
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			return nil, false
		}
		if _, exists := arguments[key]; exists {
			return nil, false
		}
		arguments[key] = args[i+1]
		i++
	}
	return arguments, true
}

func validPublicArguments(command string, arguments map[string]string) bool {
	allowed := map[string]map[string]bool{
		"conversation create": {"request_id": true, "conversation_id": true, "surface_id": true},
		"conversation send":   {"request_id": true, "conversation_id": true, "surface_id": true, "input": true, "task_id": true},
		"conversation show":   {"conversation_id": true},
		"preference set":      {"request_id": true, "name": true, "value": true},
		"memory save":         {"request_id": true, "memory_id": true, "text": true},
		"memory list":         {},
		"memory delete":       {"request_id": true, "memory_id": true},
	}
	set, ok := allowed[command]
	if !ok {
		return true
	}
	for key := range arguments {
		if !set[key] {
			return false
		}
	}
	for _, required := range map[string][]string{
		"conversation create": {"request_id", "conversation_id", "surface_id"},
		"conversation send":   {"request_id", "conversation_id", "surface_id", "input"},
		"conversation show":   {"conversation_id"},
		"preference set":      {"request_id", "name", "value"},
		"memory save":         {"request_id", "memory_id", "text"},
		"memory list":         {},
		"memory delete":       {"request_id", "memory_id"},
	}[command] {
		if arguments[required] == "" {
			return false
		}
	}
	return true
}
