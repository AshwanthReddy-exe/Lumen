package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHermesSourcePinIsCompleteAndDockerEnforcesIt(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Tag           string `json:"tag"`
		Commit        string `json:"commit"`
		ArchiveSHA256 string `json:"archiveSha256"`
	}
	b, err := os.ReadFile(filepath.Join(root, "deploy", "hermes-source.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.Tag != "v2026.9.7" || lock.Commit != "2237be355906fbe6065ce1815711eee52b2d646e" {
		t.Fatalf("unexpected source pin: %#v", lock)
	}
	if lock.ArchiveSHA256 != "907c2a72db1c5dd637ea8eeae97f4cb5b32cef615c17258f6b190924ec5bf688" {
		t.Fatalf("missing or unexpected archive digest: %q", lock.ArchiveSHA256)
	}
	dockerfile, err := os.ReadFile(filepath.Join(root, "deploy", "docker", "Dockerfile.hermes"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(dockerfile)
	for _, want := range []string{
		"HERMES_EXPECTED_COMMIT=2237be355906fbe6065ce1815711eee52b2d646e",
		"HERMES_EXPECTED_ARCHIVE_SHA256=907c2a72db1c5dd637ea8eeae97f4cb5b32cef615c17258f6b190924ec5bf688",
		"test \"$HERMES_COMMIT\" = \"$HERMES_EXPECTED_COMMIT\"",
		"test \"$HERMES_SOURCE_ARCHIVE_SHA256\" = \"$HERMES_EXPECTED_ARCHIVE_SHA256\"",
		"hermes-agent-2026.9.7/",
		"RUN uv sync --frozen",
		"COPY --from=build /src /src",
		"ENTRYPOINT [\"/src/.venv/bin/hermes\"]",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("Dockerfile missing pin enforcement %q", want)
		}
	}
}
