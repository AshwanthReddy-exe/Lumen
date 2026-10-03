package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/hermes"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

func TestWriteJSONUsesStableObjectEncoding(t *testing.T) {
	var out bytes.Buffer
	if err := writeJSON(&out, map[string]any{"z": "last", "a": "first"}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `{"a":"first","z":"last"}`+"\n"; got != want {
		t.Fatalf("JSON output = %q, want %q", got, want)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}
}

func TestLaunchActivationGateFailsClosedWithoutValidatedJournal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LUMEN_REQUIRE_VALIDATED_SETUP", "1")
	cfg := host.Config{
		DataDir: dir, SocketPath: filepath.Join(dir, "host.sock"),
		CredentialPath: filepath.Join(dir, "operator.credential"),
	}
	if launchActivationAllowedFor(cfg, "1", false, false, "darwin") {
		t.Fatal("launch activation allowed without a validated setup journal")
	}
	if launchActivationAllowedFor(cfg, "invalid", false, false, "darwin") {
		t.Fatal("unknown activation-gate value was accepted")
	}
	if launchActivationAllowedFor(cfg, "", false, false, "darwin") {
		t.Fatal("ungated LaunchAgent unexpectedly allowed on macOS")
	}
	if !launchActivationAllowedFor(cfg, "", true, false, "darwin") {
		t.Fatal("explicit manual serve was denied on macOS")
	}
	if !launchActivationAllowedFor(cfg, "", false, false, "linux") {
		t.Fatal("manual Linux serve unexpectedly required setup journal")
	}
}

func TestLaunchActivationGateIsNotLaunchdSpecificOnOtherPlatforms(t *testing.T) {
	dir := t.TempDir()
	cfg := host.Config{
		DataDir: dir, SocketPath: filepath.Join(dir, "host.sock"),
		CredentialPath: filepath.Join(dir, "operator.credential"),
	}
	t.Setenv("LUMEN_REQUIRE_VALIDATED_SETUP", "")
	if !launchActivationAllowedFor(cfg, "", false, false, "linux") {
		t.Fatal("manual serve without setup metadata unexpectedly required a journal")
	}
	if err := os.Mkdir(filepath.Join(dir, "setup"), 0700); err != nil {
		t.Fatal(err)
	}
	if !launchActivationAllowedFor(cfg, "", false, false, "linux") {
		t.Fatal("Linux serve was subjected to Launchd validation because setup metadata existed")
	}
}

func TestLaunchActivationGateAcceptsMatchingBoundJournal(t *testing.T) {
	dir := t.TempDir()
	cfg := host.Config{
		DataDir: dir, SocketPath: filepath.Join(dir, "host.sock"),
		CredentialPath: filepath.Join(dir, "operator.credential"),
		HermesBaseURL:  "http://127.0.0.1:8642", HermesProfile: hermes.ProfileDevelopment,
	}
	endpointDigest, err := setup.EndpointOriginDigest(cfg.HermesBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	artifactDigest := sha256.Sum256(contents)
	planDigest := "sha256:" + strings.Repeat("b", 64)
	binding := setup.JournalBinding{
		Profile: setup.Development, Topology: setup.TopologyExternal, Supervisor: setup.SupervisorLaunchd,
		PlanDigest: planDigest, EndpointOriginDigest: endpointDigest,
		ReferenceDigests: setup.DigestReferences(map[string]string{}),
		ArtifactPaths:    map[string]string{"lumen": executable},
		ArtifactDigests:  map[string]string{"lumen": "sha256:" + hex.EncodeToString(artifactDigest[:])},
	}
	journal, err := setup.NewJournal(filepath.Join(dir, "setup"))
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Bind(binding); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []setup.Stage{setup.Detected, setup.ArtifactsReady, setup.DirectoriesReady, setup.CredentialsReady, setup.ConfigurationReady, setup.HostInitialized} {
		if err := journal.Record(setup.StageEvidence{Stage: stage, InputDigest: "sha256:" + strings.Repeat("a", 64), Profile: setup.Development, PlanDigest: planDigest}); err != nil {
			t.Fatal(err)
		}
	}
	if !launchActivationAllowedFor(cfg, "", false, true, "darwin") {
		t.Fatal("matching setup binding did not authorize the staged LaunchAgent")
	}
	for _, stage := range []setup.Stage{setup.ServicesInstalled, setup.ServicesStarted, setup.Validated} {
		if stage == setup.Validated && !launchActivationAllowedFor(cfg, "", false, true, "darwin") {
			t.Fatal("staged LaunchAgent was denied before the terminal validation record")
		}
		if err := journal.Record(setup.StageEvidence{Stage: stage, InputDigest: "sha256:" + strings.Repeat("a", 64), Profile: setup.Development, PlanDigest: planDigest}); err != nil {
			t.Fatal(err)
		}
	}
	if !launchActivationAllowedFor(cfg, "", false, true, "darwin") {
		t.Fatal("recovery staged activation was denied after validated evidence was committed")
	}
	t.Setenv("LUMEN_REQUIRE_VALIDATED_SETUP", "1")
	if !launchActivationAllowedFor(cfg, "1", false, false, "darwin") {
		t.Fatal("matching validated journal did not authorize activation")
	}
	cfg.HermesProfile = hermes.ProfileHardened
	if launchActivationAllowedFor(cfg, "1", false, false, "darwin") {
		t.Fatal("activation accepted a profile that differs from the bound journal")
	}
}

func TestLaunchActivationGateUsesCommittedReleaseDigest(t *testing.T) {
	dir := t.TempDir()
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := host.Config{
		DataDir: dir, SocketPath: filepath.Join(dir, "host.sock"),
		CredentialPath: filepath.Join(dir, "operator.credential"),
		HermesBaseURL:  "http://127.0.0.1:8642", HermesProfile: hermes.ProfileDevelopment,
	}
	endpointDigest, err := setup.EndpointOriginDigest(cfg.HermesBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	currentDigest := sha256.Sum256(contents)
	planDigest := "sha256:" + strings.Repeat("b", 64)
	binding := setup.JournalBinding{
		Profile: setup.Development, Topology: setup.TopologyExternal, Supervisor: setup.SupervisorLaunchd,
		PlanDigest: planDigest, EndpointOriginDigest: endpointDigest,
		ReferenceDigests: setup.DigestReferences(map[string]string{}),
		ArtifactPaths:    map[string]string{"lumen": executable},
		ArtifactDigests:  map[string]string{"lumen": "sha256:" + strings.Repeat("a", 64)},
	}
	journal, err := setup.NewJournal(filepath.Join(dir, "setup"))
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Bind(binding); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []setup.Stage{setup.Detected, setup.ArtifactsReady, setup.DirectoriesReady, setup.CredentialsReady, setup.ConfigurationReady, setup.HostInitialized, setup.ServicesInstalled, setup.ServicesStarted, setup.Validated} {
		if err := journal.Record(setup.StageEvidence{Stage: stage, InputDigest: planDigest, Profile: setup.Development, PlanDigest: planDigest}); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("LUMEN_REQUIRE_VALIDATED_SETUP", "1")
	if launchActivationAllowedFor(cfg, "1", false, false, "darwin") {
		t.Fatal("accepted the original setup digest after the executable was updated")
	}
	record := setup.ReleaseRecord{
		Generation: 2, Profile: setup.Development, Topology: setup.TopologyExternal,
		Current: map[string]setup.ReleaseArtifact{"lumen": {
			Kind: setup.ArtifactExecutable, Digest: "sha256:" + hex.EncodeToString(currentDigest[:]), Path: executable,
		}},
	}
	if err := setup.SaveReleaseRecord(filepath.Join(dir, "setup", "release-record.json"), record); err != nil {
		t.Fatal(err)
	}
	if !launchActivationAllowedFor(cfg, "1", false, false, "darwin") {
		t.Fatal("denied the executable digest from the committed release generation")
	}
}

func TestCLIProcessLifecycleAndBoundary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix control socket")
	}
	bin := buildHostBinary(t)
	dataDir := t.TempDir()
	runtimeDir, err := os.MkdirTemp("", "lh-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(runtimeDir)
	if err := os.Chmod(runtimeDir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := []string{
		"LUMEN_DATA_DIR=" + dataDir,
		"LUMEN_SOCKET_PATH=" + filepath.Join(runtimeDir, "s"),
		"LUMEN_OPERATOR_CREDENTIAL_FILE=" + filepath.Join(runtimeDir, "c"),
		"LUMEN_HERMES_BASE_URL=http://127.0.0.1:1",
		"LUMEN_HERMES_PROFILE=development",
		"LUMEN_HERMES_BEARER_FILE=" + filepath.Join(runtimeDir, "hermes.token"),
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "hermes.token"), []byte("test-token"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, code := runHost(t, bin, cfg, "bogus"); code != 2 || !strings.Contains(out, "usage:") {
		t.Fatalf("usage: code=%d output=%q", code, out)
	}
	if out, code := runHost(t, bin, cfg, "init"); code != 0 || out != "" {
		t.Fatalf("init: code=%d output=%q", code, out)
	}
	credentialPath := filepath.Join(runtimeDir, "c")
	if err := os.Chmod(credentialPath, 0644); err != nil {
		t.Fatal(err)
	}
	if _, code := runHost(t, bin, cfg, "status"); code != 3 {
		t.Fatalf("credential mode failure: code=%d", code)
	}
	if err := os.Chmod(credentialPath, 0600); err != nil {
		t.Fatal(err)
	}
	credential, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	serve := exec.Command(bin, "serve", "--manual")
	serve.Env = append(os.Environ(), cfg...)
	stdout, err := serve.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	serve.Stderr = &stderr
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan error, 1)
	go func() {
		line, readErr := bufio.NewReader(stdout).ReadString('\n')
		if readErr == nil && strings.TrimSpace(line) == "ready" {
			ready <- nil
			return
		}
		ready <- fmt.Errorf("readiness: %q: %w", line, readErr)
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatalf("%v; stderr=%s", err, stderr.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not become ready")
	}

	if out, code := runHost(t, bin, cfg, "status"); code != 0 || !strings.Contains(out, "ready") {
		t.Fatalf("status: code=%d output=%q", code, out)
	} else if bytes.Contains([]byte(out), credential) {
		t.Fatal("credential appeared in status output")
	} else {
		var status map[string]any
		if err := json.Unmarshal([]byte(out), &status); err != nil || status["status"] != "ready" {
			t.Fatalf("status was not stable JSON: %q (err=%v)", out, err)
		}
	}
	conn, err := net.Dial("unix", filepath.Join(runtimeDir, "s"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = conn.Write([]byte(`{"command":"status"}` + "\n{}"))
	trailing, err := io.ReadAll(conn)
	_ = conn.Close()
	if err != nil || !bytes.Contains(trailing, []byte("trailing control data")) {
		t.Fatalf("trailing request: %q err=%v", trailing, err)
	}
	wrong := filepath.Join(runtimeDir, "wrong")
	if err := control.WriteCredential(wrong, bytes.Repeat([]byte{9}, control.CredentialSize)); err != nil {
		t.Fatal(err)
	}
	unauthorized, err := control.Call(filepath.Join(runtimeDir, "s"), wrong, control.Request{Command: "status"})
	if err != nil || unauthorized.Error != "unauthorized" {
		t.Fatalf("unauthorized request: response=%+v err=%v", unauthorized, err)
	}
	c, err := net.Dial("unix", filepath.Join(runtimeDir, "s"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = c.Write(bytes.Repeat([]byte{'x'}, control.MaxFrameSize+1))
	_ = c.Close()
	if st, err := os.Stat(filepath.Join(runtimeDir, "s")); err != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("socket permissions: stat=%v err=%v", st, err)
	}
	out, code := runHost(t, bin, cfg, "shutdown")
	if code != 0 || !strings.Contains(out, "shutting_down") {
		t.Fatalf("shutdown: code=%d output=%q", code, out)
	}
	var shutdown map[string]any
	if err := json.Unmarshal([]byte(out), &shutdown); err != nil || shutdown["status"] != "shutting_down" {
		t.Fatalf("shutdown was not stable JSON: %q (err=%v)", out, err)
	}
	if bytes.Contains([]byte(out), credential) {
		t.Fatal("credential appeared in shutdown output")
	}
	if err := serve.Wait(); err != nil {
		t.Fatalf("serve exit: %v; stderr=%s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "s")); !os.IsNotExist(err) {
		t.Fatalf("socket remains after shutdown: %v", err)
	}
	if strings.Contains(strings.Join(serve.Args, " "), string(credential)) {
		t.Fatal("credential appeared in process arguments")
	}
}

func TestCLIUsagePrecedesConfigurationFailure(t *testing.T) {
	t.Setenv("LUMEN_DATA_DIR", "")
	if code := run([]string{"serve", "extra"}); code != 2 {
		t.Fatalf("code=%d, want usage exit 2", code)
	}
}

func TestCLIServeReportsActionableStartupCause(t *testing.T) {
	bin := buildHostBinary(t)
	dataDir := t.TempDir()
	out, code := runHost(t, bin, []string{
		"LUMEN_DATA_DIR=" + dataDir,
		"LUMEN_HERMES_BASE_URL=http://127.0.0.1:8642",
		"LUMEN_HERMES_PROFILE=development",
		"LUMEN_HERMES_BEARER_FILE=" + filepath.Join(dataDir, "missing.token"),
	}, "serve")
	if runtime.GOOS == "darwin" {
		if code != 78 || !strings.Contains(out, "service activation denied: validated setup required") {
			t.Fatalf("serve activation gate: code=%d output=%q", code, out)
		}
		return
	}
	if code != 3 || !strings.Contains(out, "startup unavailable: Hermes configuration unavailable") {
		t.Fatalf("serve: code=%d output=%q", code, out)
	}
}

func TestCLIControlFailureReportsUnderlyingCause(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix control socket")
	}
	bin := buildHostBinary(t)
	dataDir := t.TempDir()
	socket := filepath.Join(dataDir, "absent.sock")
	credential := filepath.Join(dataDir, "operator.credential")
	if err := control.WriteCredential(credential, bytes.Repeat([]byte{7}, control.CredentialSize)); err != nil {
		t.Fatal(err)
	}
	out, code := runHost(t, bin, []string{
		"LUMEN_DATA_DIR=" + dataDir,
		"LUMEN_SOCKET_PATH=" + socket,
		"LUMEN_OPERATOR_CREDENTIAL_FILE=" + credential,
		"LUMEN_HERMES_BASE_URL=http://127.0.0.1:8642",
		"LUMEN_HERMES_PROFILE=development",
		"LUMEN_HERMES_BEARER_FILE=" + filepath.Join(dataDir, "hermes.token"),
	}, "status")
	if code != 3 {
		t.Fatalf("status: code=%d output=%q", code, out)
	}
	if !strings.Contains(out, "host unavailable:") {
		t.Fatalf("control failure lost its label: %q", out)
	}
	if !strings.Contains(out, socket) {
		t.Fatalf("control failure hid its cause from the operator: %q", out)
	}
}

func TestCLIDoctorReportsDeploymentProfileWithoutRunningHost(t *testing.T) {
	bin := buildHostBinary(t)
	dataDir := t.TempDir()
	cfg := []string{
		"LUMEN_DATA_DIR=" + dataDir,
		"LUMEN_HERMES_BASE_URL=http://127.0.0.1:8642",
		"LUMEN_HERMES_PROFILE=development",
		"LUMEN_HERMES_BEARER_FILE=" + filepath.Join(dataDir, "hermes.token"),
	}
	out, code := runHost(t, bin, cfg, "doctor")
	if code != 0 {
		t.Fatalf("doctor: code=%d output=%q", code, out)
	}
	var doctor map[string]any
	if err := json.Unmarshal([]byte(out), &doctor); err != nil {
		t.Fatalf("doctor output is not JSON: %q: %v", out, err)
	}
	for key, want := range map[string]any{
		"status":             "ok",
		"deployment_profile": "development",
		"host_initialized":   false,
		"hermes_profile":     "development",
		"hermes_configured":  true,
	} {
		if doctor[key] != want {
			t.Fatalf("doctor[%s]=%#v, want %#v in %#v", key, doctor[key], want, doctor)
		}
	}
	if strings.Contains(out, "hermes.token") {
		t.Fatalf("doctor leaked secret path: %q", out)
	}
}

func buildHostBinary(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "lumen-host")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/lumen-host")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build host: %v\n%s", err, output)
	}
	return out
}

func runHost(t *testing.T, bin string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(out), exitErr.ExitCode()
	}
	t.Fatal(err)
	return "", -1
}
