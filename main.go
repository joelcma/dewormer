package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
)

type Config struct {
	ScanPaths       []string `json:"scan_paths"`
	BadPackageLists []string `json:"bad_package_lists"`
	ScanInterval    string   `json:"scan_interval"` // e.g., "12h", "24h"
}

type PackageLock struct {
	Packages map[string]PackageInfo `json:"packages"`
}

type PackageInfo struct {
	Version string `json:"version"`
}

type PomXML struct {
	Dependencies struct {
		Dependency []Dependency `xml:"dependency"`
	} `xml:"dependencies"`
}

type Dependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type ScanResult struct {
	Package string
	Version string
	File    string
	List    string
}

func main() {
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

	// Parse scan interval
	interval, err := time.ParseDuration(config.ScanInterval)
	if err != nil {
		log.Printf("Invalid scan interval, defaulting to 12h: %v", err)
		interval = 12 * time.Hour
	}

	log.Printf("Dewormer started. Scanning every %s", interval)
	
	// Run initial scan immediately
	runScan(config)

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
		ScanInterval: "12h",
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

	// Load all bad packages
	badPackages := loadBadPackages(config.BadPackageLists)
	log.Printf("Loaded %d bad packages from %d lists", len(badPackages), len(config.BadPackageLists))

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

			fileFound := true
			if info.Name() == "package-lock.json" {
				filesScanned++
				deps := extractNpmDependencies(path)
				matches := findMatches(deps, badPackages, path)
				results = append(results, matches...)
			} else if info.Name() == "pom.xml" {
				filesScanned++
				deps := extractMavenDependencies(path)
				matches := findMatches(deps, badPackages, path)
				results = append(results, matches...)
			} else {
				fileFound = false
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

func extractNpmDependencies(path string) map[string]string {
	deps := make(map[string]string)

	data, err := os.ReadFile(path)
	if err != nil {
		return deps
	}

	var packageLock PackageLock
	if err := json.Unmarshal(data, &packageLock); err != nil {
		return deps
	}

	for pkgPath, info := range packageLock.Packages {
		// Skip the root package (empty string key)
		if pkgPath == "" {
			continue
		}

		// Extract package name from path (e.g., "node_modules/express" -> "express")
		pkgName := strings.TrimPrefix(pkgPath, "node_modules/")
		
		if info.Version != "" {
			deps[pkgName] = info.Version
		}
	}

	return deps
}

func extractMavenDependencies(path string) map[string]string {
	deps := make(map[string]string)

	data, err := os.ReadFile(path)
	if err != nil {
		return deps
	}

	var pom PomXML
	if err := xml.Unmarshal(data, &pom); err != nil {
		return deps
	}

	for _, dep := range pom.Dependencies.Dependency {
		if dep.Version != "" {
			// Format: groupId:artifactId
			pkgName := fmt.Sprintf("%s:%s", dep.GroupID, dep.ArtifactID)
			deps[pkgName] = dep.Version
		}
	}

	return deps
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