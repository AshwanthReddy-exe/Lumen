package contract

import (
	"bytes"
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
	for _, want := range []string{"GOOS=android GOARCH=arm64", "lumen-host-android-arm64"} {
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
	for _, program := range f.Progs {
		if program.Type != elf.PT_INTERP {
			continue
		}
		interpreter := make([]byte, program.Filesz)
		if _, err := program.ReadAt(interpreter, 0); err != nil {
			t.Fatal(err)
		}
		if string(bytes.TrimRight(interpreter, "\x00")) != "/system/bin/linker64" {
			t.Fatalf("artifact uses a non-Android loader: %q", interpreter)
		}
		return
	}
	t.Fatal("artifact is missing an Android dynamic loader")
}
