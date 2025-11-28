package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/joelcma/dewormer/readers"
	statepkg "github.com/joelcma/dewormer/state"
)

// Version can be overridden at build time like this: go build -ldflags "-X main.Version=1.2.3" -o dewormer
var Version = "dev"

// ConfigPathOverride will be used when the user passes --config on the CLI.
// When non-empty getConfigPath() will return this value instead of the default
// ~/.dewormer/config.json path.
var ConfigPathOverride string

// BadListsDirOverride when set will make runScan look for bad package list files
// in the specified directory instead of the default ~/.dewormer/bad_package_lists
var BadListsDirOverride string

type Config struct {
	ScanPaths       []string `json:"scan_paths"`
	BadPackageLists []string `json:"bad_package_lists"`
}

type ScanResult struct {
	Package string
	Version string
	File    string
	List    string
}

func main() {
	// CLI flags
	var showVersion bool
	var intervalFlag string
	var configFlag string
	var badListsFlag string
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.BoolVar(&showVersion, "v", false, "Show version and exit (shorthand)")
	flag.StringVar(&intervalFlag, "interval", "", "Run periodically with this interval (e.g. 12h). If omitted the program performs a single run and exits.")
	flag.StringVar(&configFlag, "config", "", "Path to config.json (default: ~/.dewormer/config.json)")
	flag.StringVar(&badListsFlag, "bad-package-files", "", "Path to a directory containing bad-package list files (default: ~/.dewormer/bad_package_lists)")
	flag.StringVar(&badListsFlag, "b", "", "Shorthand for --bad-package-files")
	flag.StringVar(&intervalFlag, "i", "", "Shorthand for --interval")
	flag.Parse()
	if showVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
	// Determine which config path to use. CLI flag takes precedence.
	var configPath string
	if configFlag != "" {
		configPath = configFlag
	} else {
		configPath = getConfigPath()
	}

	// Make the resolved path available to helper functions that call getConfigPath().
	ConfigPathOverride = configPath

	// Bad lists dir flagged by user? make it available to runScan through a global
	// override variable (helpers call os.UserHomeDir which respects HOME, so this
	// keeps parity with getConfigPath override semantics).
	if badListsFlag != "" {
		BadListsDirOverride = badListsFlag
	}

	// Check if config exists, create default if not
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			log.Fatalf("Failed to create default config: %v", err)
		}
		fmt.Printf("Created default config at: %s\n", configPath)
		fmt.Println("Please edit the config file to add your scan paths and bad package lists.")
		os.Exit(0)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var interval time.Duration

	if intervalFlag == "" {
		// no --interval provided -> single-run mode
		log.Println("Dewormer started in single-run mode (no --interval provided)")
	} else {
		interval, err = time.ParseDuration(intervalFlag)
		if err != nil {
			log.Printf("Invalid --interval value, defaulting to 12h: %v", err)
			interval = 12 * time.Hour
		}

		log.Printf("Dewormer started. Scanning every %s", interval)
	}

	// Run initial scan immediately
	runScan(config)

	// If --interval wasn't provided then we run a single scan and exit.
	if intervalFlag == "" {
		log.Println("--interval not provided — single run complete, exiting")
		return
	}

	// Run periodic scans
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		runScan(config)
	}
}

func getConfigPath() string {
	// If CLI override is set, return it.
	if ConfigPathOverride != "" {
		return ConfigPathOverride
	}
	// prefer $HOME environment variable if set (makes testing easier)
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			log.Fatalf("Could not find home directory: %v", err)
		}
	}

	configDir := filepath.Join(homeDir, ".dewormer")
	os.MkdirAll(configDir, 0755)

	return filepath.Join(configDir, "config.json")
}

func createDefaultConfig(path string) error {
	// Ensure parent directory for the config exists (handles custom --config paths)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	homeDir, _ := os.UserHomeDir()

	defaultConfig := Config{
		ScanPaths: []string{
			filepath.Join(homeDir, "projects"),
		},
		BadPackageLists: []string{
			filepath.Join(homeDir, ".dewormer", "bad_package_lists", "npm-malicious.txt"),
		},
	}

	// Create bad package lists directory
	listsDir := filepath.Join(homeDir, ".dewormer", "bad_package_lists")
	os.MkdirAll(listsDir, 0755)

	// Create example bad package list
	exampleList := filepath.Join(listsDir, "npm-malicious.txt")
	exampleContent := `# Example bad package list
# Format: package@version (one per line)
# Lines starting with # are comments

voip-callkit@1.0.2
voip-callkit@1.0.3
eslint-config-teselagen@6.1.7
@rxap/ngx-bootstrap@19.0.3
`
	os.WriteFile(exampleList, []byte(exampleContent), 0644)

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func runScan(config *Config) {
	log.Println("Starting scan...")
	startTime := time.Now()

	// Build the list of bad package list files to load.
	// We use any entries in config.BadPackageLists plus every file found in
	// ~/.dewormer/bad_package_lists so users don't need to enumerate each file.
	listPaths := make([]string, 0, len(config.BadPackageLists))
	// add configured lists first (may be empty)
	listPaths = append(listPaths, config.BadPackageLists...)

	// also include every file under ~/.dewormer/bad_package_lists (or an
	// override directory supplied by --bad-package-files).
	listsDir := ""
	if BadListsDirOverride != "" {
		listsDir = BadListsDirOverride
	} else if home, err := os.UserHomeDir(); err == nil {
		listsDir = filepath.Join(home, ".dewormer", "bad_package_lists")
	}

	if listsDir != "" {
		if entries, err := os.ReadDir(listsDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				full := filepath.Join(listsDir, e.Name())
				// avoid duplicates
				found := false

				for _, p := range listPaths {
					if p == full {
						found = true
						break
					}
				}
				if !found {
					listPaths = append(listPaths, full)
				}
			}
		}
	}

	// Load all bad packages
	badPackages := loadBadPackages(listPaths)
	log.Printf("Loaded %d bad packages from %d lists", len(badPackages), len(listPaths))

	// compute latest modtime of the bad-package lists; we'll use this to
	// determine whether a given package file needs scanning. If any list has
	// changed more recently than the package file we should check it.
	var latestListMod time.Time

	// Use same config dir as getConfigPath to determine where to persist
	// the scan state so it's always colocated with the config file.
	cfgPath := getConfigPath()
	scanStatePath := filepath.Join(filepath.Dir(cfgPath), "scan_state.json")

	log.Printf("Loading scan state from %s", scanStatePath)
	state := statepkg.LoadScanState(scanStatePath)
	for _, p := range listPaths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}

		// (state already loaded)
		if st.ModTime().After(latestListMod) {
			latestListMod = st.ModTime()
		}
	}

	// initialize available readers
	readersList := []readers.DependencyReader{
		readers.NewPackageLockReader(),
		readers.NewPomReader(),
	}

	var results []ScanResult
	filesScanned := 0

	// Scan all configured paths
	for _, scanPath := range config.ScanPaths {
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			log.Printf("Scan path does not exist: %s", scanPath)
			continue
		}

		log.Printf("Scanning path: %s", scanPath)

		filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			if info.IsDir() {
				return nil
			}

			fileFound := false
			for _, r := range readersList {
				if r.Supports(info.Name()) {
					// decide whether we need to scan this file using persisted
					// state. We'll re-scan when any of the following is true:
					//  - We've never scanned this file before
					//  - The package file changed since we last scanned it
					//  - Any bad-package list was modified since we last scanned it
					pkgMod := info.ModTime()

					// normalize path key (absolute + clean) so persisted state
					// matches across runs regardless of how the scan was started
					absPath := path
					if !filepath.IsAbs(absPath) {
						if a, err := filepath.Abs(path); err == nil {
							absPath = a
						}
					}
					absPath = filepath.Clean(absPath)

					var lastScan time.Time
					if ts, ok := state[absPath]; ok && ts > 0 {
						lastScan = time.Unix(0, ts)
					}

					needScan := lastScan.IsZero() || pkgMod.After(lastScan) || latestListMod.After(lastScan)

					if !needScan {
						log.Printf("Skipping scan for %s (no changes since last scan at %s)", path, lastScan)
						return nil
					}

					filesScanned++
					deps, err := r.ReadDependencies(path)
					if err != nil {
						log.Printf("could not read dependencies with %s: %v", r.Name(), err)
					} else {
						matches := findMatches(deps, badPackages, path)
						results = append(results, matches...)
					}

					// Mark file as scanned now (store UnixNano)
					state[absPath] = time.Now().UnixNano()

					fileFound = true
					break
				}
			}

			if fileFound {
				log.Printf("Scanned: %s", path)
			}

			return nil
		})
	}

	// persist scan state
	if scanStatePath != "" {
		if err := statepkg.SaveScanState(scanStatePath, state); err != nil {
			log.Printf("Failed to save scan state: %v", err)
		}
	}

	duration := time.Since(startTime)
	log.Printf("Scan completed in %s. Files scanned: %d", duration, filesScanned)

	if len(results) > 0 {
		log.Printf("⚠️  WARNING: Found %d infected dependencies!", len(results))
		for _, result := range results {
			log.Printf("  - %s@%s in %s (matched: %s)", result.Package, result.Version, result.File, result.List)
		}

		// Show desktop notification
		message := fmt.Sprintf("Found %d infected dependencies! Check logs for details.", len(results))
		beeep.Alert("Dewormer - Threats Detected", message, "")
	} else {
		log.Println("✓ No threats detected")
	}
}

func loadBadPackages(listPaths []string) map[string]map[string]string {
	// Map of package@version -> list filename
	badPackages := make(map[string]map[string]string)

	for _, listPath := range listPaths {
		file, err := os.Open(listPath)
		if err != nil {
			log.Printf("Could not open bad package list %s: %v", listPath, err)
			continue
		}
		defer file.Close()

		listName := filepath.Base(listPath)
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Skip comments and empty lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Expected format: package@version
			parts := strings.Split(line, "@")
			if len(parts) < 2 {
				continue
			}

			pkg := strings.Join(parts[:len(parts)-1], "@") // Handle scoped packages like @rxap/ngx-bootstrap
			version := parts[len(parts)-1]

			if badPackages[pkg] == nil {
				badPackages[pkg] = make(map[string]string)
			}
			badPackages[pkg][version] = listName
		}
	}

	return badPackages
}

func findMatches(deps map[string]string, badPackages map[string]map[string]string, filePath string) []ScanResult {
	var results []ScanResult

	for pkg, version := range deps {
		if badVersions, exists := badPackages[pkg]; exists {
			if listName, isBad := badVersions[version]; isBad {
				results = append(results, ScanResult{
					Package: pkg,
					Version: version,
					File:    filePath,
					List:    listName,
				})
			}
		}
	}

	return results
}
