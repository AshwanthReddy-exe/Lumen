package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTermuxInstallerCreatesPrivateHostAndService(t *testing.T) {
	root, err := os.MkdirTemp("/private/tmp", "lumen-termux-installer-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"lumen-host-android-arm64", "run", "finish"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("fixture"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"hermes-android-arm64", "hermes-run"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("fixture"), 0700); err != nil { t.Fatal(err) }
	}
	developmentConfig := "LUMEN_HERMES_PROFILE=development\nLUMEN_HERMES_BASE_URL=http://127.0.0.1:8642\n"
	if err := os.WriteFile(filepath.Join(source, "host.env"), []byte(developmentConfig), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "hermes.token"), []byte("development-token"), 0600); err != nil {
		t.Fatal(err)
	}
	installer, err := filepath.Abs("../../deploy/termux/install-lumen-host")
	if err != nil {
		t.Fatal(err)
	}
	installerBody, err := os.ReadFile(installer)
	if err != nil {
		t.Fatal(err)
	}
	installer = filepath.Join(source, "install-lumen-host")
	if err := os.WriteFile(installer, installerBody, 0700); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatal(err)
	}
	pkgLog := filepath.Join(root, "pkg.log")
	if err := os.WriteFile(filepath.Join(binDir, "pkg"), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >\"$PKG_LOG\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "home", "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "home", "bin", "lumen-host"), []byte("old-binary"), 0700); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("sh", installer)
	command.Dir = source
	command.Env = append(os.Environ(), "HOME="+filepath.Join(root, "home"), "PATH="+binDir+":"+os.Getenv("PATH"), "LUMEN_REINSTALL=1", "PKG_LOG="+pkgLog)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}

	for _, path := range []string{
		filepath.Join(root, "home", "bin", "lumen-host"),
		filepath.Join(root, "home", "service", "lumen-host", "run"),
		filepath.Join(root, "home", "service", "lumen-host", "finish"),
		filepath.Join(root, "home", ".termux", "boot", "lumen-host"),
	} {
		if st, err := os.Stat(path); err != nil || st.Mode()&0111 == 0 {
			t.Fatalf("missing executable %s: %v", path, err)
		}
	}
	boot, err := os.ReadFile(filepath.Join(root, "home", ".termux", "boot", "lumen-host"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"runsvdir -P \"$HOME/service\"", "exec"} {
		if !strings.Contains(string(boot), want) {
			t.Fatalf("Termux boot script missing %q: %s", want, boot)
		}
	}
	if got, err := os.ReadFile(filepath.Join(root, "home", "bin", "lumen-host")); err != nil || string(got) != "fixture" {
		t.Fatalf("installer did not replace the uninitialized Host binary: %v", err)
	}
	if got, err := os.ReadFile(pkgLog); err != nil || string(got) != "install -y runit jq openssl-tool\n" {
		t.Fatalf("installer did not request the Termux OpenSSL executable package: %v", err)
	}
	config := filepath.Join(root, "home", ".config", "lumen", "host.env")
	if st, err := os.Stat(config); err != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("host configuration is not private: %v", err)
	}
	if got, err := os.ReadFile(config); err != nil || string(got) != developmentConfig {
		t.Fatalf("installer did not move the bundled development configuration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(source, "host.env")); !os.IsNotExist(err) {
		t.Fatalf("installer retained the shared development configuration: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "home", ".local", "share", "lumen", "hermes.token")); err != nil || string(got) != "development-token" {
		t.Fatalf("installer did not move the development bearer token: %v", err)
	}
	if _, err := os.Stat(filepath.Join(source, "hermes.token")); !os.IsNotExist(err) {
		t.Fatalf("installer retained the shared development bearer token: %v", err)
	}
}

func TestTermuxRunitServiceExportsHostConfiguration(t *testing.T) {
	root, err := os.MkdirTemp("/private/tmp", "lumen-termux-runit-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".config", "lumen"), 0700); err != nil {
		t.Fatal(err)
	}
	config := "LUMEN_HERMES_PROFILE=development\nLUMEN_HERMES_BASE_URL=http://127.0.0.1:8642\n"
	if err := os.WriteFile(filepath.Join(home, ".config", "lumen", "host.env"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "bin", "lumen-host"), []byte("#!/bin/sh\nprintf '%s\\n%s\\n' \"$LUMEN_HERMES_PROFILE\" \"$LUMEN_HERMES_BASE_URL\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	run, err := filepath.Abs("../../deploy/termux/run")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", run)
	command.Env = append(os.Environ(), "HOME="+home)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("runit service failed: %v\n%s", err, output)
	}
	for _, want := range []string{"development", "http://127.0.0.1:8642"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("runit service did not export %q: %s", want, output)
		}
	}
}
