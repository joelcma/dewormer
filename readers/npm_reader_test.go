package readers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageLockReader_ReadDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "package-lock.json")

	data := `{
  "packages": {
    "": { "version": "1.0.0" },
    "node_modules/left-pad": { "version": "1.2.3" },
    "node_modules/@scope/pkg": { "version": "0.1.0" }
  }
}`

	if err := os.WriteFile(fpath, []byte(data), 0644); err != nil {
		t.Fatalf("write tmp package-lock: %v", err)
	}

	r := NewPackageLockReader()
	deps, err := r.ReadDependencies(fpath)
	if err != nil {
		t.Fatalf("ReadDependencies returned error: %v", err)
	}

	if got := deps["left-pad"]; got != "1.2.3" {
		t.Fatalf("expected left-pad=1.2.3 got=%q", got)
	}
	if got := deps["@scope/pkg"]; got != "0.1.0" {
		t.Fatalf("expected @scope/pkg=0.1.0 got=%q", got)
	}
}
