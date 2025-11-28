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

type Config struct {
	ScanPaths       []string `json:"scan_paths"`
	BadPackageLists []string `json:"bad_package_lists"`
	ScanInterval    string   `json:"scan_interval"` // e.g., "12h", "24h"
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
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.BoolVar(&showVersion, "v", false, "Show version and exit (shorthand)")
	flag.StringVar(&intervalFlag, "interval", "", "Run periodically with this interval (e.g. 12h). If omitted the program performs a single run and exits.")
	flag.StringVar(&intervalFlag, "i", "", "Shorthand for --interval")
	flag.Parse()
	if showVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
	configPath := getConfigPath()

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Could not find home directory: %v", err)
	}

	configDir := filepath.Join(homeDir, ".dewormer")
	os.MkdirAll(configDir, 0755)
	
	return filepath.Join(configDir, "config.json")
}

func createDefaultConfig(path string) error {
	homeDir, _ := os.UserHomeDir()
	
	defaultConfig := Config{
		ScanPaths: []string{
			filepath.Join(homeDir, "projects"),
		},
		BadPackageLists: []string{
			filepath.Join(homeDir, ".dewormer", "bad_package_lists", "npm-malicious.txt"),
		},
		ScanInterval:   "12h",
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

	// also include every file under ~/.dewormer/bad_package_lists
	if home, err := os.UserHomeDir(); err == nil {
		listsDir := filepath.Join(home, ".dewormer", "bad_package_lists")
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
					filesScanned++
					deps, err := r.ReadDependencies(path)
					if err != nil {
						log.Printf("could not read dependencies with %s: %v", r.Name(), err)
					} else {
						matches := findMatches(deps, badPackages, path)
						results = append(results, matches...)
					}

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

// dependency readers (package-lock.json, pom.xml) live in the readers/ package.

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