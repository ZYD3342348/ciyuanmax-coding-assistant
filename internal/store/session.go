package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/fsutil"
)

const stateFileName = "session.json"

type Session struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	DeviceName  string    `json:"device_name"`
	BaseURL     string    `json:"base_url"`
}

func ConfigDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("CIYUANMAX_CONFIG_DIR")); override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(base, "CiYuanMax"), nil
	}
	return filepath.Join(base, "ciyuanmax"), nil
}

func StatePath() (string, error) {
	directory, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, stateFileName), nil
}

func Load() (Session, error) {
	path, err := StatePath()
	if err != nil {
		return Session{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Session{BaseURL: "https://llm.ciyuanmax.art", DeviceName: DefaultDeviceName()}, nil
	}
	if err != nil {
		return Session{}, fmt.Errorf("read session: %w", err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("decode session: %w", err)
	}
	if session.BaseURL == "" {
		session.BaseURL = "https://llm.ciyuanmax.art"
	}
	if session.DeviceName == "" {
		session.DeviceName = DefaultDeviceName()
	}
	return session, nil
}

func Save(session Session) error {
	directory, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	path := filepath.Join(directory, stateFileName)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(directory, ".session-*.tmp")
	if err != nil {
		return fmt.Errorf("create session temporary file: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect session temporary file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write session: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	if err := fsutil.Replace(temporaryName, path); err != nil {
		return fmt.Errorf("replace session: %w", err)
	}
	return nil
}

func DefaultDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown-device"
	}
	hostname = strings.TrimSpace(hostname)
	if len(hostname) > 64 {
		hostname = hostname[:64]
	}
	return hostname
}

func NewDeviceName() string {
	name := DefaultDeviceName()
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err == nil {
		name += "-" + hex.EncodeToString(suffix[:])
	}
	return name
}
