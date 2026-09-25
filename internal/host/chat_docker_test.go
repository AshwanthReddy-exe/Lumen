package host

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AshwanthReddy-exe/Lumen/internal/conversation"
	"github.com/AshwanthReddy-exe/Lumen/internal/space"
)

func TestDockerChatCertifierBindsQualifiedContainer(t *testing.T) {
	d := t.TempDir()
	configPath := filepath.Join(d, "chat-config.yaml")
	tokenPath := filepath.Join(d, "chat.token")
	probePath := filepath.Join(d, "probe.json")
	for path, content := range map[string]string{configPath: "zero-tool-config", tokenPath: "synthetic-test-key"} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	image := "sha256:" + strings.Repeat("a", 64)
	containerID := strings.Repeat("b", 64)
	imageEnv := []string{"PATH=/usr/bin", "LANG=C.UTF-8", "GPG_KEY=test", "PYTHON_VERSION=3.12", "PYTHON_SHA256=test"}
	goodEnv := append(append([]string{}, imageEnv...), "HERMES_SAFE_MODE=1", "LUMEN_CHAT_ZERO_TOOL=1", "HERMES_HOME=/var/lib/hermes", "TERMINAL_CWD=/var/lib/hermes", "API_SERVER_ENABLED=true", "API_SERVER_HOST=0.0.0.0", "API_SERVER_PORT=8642", "API_SERVER_KEY=synthetic-test-key", "API_SERVER_MODEL_NAME=synthetic-model", "HERMES_INFERENCE_PROVIDER=custom", "HERMES_INFERENCE_MODEL=synthetic-model", "OPENAI_API_KEY=synthetic-only", "OPENROUTER_API_KEY=synthetic-only", "OPENAI_BASE_URL=http://host.docker.internal:12345/v1", "OPENROUTER_BASE_URL=http://host.docker.internal:12345/v1")
	probe, _ := json.Marshal(map[string]any{"status": "container_probe_passed", "imageId": image, "configDigest": space.DigestText("zero-tool-config"), "providerCalls": 4, "rejectedToolCalls": 3})
	if err := os.WriteFile(probePath, probe, 0600); err != nil {
		t.Fatal(err)
	}
	inspect := map[string]any{
		"Id": containerID, "Image": image,
		"State":      map[string]any{"Running": true, "Pid": 535, "StartedAt": "2026-09-25T18:16:46.261978666Z"},
		"Config":     map[string]any{"User": "65532:65532", "Entrypoint": []string{"/src/.venv/bin/hermes"}, "Cmd": []string{"gateway", "run", "--no-supervise", "--force"}, "Env": goodEnv},
		"HostConfig": map[string]any{"ReadonlyRootfs": true, "Privileged": false, "PortBindings": map[string]any{"8642/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "28643"}}}, "Tmpfs": map[string]string{"/var/lib/hermes": "rw,uid=65532,gid=65532,mode=0700"}},
		"Mounts":     []any{map[string]any{"Source": configPath, "Destination": "/var/lib/hermes/config.yaml", "RW": false}},
	}
	writeInspect := func() {
		t.Helper()
		b, _ := json.Marshal([]any{inspect})
		fixture := filepath.Join(d, "inspect.json")
		if err := os.WriteFile(fixture, b, 0600); err != nil {
			t.Fatal(err)
		}
		toolDir := filepath.Join(d, "bin")
		if err := os.MkdirAll(toolDir, 0700); err != nil {
			t.Fatal(err)
		}
		imageFixture := filepath.Join(d, "image.json")
		imageBytes, _ := json.Marshal([]any{map[string]any{"Config": map[string]any{"Env": imageEnv}}})
		if err := os.WriteFile(imageFixture, imageBytes, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(toolDir, "docker"), []byte("#!/bin/sh\nif [ \"$1\" = image ]; then exec /bin/cat '"+imageFixture+"'; fi\nexec /bin/cat '"+fixture+"'\n"), 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", toolDir+":"+os.Getenv("PATH"))
	}
	writeInspect()
	cfg := Config{HermesProfile: "development", ChatBaseURL: "http://127.0.0.1:28643", ChatBearerPath: tokenPath, ChatContainerID: containerID, ChatConfigPath: configPath, ChatProbePath: probePath, ChatProviderURL: "http://host.docker.internal:12345/v1"}
	certifier := newDockerChatCertifier(cfg, &fakeRuntime{})
	profile := space.DefaultRuntimeProfile()
	cert, err := certifier.Certify(context.Background(), space.State{}, profile)
	if err != nil || cert.ArtifactDigest != image || cert.ProcessIdentity == "" || cert.ConfigDigest != space.DigestText("zero-tool-config") || cert.EndpointIdentity != "endpoint:test" {
		t.Fatalf("qualified container certificate=%#v err=%v", cert, err)
	}
	cfg.ChatProviderURL = "http://host.docker.internal:54321/v1"
	if _, err := newDockerChatCertifier(cfg, &fakeRuntime{}).Certify(context.Background(), space.State{}, profile); !errors.Is(err, conversation.ErrRuntimeProfileUnverified) {
		t.Fatalf("provider destination mismatch was certified: %v", err)
	}
	config := inspect["Config"].(map[string]any)
	imageEnv = append(imageEnv, "HTTP_PROXY=http://untrusted.example:8080")
	config["Env"] = append(append([]string{}, goodEnv...), "HTTP_PROXY=http://untrusted.example:8080")
	writeInspect()
	if _, err := certifier.Certify(context.Background(), space.State{}, profile); !errors.Is(err, conversation.ErrRuntimeProfileUnverified) {
		t.Fatalf("inherited proxy was certified: %v", err)
	}
	imageEnv = imageEnv[:len(imageEnv)-1]
	config["Env"] = goodEnv
	inspect["Image"] = "sha256:" + strings.Repeat("c", 64)
	writeInspect()
	if _, err := certifier.Certify(context.Background(), space.State{}, profile); !errors.Is(err, conversation.ErrRuntimeProfileUnverified) {
		t.Fatalf("changed image was certified: %v", err)
	}
	inspect["Image"] = image
	config["Env"] = append([]string{}, goodEnv...)
	config["Env"].([]string)[len(imageEnv)] = "HERMES_SAFE_MODE=0"
	writeInspect()
	if _, err := certifier.Certify(context.Background(), space.State{}, profile); !errors.Is(err, conversation.ErrRuntimeProfileUnverified) {
		t.Fatalf("disabled safe mode was certified: %v", err)
	}
	config["Env"] = append([]string{}, goodEnv...)
	config["Env"] = append(config["Env"].([]string), "OPENAI_BASE_URL=https://untrusted.example/v1")
	writeInspect()
	if _, err := certifier.Certify(context.Background(), space.State{}, profile); !errors.Is(err, conversation.ErrRuntimeProfileUnverified) {
		t.Fatalf("unexpected provider override was certified: %v", err)
	}
	config["Env"] = append([]string{}, goodEnv...)
	host := inspect["HostConfig"].(map[string]any)
	host["PortBindings"] = map[string]any{"8642/tcp": []any{map[string]any{"HostIp": "0.0.0.0", "HostPort": "28643"}}}
	writeInspect()
	if _, err := certifier.Certify(context.Background(), space.State{}, profile); !errors.Is(err, conversation.ErrRuntimeProfileUnverified) {
		t.Fatalf("public chat port was certified: %v", err)
	}
	host["PortBindings"] = map[string]any{"8642/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "28643"}}}
	inspect["Mounts"] = []any{map[string]any{"Source": configPath, "Destination": "/var/lib/hermes/config.yaml", "RW": true}}
	writeInspect()
	if _, err := certifier.Certify(context.Background(), space.State{}, profile); !errors.Is(err, conversation.ErrRuntimeProfileUnverified) {
		t.Fatalf("writable config mount was certified: %v", err)
	}
}
