package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxSetupJourneyScriptContract(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(root, "scripts", "lumen-linux-check")
	b, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Linux journey script: %v", err)
	}
	script := string(b)
	if info, err := os.Stat(scriptPath); err != nil || info.Mode()&0111 == 0 {
		t.Fatalf("Linux journey script must be executable: %v", err)
	}
	if !strings.HasPrefix(script, "#!/bin/sh\n") || !strings.Contains(script, "set -eu") {
		t.Fatal("Linux journey script must use POSIX shell fail-fast mode")
	}
	for _, checkpoint := range []string{
		"clean-volume setup", "lumen setup", "rerun", "forced interruption",
		"doctor", "service restart", "configured synthetic task", "failed update",
		"rollback", "final canonical state comparison",
	} {
		if !strings.Contains(script, checkpoint) {
			t.Errorf("missing journey checkpoint %q", checkpoint)
		}
	}
	for _, required := range []string{
		"COMPOSE_PROJECT_NAME", "docker compose", "--volumes", "trap cleanup EXIT",
		"gateway", "8642",
		"compose.milestone1.yaml", "LUMEN_HERMES_CONTAINER_BASE_URL",
		"v2026.9.7.tar.gz",
		"907c2a72db1c5dd637ea8eeae97f4cb5b32cef615c17258f6b190924ec5bf688",
		"sha256sum -c", "LUMEN_HERMES_BUILD_IMAGE", "docker image inspect --format '{{.Id}}'", "imageRef", "docker-image",
		"--status running", "compose kill", "compose logs", "grep -v",
		"COMPOSE_FILE", "$cli service restart", "lumen-registry", "docker image push",
		".NetworkSettings.Ports", "compose ps -q lumen-registry",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("missing required journey operation %q", required)
		}
	}
	for _, forbidden := range []string{
		"LUMEN_LINUX_CHECK_TEST_MODE", "dummy-hermes", "dummy hermes", "|| true",
		"echo-only", "host reboot", "reboot", "example.invalid", "docker cp",
	} {
		if strings.Contains(strings.ToLower(script), strings.ToLower(forbidden)) {
			t.Errorf("forbidden shortcut or unsupported claim %q", forbidden)
		}
	}
	composeBytes, err := os.ReadFile(filepath.Join(root, "deploy", "docker", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	compose := string(composeBytes)
	for _, required := range []string{"API_SERVER_ENABLED", "API_SERVER_KEY", "API_SERVER_HOST", "API_SERVER_PORT", "gateway", "run", "--no-supervise"} {
		if !strings.Contains(compose, required) {
			t.Errorf("Compose is missing Hermes api_server setting %q", required)
		}
	}
	if hermesEnd := strings.Index(compose, "  lumen-host:"); hermesEnd >= 0 && strings.Contains(compose[:hermesEnd], `command: ["serve"]`) {
		t.Fatal("Hermes Compose service must use api_server gateway, not desktop serve")
	}
	overrideBytes, err := os.ReadFile(filepath.Join(root, "deploy", "docker", "compose.milestone1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	override := string(overrideBytes)
	for _, required := range []string{"lumen-provider", "lumen-test-key", "/v1/chat/completions", "stream", "LUMEN_HERMES_CONTAINER_BASE_URL"} {
		if !strings.Contains(override, required) {
			t.Errorf("milestone journey override is missing %q", required)
		}
	}
	if strings.Contains(compose, "lumen-test-key") || strings.Contains(compose, "lumen-provider") {
		t.Fatal("deterministic provider must remain test-override-only")
	}

	mise, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	miseText := string(mise)
	start := strings.Index(miseText, "[tasks.milestone1-linux-check]")
	if start < 0 {
		t.Fatal("mise.toml is missing milestone1-linux-check")
	}
	end := strings.Index(miseText[start+1:], "\n[tasks.")
	if end < 0 {
		end = len(miseText) - start - 1
	}
	task := miseText[start : start+1+end]
	for _, required := range []string{"go test", "go build", "docker compose", "lumen-linux-check", "secret", "deploy"} {
		if !strings.Contains(task, required) {
			t.Errorf("milestone1-linux-check is missing %q", required)
		}
	}
	for _, forbidden := range []string{"ANDROID_HOME", "gradle", "plutil", "launchd", "termux"} {
		if strings.Contains(strings.ToLower(task), strings.ToLower(forbidden)) {
			t.Errorf("milestone1-linux-check includes phase-2 check %q", forbidden)
		}
	}
}
