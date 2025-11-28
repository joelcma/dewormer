package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadScanState_Normalization(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scan_state.json")

	// create a state with a relative key
	state := map[string]int64{"./some/file": time.Now().Add(-time.Hour).UnixNano()}

	if err := SaveScanState(path, state); err != nil {
		t.Fatalf("SaveScanState: %v", err)
	}

	loaded := LoadScanState(path)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry after load, got %d", len(loaded))
	}

	// ensure the key was normalized into an absolute cleaned path
	for k := range loaded {
		if !filepath.IsAbs(k) {
			t.Fatalf("expected normalized absolute key, got %q", k)
		}
	}

	// cleanup
	os.Remove(path)
}
