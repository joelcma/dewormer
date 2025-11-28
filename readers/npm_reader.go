package readers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type PackageLockReader struct{}

func NewPackageLockReader() DependencyReader { return &PackageLockReader{} }

func (r *PackageLockReader) Name() string { return "package-lock.json" }

func (r *PackageLockReader) Supports(filename string) bool {
	return filename == "package-lock.json"
}

// internal structures mirror the shape of npm's package-lock.json
type packageLock struct {
	Packages map[string]packageInfo `json:"packages"`
}

type packageInfo struct {
	Version string `json:"version"`
}

func (r *PackageLockReader) ReadDependencies(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var pl packageLock
	if err := json.Unmarshal(data, &pl); err != nil {
		return nil, fmt.Errorf("unmarshal package-lock: %w", err)
	}

	deps := make(map[string]string)
	for pkgPath, info := range pl.Packages {
		if pkgPath == "" { // skip root
			continue
		}

		// Example path: node_modules/express or @scope/node_modules/...
		pkgName := strings.TrimPrefix(pkgPath, "node_modules/")
		if info.Version != "" {
			deps[pkgName] = info.Version
		}
	}

	return deps, nil
}
