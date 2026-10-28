package downutils

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCollectConfigDirectories 测试目录收集功能
func TestCollectConfigDirectories(t *testing.T) {
	config := DownConfig{
		"group1": []DownItem{
			{
				Module:     "test1",
				FileName:   "test1.txt",
				StorageDir: "./dir1",
				Enable:     true,
			},
			{
				Module:     "test2",
				FileName:   "test2.txt",
				StorageDir: "./dir2",
				Enable:     true,
			},
		},
		"group2": []DownItem{
			{
				Module:     "test3",
				FileName:   "test3.txt",
				StorageDir: "./dir1",
				Enable:     true,
			},
		},
	}

	// 测试不使用 outputForce
	dirs := collectConfigDirs(config, "")
	if len(dirs) != 2 {
		t.Errorf("collectConfigDirs() got %d dirs, want 2 dirs", len(dirs))
	}

	// 测试使用 outputForce
	dirs = collectConfigDirs(config, "./force-dir")
	if len(dirs) != 1 || dirs[0] != "./force-dir" {
		t.Errorf("collectConfigDirs() with outputForce got %v, want [./force-dir]", dirs)
	}
}

// TestCleanupIncompleteDownloads tests cleanup of incomplete downloads
func TestCleanupIncompleteDownloads(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "testcleanup")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.download"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)

	// Test cleanup
	err = cleanupIncompleteDownloads(tmpDir)
	if err != nil {
		t.Fatalf("cleanupIncompleteDownloads failed: %v", err)
	}

	// Verify .download file is deleted
	if fileExists(filepath.Join(tmpDir, "file1.download")) {
		t.Error("cleanupIncompleteDownloads should delete .download files")
	}

	// Verify other files are not deleted
	if !fileExists(filepath.Join(tmpDir, "file2.txt")) {
		t.Error("cleanupIncompleteDownloads should not delete non-.download files")
	}
}
