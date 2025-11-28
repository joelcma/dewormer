package readers

// DependencyReader is an interface for reading dependency files (package-lock.json, pom.xml, etc.)
// Implementations must detect whether they support a filename and return a map of package->version.
type DependencyReader interface {
	// Name returns the reader's human-friendly name.
	Name() string

	// Supports returns true if the reader can parse a file with the provided filename.
	Supports(filename string) bool

	// ReadDependencies reads the file at path and returns a map[pkg]version or an error.
	ReadDependencies(path string) (map[string]string, error)
}
