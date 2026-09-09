package contract

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSupervisorDefinitionsAreStructuredAndBounded(t *testing.T) {
	unit := readDefinition(t, "../../deploy/systemd/lumen-host.service")
	for _, want := range []string{"ExecStart=/usr/local/bin/lumen-host serve", "User=lumen", "Group=lumen", "Restart=on-failure", "StartLimitBurst=3", "StartLimitIntervalSec=60s", "KillSignal=SIGTERM", "StateDirectory=lumen", "UMask=0077"} {
		if !strings.Contains(unit, want) {
			t.Errorf("systemd unit missing %q", want)
		}
	}
	if strings.Contains(unit, "--token") || strings.Contains(unit, "--key") || strings.Contains(unit, "--secret") {
		t.Error("systemd must not pass secrets as flags")
	}
	for _, want := range []string{"EnvironmentFile=-/etc/lumen/host.env", "ProtectSystem=strict", "ProtectHome=true", "ReadWritePaths=/var/lib/lumen", "UMask=0077"} {
		if !strings.Contains(unit, want) {
			t.Errorf("systemd boundary missing %q", want)
		}
	}
	plist := readDefinition(t, "../../deploy/launchd/dev.lumen.host.plist")
	var doc struct {
		XMLName xml.Name `xml:"plist"`
		Dict    struct {
			Keys []struct {
				Key     string    `xml:"key"`
				String  string    `xml:"string"`
				Integer string    `xml:"integer"`
				False   *struct{} `xml:"false"`
			} `xml:"key"`
		} `xml:"dict"`
	}
	if err := xml.Unmarshal([]byte(plist), &doc); err != nil {
		t.Fatalf("parse launchd plist: %v", err)
	}
	for _, want := range []string{"dev.lumen.host-launcher", "Library/Application Support/Lumen", "ThrottleInterval", "ExitTimeOut"} {
		if !strings.Contains(plist, want) {
			t.Errorf("launchd plist missing %q", want)
		}
	}
	if strings.Contains(plist, "/Users/Shared") || strings.Contains(plist, "--token") {
		t.Error("launchd must use a private path and no secret flags")
	}
	for _, want := range []string{"LUMEN_OPERATOR_CREDENTIAL_FILE", "LUMEN_HERMES_BEARER_FILE", "owner-readable (0600)", "dev.lumen.host-launcher.sh"} {
		if !strings.Contains(plist, want) {
			t.Errorf("launchd boundary missing %q", want)
		}
	}
	for _, path := range []string{"../../deploy/launchd/dev.lumen.host-launcher.sh", "../../deploy/termux/finish", "../../deploy/termux/run"} {
		if err := exec.Command("sh", "-n", path).Run(); err != nil {
			t.Errorf("shell syntax %s: %v", path, err)
		}
	}
	if st, err := os.Stat("../../deploy/termux/finish"); err != nil || st.Mode()&0111 == 0 {
		t.Error("Termux finish must be executable")
	}
	termux := readDefinition(t, "../../deploy/termux/run")
	if strings.Contains(termux, "LUMEN_HOST_BIN") || !strings.Contains(termux, "$HOME/bin/lumen-host") {
		t.Error("Termux must use the fixed installed Host path")
	}
	if !strings.Contains(termux, "umask 077") || !strings.Contains(termux, "$HOME/.config/lumen/host.env") {
		t.Error("Termux must enforce private host.env configuration")
	}
	finish := readDefinition(t, "../../deploy/termux/finish")
	if !strings.Contains(finish, "basename \"$PWD\"") || !strings.Contains(finish, "sv down \"$service_name\"") {
		t.Error("Termux finish must stop the exact current runit service")
	}
	serviceDir := t.TempDir()
	finishPath, err := filepath.Abs("../../deploy/termux/finish")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	fakeBin := t.TempDir()
	logPath := filepath.Join(home, "sv.log")
	if err := os.WriteFile(filepath.Join(fakeBin, "sv"), []byte("#!/bin/sh\nprintf '%s %s\\n' \"$1\" \"$2\" >> \"$HOME/sv.log\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		finishCmd := exec.Command("sh", finishPath, "1")
		finishCmd.Dir = serviceDir
		finishCmd.Env = append(cleanEnv("HOME", "PATH"), "HOME="+home, "PATH="+fakeBin+":"+os.Getenv("PATH"))
		if out, err := finishCmd.CombinedOutput(); err != nil {
			t.Fatalf("termux finish: %v: %s", err, out)
		}
	}
	if out, err := os.ReadFile(logPath); err != nil || !strings.Contains(string(out), "down "+filepath.Base(serviceDir)) {
		t.Fatalf("termux finish did not stop exact service: %s %v", out, err)
	}
	compose := readDefinition(t, "../../deploy/docker/compose.yaml")
	for _, want := range []string{"restart: on-failure:3", "lumen-host", "read_only: true", "SIGTERM", "LUMEN_HERMES_PROFILE", "LUMEN_HERMES_BASE_URL", "LUMEN_HERMES_CA_FILE", "LUMEN_HERMES_CLIENT_CERT_FILE", "LUMEN_HERMES_CLIENT_KEY_FILE", "LUMEN_HERMES_SERVER_CERT_PIN", "hermes_ca.pem:ro", "hermes_client.crt:ro", "hermes_client.key:ro"} {
		if !strings.Contains(compose, want) {
			t.Errorf("Docker compose missing %q", want)
		}
	}
	if strings.Contains(compose, "--token") || strings.Contains(compose, "--key") {
		t.Error("Docker Compose must not pass secrets as flags")
	}
	if strings.Contains(compose, "/var/lib/lumen/hermes") {
		t.Error("Docker must not store Hermes secrets in the data volume")
	}
}

func TestSupervisorLaunchesForegroundLifecycle(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("Unix lifecycle")
	}
	bin := filepath.Join(t.TempDir(), "lumen-host")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/lumen-host").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// Keep the socket name short so Unix path limits do not affect Linux/VPS runs.
	sock := filepath.Join(dir, "s")
	cred := filepath.Join(dir, "c")
	env := cleanEnv("LUMEN_DATA_DIR", "LUMEN_SOCKET_PATH", "LUMEN_OPERATOR_CREDENTIAL_FILE", "LUMEN_HERMES_PROFILE", "LUMEN_HERMES_BASE_URL", "LUMEN_HERMES_BEARER_FILE")
	env = append(env, "LUMEN_DATA_DIR="+dir, "LUMEN_SOCKET_PATH="+sock, "LUMEN_OPERATOR_CREDENTIAL_FILE="+cred, "LUMEN_HERMES_PROFILE=development", "LUMEN_HERMES_BASE_URL=http://127.0.0.1:1", "LUMEN_HERMES_BEARER_FILE="+filepath.Join(dir, "hermes.token"))
	if err := os.WriteFile(filepath.Join(dir, "hermes.token"), []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	init := exec.Command(bin, "init")
	init.Env = env
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("init: %v: %s", err, out)
	}
	serve := exec.Command(bin, "serve")
	serve.Env = env
	stdout, err := serve.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	serve.Stderr = io.Discard
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan error, 1)
	go func() {
		line, readErr := bufio.NewReader(stdout).ReadString('\n')
		if readErr != nil || strings.TrimSpace(line) != "ready" {
			ready <- fmt.Errorf("readiness: %q: %w", line, readErr)
			return
		}
		ready <- nil
	}()
	select {
	case err := <-ready:
		if err != nil {
			_ = serve.Process.Kill()
			if goruntime.GOOS == "darwin" && strings.Contains(err.Error(), "EOF") {
				t.Skipf("sandbox denied Unix socket under testing.TempDir: %v", err)
			}
			t.Fatalf("serve did not become ready: %v", err)
		}
	case <-time.After(3 * time.Second):
		_ = serve.Process.Kill()
		t.Fatalf("serve did not become ready")
	}
	status := exec.Command(bin, "status")
	status.Env = env
	if out, err := status.CombinedOutput(); err != nil || !bytes.Contains(out, []byte("ready")) {
		t.Fatalf("status: %v %s", err, out)
	}
	if err := serve.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := serve.Wait(); err != nil {
		t.Fatalf("graceful stop: %v", err)
	}
}

func readDefinition(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func cleanEnv(keys ...string) []string {
	var env []string
	for _, entry := range os.Environ() {
		keep := true
		for _, key := range keys {
			if strings.HasPrefix(entry, key+"=") {
				keep = false
				break
			}
		}
		if keep {
			env = append(env, entry)
		}
	}
	return env
}
