package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/api"
	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/store"
)

func TestParseTools(t *testing.T) {
	tools, err := parseTools("claude,gpt,claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 || tools[0] != "claude" || tools[1] != "codex" {
		t.Fatalf("tools = %#v", tools)
	}
	if _, err := parseTools("unknown"); err == nil {
		t.Fatal("expected unsupported tool error")
	}
}

func TestEnsureModeAvailable(t *testing.T) {
	modes := []api.Mode{{ID: "stable", Available: true, Tools: []string{"claude", "codex"}}}
	if err := ensureModeAvailable(modes, "stable", []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	if err := ensureModeAvailable(modes, "economy", []string{"codex"}); err == nil {
		t.Fatal("expected unavailable mode error")
	}
}

func TestConfigureCommandEndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer cli_test" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		switch request.URL.Path {
		case "/api/cli/modes":
			_ = json.NewEncoder(writer).Encode([]api.Mode{{ID: "stable", Available: true, Tools: []string{"claude", "codex"}}})
		case "/api/cli/keys/ensure":
			var requestBody api.EnsureKeyRequest
			if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
				t.Fatal(err)
			}
			key := api.Key{Tool: requestBody.Tool, Mode: requestBody.Mode, Key: "sk-" + requestBody.Tool, Endpoint: "https://gateway.test", Model: "test-model"}
			if requestBody.Tool == "codex" {
				key.Endpoint += "/v1"
				key.WireAPI = "responses"
			}
			_ = json.NewEncoder(writer).Encode(key)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	configDirectory := filepath.Join(home, "ciyuanmax-state")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CIYUANMAX_CONFIG_DIR", configDirectory)
	if err := store.Save(store.Session{AccessToken: "cli_test", DeviceName: "test-device", BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := configureCommand([]string{"--mode", "stable", "--tools", "claude,codex"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(home, ".claude", "settings.json"), filepath.Join(home, ".codex", "config.toml")} {
		if data, err := os.ReadFile(path); err != nil || len(data) == 0 {
			t.Fatalf("config %s not written: bytes=%d err=%v", path, len(data), err)
		}
	}
}
