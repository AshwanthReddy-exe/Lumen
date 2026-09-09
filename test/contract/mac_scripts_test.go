package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMacHostForwardsStatusWithConfiguredBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "lumen-host")
	writeExecutable(t, bin, "#!/bin/sh\n[ \"$1\" = status ] || exit 9\nprintf '{\"status\":\"ready\"}\\n'\n")
	cmd := exec.Command("../../scripts/lumen-mac-host", "status")
	cmd.Env = append(os.Environ(), "LUMEN_BINARY="+bin, "LUMEN_DATA_DIR="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "{\"status\":\"ready\"}\n" {
		t.Fatalf("status: %v %q", err, out)
	}
}

func TestMacE2EApprovesOncePollsAndPrintsDurableOutput(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	dataDir := filepath.Join(home, ".local", "share", "lumen")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "lumen-host")
	calls := filepath.Join(dir, "calls")
	showCount := filepath.Join(dir, "show-count")
	fake := `#!/bin/sh
test "$LUMEN_DATA_DIR" = "$TEST_DATA_DIR" || exit 10
printf '%s\n' "$*" >> "$TEST_CALLS"
case "$1 $2" in
  "status ") printf '{"status":"ready","host_id":"host-test"}\n' ;;
  "task submit") printf '{"outcome":"awaiting_permission","requestId":"submit","subjectId":"task"}\n' ;;
  "approval resolve") printf '{"outcome":"queued","requestId":"approve","subjectId":"task"}\n' ;;
  "task show")
    count=0; [ ! -f "$TEST_SHOW_COUNT" ] || count=$(cat "$TEST_SHOW_COUNT")
    count=$((count + 1)); printf '%s' "$count" > "$TEST_SHOW_COUNT"
    if [ "$count" -eq 1 ]; then
      printf '{"runtimeApprovals":[],"task":{"status":"running"}}\n'
    else
      printf '{"runtimeApprovals":[],"task":{"status":"completed","output":"Hermes answered through Lumen."}}\n'
    fi ;;
  *) exit 8 ;;
esac
`
	writeExecutable(t, bin, fake)
	fakeTools := filepath.Join(dir, "tools")
	if err := os.Mkdir(fakeTools, 0700); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(fakeTools, "curl"), "#!/bin/sh\nexit 0\n")
	cmd := exec.Command("../../scripts/lumen-mac-test", "Give me one short quote.")
	cmd.Env = append(cleanEnv("HOME", "LUMEN_DATA_DIR", "LUMEN_HERMES_BEARER_FILE", "LUMEN_BINARY"), "HOME="+home, "PATH="+fakeTools+":"+os.Getenv("PATH"), "LUMEN_BINARY="+bin, "LUMEN_TEST_POLL_SECONDS=0", "TEST_CALLS="+calls, "TEST_SHOW_COUNT="+showCount, "TEST_DATA_DIR="+dataDir)
	if err := os.WriteFile(filepath.Join(dataDir, "hermes.token"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "Hermes answered through Lumen.\n" {
		t.Fatalf("e2e: %v %q", err, out)
	}
	log, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	text := string(log)
	for _, want := range []string{"task submit", "approval resolve", "--decision once", "task show"} {
		if !strings.Contains(text, want) {
			t.Fatalf("calls missing %q: %s", want, text)
		}
	}
	if strings.Contains(string(out), "secret") || strings.Contains(text, "secret") {
		t.Fatal("credential leaked through output or arguments")
	}
}

func TestMacE2ERejectsHermesURLUserinfoWithoutLeakingIt(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "lumen-host")
	writeExecutable(t, bin, "#!/bin/sh\nprintf '{\"status\":\"ready\",\"host_id\":\"host-test\"}\\n'\n")
	token := filepath.Join(dir, "hermes.token")
	if err := os.WriteFile(token, []byte("bearer-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("../../scripts/lumen-mac-test")
	cmd.Env = append(cleanEnv("LUMEN_BINARY", "LUMEN_DATA_DIR", "LUMEN_HERMES_BEARER_FILE", "LUMEN_HERMES_BASE_URL"), "LUMEN_BINARY="+bin, "LUMEN_DATA_DIR="+dir, "LUMEN_HERMES_BEARER_FILE="+token, "LUMEN_HERMES_BASE_URL=http://user:url-secret@127.0.0.1:8642")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("URL userinfo was accepted")
	}
	if strings.Contains(string(out), "url-secret") || strings.Contains(string(out), "bearer-secret") {
		t.Fatalf("credential leaked: %q", out)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
}
