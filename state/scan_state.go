package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LoadScanState reads the scan-state file (JSON map[string]int64).
// It normalizes the keys to absolute cleaned paths before returning.
func LoadScanState(path string) map[string]int64 {
	state := make(map[string]int64)
	if path == "" {
		return state
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return state
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return make(map[string]int64)
	}

	// normalize keys to absolute cleaned paths so state is robust
	normalized := make(map[string]int64, len(state))
	for k, v := range state {
		nk := k
		if !filepath.IsAbs(nk) {
			if a, err := filepath.Abs(nk); err == nil {
				nk = a
			}
		}
		nk = filepath.Clean(nk)
		normalized[nk] = v
	}

	return normalized
}

// SaveScanState writes the scan state to disk atomically (tmp then rename).
func SaveScanState(path string, state map[string]int64) error {
	if path == "" {
		return nil
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
