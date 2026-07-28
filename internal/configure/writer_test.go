package configure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/api"
	"github.com/pelletier/go-toml/v2"
)

func TestApplyClaudePreservesExistingSettingsAndBacksUp(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "settings.json")
	if err := os.WriteFile(path, []byte("{\"permissions\":{\"allow\":[\"Read\"]},\"env\":{\"KEEP\":\"yes\"}}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Apply(Paths{ClaudeSettings: path}, api.Key{Tool: "claude", Key: "sk-claude", Endpoint: "https://example.test", Model: "claude-test"})
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath == "" {
		t.Fatal("expected backup path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	environment := document["env"].(map[string]any)
	if environment["KEEP"] != "yes" || environment["ANTHROPIC_AUTH_TOKEN"] != "sk-claude" {
		t.Fatalf("unexpected env: %#v", environment)
	}
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Fatal(err)
	}
}

func TestApplyCodexPreservesUnrelatedConfig(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(path, []byte("sandbox_mode = \"workspace-write\"\n[features]\nweb_search = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Apply(Paths{CodexConfig: path}, api.Key{Tool: "codex", Key: "sk-codex", Endpoint: "https://example.test/v1", Model: "gpt-test", WireAPI: "responses"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document["sandbox_mode"] != "workspace-write" || document["model_provider"] != "ciyuanmax" {
		t.Fatalf("unexpected document: %#v", document)
	}
	if !strings.Contains(string(data), "experimental_bearer_token = 'sk-codex'") && !strings.Contains(string(data), "experimental_bearer_token = \"sk-codex\"") {
		t.Fatalf("missing configured token: %s", data)
	}
}
