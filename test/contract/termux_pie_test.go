package contract

import (
	"debug/elf"
	"os"
	"strings"
	"testing"
)

func TestPhase2BuildDefinesTermuxPIEArtifact(t *testing.T) {
	b, err := os.ReadFile("../../mise.toml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"-buildmode=pie", "lumen-host-linux-arm64-pie"} {
		if !strings.Contains(s, want) {
			t.Fatalf("phase2-check missing Termux PIE setting %q", want)
		}
	}
}

func TestTermuxPIEArtifactIsPositionIndependentWhenBuilt(t *testing.T) {
	path := os.Getenv("LUMEN_TERMUX_PIE_ARTIFACT")
	if path == "" {
		t.Skip("set LUMEN_TERMUX_PIE_ARTIFACT to inspect a built Termux artifact")
	}
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Class != elf.ELFCLASS64 || f.Machine != elf.EM_AARCH64 || f.Type != elf.ET_DYN {
		t.Fatalf("artifact is not a 64-bit ARM PIE: class=%v machine=%v type=%v", f.Class, f.Machine, f.Type)
	}
}
