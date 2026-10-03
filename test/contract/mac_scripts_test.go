package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
)

func TestMacHostForwardsStatusWithConfiguredBinary(t *testing.T) {
	if goruntime.GOOS != "darwin" {
		t.Skip("macOS script contract")
	}
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
	if goruntime.GOOS != "darwin" {
		t.Skip("macOS script contract")
	}
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
  "status ") printf '%s\n' '{"status":"ready","host_id":"host-test"}' ;;
  "task submit") printf '{"outcome":"awaiting_permission","requestId":"submit","subjectId":"task"}\n' ;;
  "approval resolve") printf '{"outcome":"queued","requestId":"approve","subjectId":"task"}\n' ;;
  "task show")
    if [ "$TEST_FAIL_TASK" = yes ]; then
      printf '{"runtimeApprovals":[],"task":{"status":"failed"}}\n'
    else
      count=0; [ ! -f "$TEST_SHOW_COUNT" ] || count=$(cat "$TEST_SHOW_COUNT")
      count=$((count + 1)); printf '%s' "$count" > "$TEST_SHOW_COUNT"
      if [ "$count" -eq 1 ]; then
        printf '{"runtimeApprovals":[],"task":{"status":"running"}}\n'
      else
        printf '{"runtimeApprovals":[],"task":{"status":"completed","output":"Hermes answered through Lumen."}}\n'
      fi
    fi ;;
  *) exit 8 ;;
esac
`
	writeExecutable(t, bin, fake)
	fakeTools := filepath.Join(dir, "tools")
	if err := os.Mkdir(fakeTools, 0700); err != nil {
		t.Fatal(err)
	}
	toolsetFile := filepath.Join(dir, "toolsets.json")
	if err := os.WriteFile(toolsetFile, []byte(`{"data":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	curl := `#!/bin/sh
case "$*" in *secret*|*credential-value*) exit 41 ;; esac
case "$*" in *"/v1/capabilities"*) exit 0 ;; *"/v1/toolsets"*) cat "$FAKE_TOOLSETS_FILE" ;; *) exit 9 ;; esac
`
	writeExecutable(t, filepath.Join(fakeTools, "curl"), curl)
	cmd := exec.Command("../../scripts/lumen-mac-test", "Give me one short quote.")
	cmd.Env = append(cleanEnv("HOME", "LUMEN_DATA_DIR", "LUMEN_HERMES_BEARER_FILE", "LUMEN_BINARY"), "HOME="+home, "PATH="+fakeTools+":"+os.Getenv("PATH"), "LUMEN_BINARY="+bin, "LUMEN_TEST_POLL_SECONDS=0", "TEST_CALLS="+calls, "TEST_SHOW_COUNT="+showCount, "TEST_DATA_DIR="+dataDir, "FAKE_TOOLSETS_FILE="+toolsetFile)
	if err := os.WriteFile(filepath.Join(dataDir, "hermes.token"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Host task receipt: mac-hermes-test-") || !strings.Contains(string(out), "host=host-test") || !strings.HasSuffix(string(out), "Hermes answered through Lumen.\n") {
		t.Fatalf("e2e: %v %q", err, out)
	}
	if accepted, completed := strings.Index(string(out), "status=awaiting_permission"), strings.Index(string(out), "status=completed"); accepted < 0 || completed <= accepted {
		t.Fatalf("submission and terminal receipts missing or out of order: %q", out)
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

	cmd = exec.Command("../../scripts/lumen-mac-test", "Give me one short quote.")
	cmd.Env = append(cleanEnv("HOME", "LUMEN_DATA_DIR", "LUMEN_HERMES_BEARER_FILE", "LUMEN_BINARY", "TEST_FAIL_TASK"), "HOME="+home, "PATH="+fakeTools+":"+os.Getenv("PATH"), "LUMEN_BINARY="+bin, "LUMEN_TEST_POLL_SECONDS=0", "TEST_CALLS="+calls, "TEST_SHOW_COUNT="+showCount, "TEST_DATA_DIR="+dataDir, "FAKE_TOOLSETS_FILE="+toolsetFile, "TEST_FAIL_TASK=yes")
	out, err = cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "Host task receipt: mac-hermes-test-") || !strings.Contains(string(out), "status=failed host=host-test") {
		t.Fatalf("failure receipt missing: %v %q", err, out)
	}
	if strings.Contains(string(out), "Give me one short quote.") || strings.Contains(string(out), "secret") {
		t.Fatalf("prompt or credential leaked: %q", out)
	}
}

func TestMacE2ERejectsHermesURLUserinfoWithoutLeakingIt(t *testing.T) {
	if goruntime.GOOS != "darwin" {
		t.Skip("macOS script contract")
	}
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

func TestMacE2ERejectsEnabledHermesToolsetsBeforeSubmission(t *testing.T) {
	if goruntime.GOOS != "darwin" {
		t.Skip("macOS script contract")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "lumen-host")
	calls := filepath.Join(dir, "calls")
	fake := `#!/bin/sh
case "$1 $2" in
  "status ") printf '%s\n' '{"status":"ready","host_id":"host-test"}' ;;
  "task submit") printf submit >> "$TEST_CALLS" ;;
  *) exit 8 ;;
esac
`
	writeExecutable(t, bin, fake)
	fakeTools := filepath.Join(dir, "tools")
	if err := os.Mkdir(fakeTools, 0700); err != nil {
		t.Fatal(err)
	}
	toolsetFile := filepath.Join(dir, "toolsets.json")
	if err := os.WriteFile(toolsetFile, []byte(`{"data":[{"enabled":true}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	curl := `#!/bin/sh
case "$*" in *secret*|*credential-value*) exit 41 ;; esac
case "$*" in *"/v1/capabilities"*) exit 0 ;; *"/v1/toolsets"*) cat "$FAKE_TOOLSETS_FILE" ;; esac
`
	writeExecutable(t, filepath.Join(fakeTools, "curl"), curl)
	token := filepath.Join(dir, "token")
	if err := os.WriteFile(token, []byte("credential-value"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("../../scripts/lumen-mac-test")
	cmd.Env = append(cleanEnv("HOME", "LUMEN_BINARY", "LUMEN_DATA_DIR", "LUMEN_HERMES_BEARER_FILE"),
		"HOME="+dir, "PATH="+fakeTools+":"+os.Getenv("PATH"), "LUMEN_BINARY="+bin,
		"LUMEN_DATA_DIR="+dir, "LUMEN_HERMES_BEARER_FILE="+token,
		"FAKE_TOOLSETS_FILE="+toolsetFile, "TEST_CALLS="+calls)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "enabled or unverifiable API toolsets") {
		t.Fatalf("unsafe toolsets accepted: %v %q", err, out)
	}
	if _, err := os.Stat(calls); !os.IsNotExist(err) {
		t.Fatalf("Host submitted before toolset denial: %v", err)
	}
	if strings.Contains(string(out), "credential-value") {
		t.Fatal("credential leaked")
	}
}

func TestMacE2ERejectsCurlConfigCharactersInHermesCredential(t *testing.T) {
	if goruntime.GOOS != "darwin" {
		t.Skip("macOS script contract")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "lumen-host")
	writeExecutable(t, bin, "#!/bin/sh\nprintf '%s\\n' '{\"status\":\"ready\",\"host_id\":\"host-test\"}'\n")
	fakeTools := filepath.Join(dir, "tools")
	if err := os.Mkdir(fakeTools, 0700); err != nil {
		t.Fatal(err)
	}
	curlCalled := filepath.Join(dir, "curl-called")
	curl := `#!/bin/sh
: > "$TEST_CURL_CALLED"
exit 0
`
	writeExecutable(t, filepath.Join(fakeTools, "curl"), curl)
	token := filepath.Join(dir, "token")
	unsafe := "bearer\"\noutput = \"/tmp/lumen-curl-config-injection\""
	if err := os.WriteFile(token, []byte(unsafe), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("../../scripts/lumen-mac-test")
	cmd.Env = append(cleanEnv("HOME", "LUMEN_BINARY", "LUMEN_DATA_DIR", "LUMEN_HERMES_BEARER_FILE"),
		"HOME="+dir, "PATH="+fakeTools+":"+os.Getenv("PATH"), "LUMEN_BINARY="+bin,
		"LUMEN_DATA_DIR="+dir, "LUMEN_HERMES_BEARER_FILE="+token,
		"TEST_CURL_CALLED="+curlCalled)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "unsupported characters") {
		t.Fatalf("unsafe credential accepted: %v %q", err, out)
	}
	if _, err := os.Stat(curlCalled); !os.IsNotExist(err) {
		t.Fatalf("curl ran with unsafe credential: %v", err)
	}
	if strings.Contains(string(out), unsafe) {
		t.Fatal("unsafe credential leaked")
	}
}
