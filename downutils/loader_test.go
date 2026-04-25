package downutils

import (
	"os"
	"testing"
)

// TestLoadConfig tests config file loading
func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "testconfig*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
databases:
  - module: test
    filename: test.txt
    download-urls:
      - https://example.com/test.txt
    keep-updated: true
    enable: true
`
	tmpFile.WriteString(configContent)
	tmpFile.Close()

	// Test loading config
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(config) != 1 {
		t.Errorf("LoadConfig should load 1 config group, loaded %d", len(config))
	}

	// Verify config content
	items, exists := config["databases"]
	if !exists {
		t.Fatal("LoadConfig did not find databases config group")
	}

	if len(items) != 1 || items[0].Module != "test" {
		t.Errorf("LoadConfig config content is incorrect")
	}
}
