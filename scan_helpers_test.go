package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldScan_Behavior(t *testing.T) {
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "package-lock.json")
	if err := os.WriteFile(fpath, []byte("{}"), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	info, err := os.Stat(fpath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Case 1: no last-scan entry -> should scan
	state := map[string]int64{}
	_, last, need := shouldScan(fpath, info, time.Time{}, state)
	if !need {
		t.Fatalf("expected needScan when lastScan missing, got need=%v last=%v", need, last)
	}

	// Case 2: lastScan after file modification -> do not scan
	later := time.Now().Add(time.Hour)
	state = map[string]int64{}
	abs := filepath.Clean(fpath)
	state[abs] = later.UnixNano()
	_, last2, need2 := shouldScan(fpath, info, time.Time{}, state)
	if need2 {
		t.Fatalf("expected skip when lastScan after file mod, got need=%v last=%v", need2, last2)
	}

	// Case 3: lastScan older than latestListMod -> should scan
	old := time.Now().Add(-time.Hour)
	state = map[string]int64{abs: old.UnixNano()}
	latestListMod := time.Now()
	_, last3, need3 := shouldScan(fpath, info, latestListMod, state)
	if !need3 {
		t.Fatalf("expected needScan when list mod is newer, got need=%v last=%v", need3, last3)
	}
}
