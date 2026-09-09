package main

import (
	"bufio"
	"bytes"
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
)

func TestCLIProcessLifecycleAndBoundary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix control socket")
	}
	bin := buildHostBinary(t)
	dataDir := t.TempDir()
	runtimeDir, err := os.MkdirTemp("/private/tmp", "lh-")
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
	serve := exec.Command(bin, "serve")
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
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not become ready")
	}

	if out, code := runHost(t, bin, cfg, "status"); code != 0 || !strings.Contains(out, "ready") {
		t.Fatalf("status: code=%d output=%q", code, out)
	} else if bytes.Contains([]byte(out), credential) {
		t.Fatal("credential appeared in status output")
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
