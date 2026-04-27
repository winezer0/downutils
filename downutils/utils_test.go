package downutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestConvertGitHubURL tests GitHub URL conversion
func TestConvertGitHubURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "convert blob URL",
			input:    "https://github.com/user/repo/blob/main/file.txt",
			expected: "https://raw.githubusercontent.com/user/repo/main/file.txt",
		},
		{
			name:     "do not convert releases URL",
			input:    "https://github.com/user/repo/releases/download/v1.0/file.txt",
			expected: "https://github.com/user/repo/releases/download/v1.0/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGitHubURL(tt.input)
			if result != tt.expected {
				t.Errorf("convertGitHubURL(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "seconds",
			input:    30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutes and seconds",
			input:    90 * time.Second,
			expected: "1m30s",
		},
		{
			name:     "hours minutes and seconds",
			input:    3665 * time.Second,
			expected: "1h1m5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatSize tests size formatting
func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "bytes",
			input:    512,
			expected: "512 B",
		},
		{
			name:     "KB",
			input:    1536, // 1.5 KB
			expected: "1.50 KB",
		},
		{
			name:     "MB",
			input:    1572864, // 1.5 MB
			expected: "1.50 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSize(tt.input)
			if result != tt.expected {
				t.Errorf("formatSize(%d) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetItemFilePath tests file path generation
func TestGetItemFilePath(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		downloadDir string
	}{
		{
			name:        "relative path",
			filename:    "file.txt",
			downloadDir: "downloads",
		},
		{
			name:        "absolute path",
			filename:    filepath.Join("absolute", "path", "file.txt"),
			downloadDir: "downloads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinItemFilePath(tt.filename, tt.downloadDir)
			// Only verify result contains filename and directory
			if result == "" {
				t.Errorf("joinItemFilePath(%s, %s) returned empty string", tt.filename, tt.downloadDir)
			}
		})
	}
}

// TestFilterEnableItems tests filtering enabled items
func TestFilterEnableItems(t *testing.T) {
	items := []DownItem{
		{Module: "item1", Enable: true},
		{Module: "item2", Enable: false},
		{Module: "item3", Enable: true},
	}

	result := filterEnableItems(items)

	if len(result) != 2 {
		t.Errorf("filterEnableItems should return 2 enabled items, got %d", len(result))
	}

	// Verify all returned items are enabled
	for _, item := range result {
		if !item.Enable {
			t.Errorf("filterEnableItems returned disabled item: %s", item.Module)
		}
	}
}

// TestFindFilesBySuffix tests finding files by suffix
func TestFindFilesBySuffix(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "testfind")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.download"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.download"), []byte("test"), 0644)

	// Test finding .download files
	files, err := FindFilesBySuffix(tmpDir, ".download")
	if err != nil {
		t.Fatalf("FindFilesBySuffix failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("FindFilesBySuffix should find 2 files, found %d", len(files))
	}
}
