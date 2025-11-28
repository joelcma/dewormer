package readers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPomReader_ReadDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "pom.xml")

	data := `<?xml version="1.0"?>
<project>
  <dependencies>
    <dependency>
      <groupId>com.example</groupId>
      <artifactId>evil</artifactId>
      <version>1.2.3</version>
    </dependency>
  </dependencies>
</project>`

	if err := os.WriteFile(fpath, []byte(data), 0644); err != nil {
		t.Fatalf("write tmp pom: %v", err)
	}

	r := NewPomReader()
	deps, err := r.ReadDependencies(fpath)
	if err != nil {
		t.Fatalf("ReadDependencies returned error: %v", err)
	}

	if deps["com.example:evil"] != "1.2.3" {
		t.Fatalf("expected com.example:evil=1.2.3 got=%q", deps["com.example:evil"])
	}
}
