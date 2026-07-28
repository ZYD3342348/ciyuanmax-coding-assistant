package configure

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/api"
	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/fsutil"
	"github.com/pelletier/go-toml/v2"
)

type Paths struct {
	ClaudeSettings string
	CodexConfig    string
}

type Result struct {
	Tool       string
	Path       string
	BackupPath string
}

func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("find home directory: %w", err)
	}
	return Paths{
		ClaudeSettings: filepath.Join(home, ".claude", "settings.json"),
		CodexConfig:    filepath.Join(home, ".codex", "config.toml"),
	}, nil
}

func Apply(paths Paths, key api.Key) (Result, error) {
	switch strings.ToLower(strings.TrimSpace(key.Tool)) {
	case "claude":
		return applyClaude(paths.ClaudeSettings, key)
	case "codex":
		return applyCodex(paths.CodexConfig, key)
	default:
		return Result{}, fmt.Errorf("unsupported tool %q", key.Tool)
	}
}

func applyClaude(path string, key api.Key) (Result, error) {
	document := map[string]any{}
	data, err := os.ReadFile(path)
	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &document); err != nil {
			return Result{}, fmt.Errorf("parse existing Claude settings %s: %w", path, err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Result{}, fmt.Errorf("read Claude settings %s: %w", path, err)
	}

	environment, ok := document["env"].(map[string]any)
	if !ok {
		environment = map[string]any{}
	}
	environment["ANTHROPIC_BASE_URL"] = key.Endpoint
	environment["ANTHROPIC_AUTH_TOKEN"] = key.Key
	environment["ANTHROPIC_MODEL"] = key.Model
	environment["ANTHROPIC_DEFAULT_SONNET_MODEL"] = key.Model
	document["env"] = environment

	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode Claude settings: %w", err)
	}
	encoded = append(encoded, '\n')
	backupPath, err := replaceWithBackup(path, encoded, 0600)
	if err != nil {
		return Result{}, err
	}
	return Result{Tool: "claude", Path: path, BackupPath: backupPath}, nil
}

func applyCodex(path string, key api.Key) (Result, error) {
	document := map[string]any{}
	data, err := os.ReadFile(path)
	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := toml.Unmarshal(data, &document); err != nil {
			return Result{}, fmt.Errorf("parse existing Codex config %s: %w", path, err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Result{}, fmt.Errorf("read Codex config %s: %w", path, err)
	}

	document["model"] = key.Model
	document["model_provider"] = "ciyuanmax"
	providers, ok := document["model_providers"].(map[string]any)
	if !ok {
		providers = map[string]any{}
	}
	provider := map[string]any{
		"name":                      "CiYuanMax",
		"base_url":                  key.Endpoint,
		"wire_api":                  key.WireAPI,
		"experimental_bearer_token": key.Key,
	}
	providers["ciyuanmax"] = provider
	document["model_providers"] = providers

	encoded, err := toml.Marshal(document)
	if err != nil {
		return Result{}, fmt.Errorf("encode Codex config: %w", err)
	}
	backupPath, err := replaceWithBackup(path, encoded, 0600)
	if err != nil {
		return Result{}, err
	}
	return Result{Tool: "codex", Path: path, BackupPath: backupPath}, nil
}

func replaceWithBackup(path string, data []byte, mode os.FileMode) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("config path is empty")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", fmt.Errorf("create config directory %s: %w", directory, err)
	}
	backupPath := ""
	if _, err := os.Stat(path); err == nil {
		backupPath = path + ".ciyuanmax-backup-" + time.Now().Format("20060102-150405.000000000")
		if err := copyFile(path, backupPath); err != nil {
			return "", fmt.Errorf("back up %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect config %s: %w", path, err)
	}
	temporary, err := os.CreateTemp(directory, ".ciyuanmax-config-*.tmp")
	if err != nil {
		return backupPath, fmt.Errorf("create temporary config: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return backupPath, fmt.Errorf("protect temporary config: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return backupPath, fmt.Errorf("write temporary config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return backupPath, fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return backupPath, fmt.Errorf("close temporary config: %w", err)
	}
	if err := fsutil.Replace(temporaryName, path); err != nil {
		return backupPath, fmt.Errorf("replace config %s: %w", path, err)
	}
	return backupPath, nil
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}
