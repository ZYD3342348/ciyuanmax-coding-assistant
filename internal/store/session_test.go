package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadUsesPrivateAtomicState(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("CIYUANMAX_CONFIG_DIR", directory)
	want := Session{AccessToken: "cli_test", DeviceName: "test-machine", BaseURL: "https://example.test"}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != want.AccessToken || got.DeviceName != want.DeviceName || got.BaseURL != want.BaseURL {
		t.Fatalf("loaded %#v, want %#v", got, want)
	}
	info, err := os.Stat(filepath.Join(directory, stateFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("session mode = %o, want 600", info.Mode().Perm())
	}
}
