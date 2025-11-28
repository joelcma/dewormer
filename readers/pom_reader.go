package readers

import (
	"encoding/xml"
	"fmt"
	"os"
)

type PomReader struct{}

func NewPomReader() DependencyReader { return &PomReader{} }

func (r *PomReader) Name() string { return "pom.xml" }

func (r *PomReader) Supports(filename string) bool {
    return filename == "pom.xml"
}

type pomXML struct {
    Dependencies struct {
        Dependency []pomDependency `xml:"dependency"`
    } `xml:"dependencies"`
}

type pomDependency struct {
    GroupID    string `xml:"groupId"`
    ArtifactID string `xml:"artifactId"`
    Version    string `xml:"version"`
}

func (r *PomReader) ReadDependencies(path string) (map[string]string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read file: %w", err)
    }

    var p pomXML
    if err := xml.Unmarshal(data, &p); err != nil {
        return nil, fmt.Errorf("unmarshal pom.xml: %w", err)
    }

    deps := make(map[string]string)
    for _, d := range p.Dependencies.Dependency {
        if d.Version == "" {
            continue
        }
        pkgName := d.GroupID + ":" + d.ArtifactID
        deps[pkgName] = d.Version
    }

    return deps, nil
}
