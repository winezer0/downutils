package downutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestParseKeepUpdated 测试更新策略解析
func TestParseKeepUpdated(t *testing.T) {
	tests := []struct {
		name        string
		keepUpdated string
		fileModTime time.Time
		want        bool
	}{
		{
			name:        "empty string defaults to false",
			keepUpdated: "",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "disable strategy",
			keepUpdated: "disable",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "enable strategy",
			keepUpdated: "enable",
			fileModTime: time.Now(),
			want:        true,
		},
		{
			name:        "no strategy",
			keepUpdated: "no",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "yes strategy",
			keepUpdated: "yes",
			fileModTime: time.Now(),
			want:        true,
		},
		{
			name:        "false strategy",
			keepUpdated: "false",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "true strategy",
			keepUpdated: "true",
			fileModTime: time.Now(),
			want:        true,
		},
		{
			name:        "1h strategy with expired file",
			keepUpdated: "1h",
			fileModTime: time.Now().Add(-2 * time.Hour),
			want:        true,
		},
		{
			name:        "1h strategy with non-expired file",
			keepUpdated: "1h",
			fileModTime: time.Now().Add(-30 * time.Minute),
			want:        false,
		},
		{
			name:        "24h strategy with expired file",
			keepUpdated: "24h",
			fileModTime: time.Now().Add(-25 * time.Hour),
			want:        true,
		},
		{
			name:        "24h strategy with non-expired file",
			keepUpdated: "24h",
			fileModTime: time.Now().Add(-12 * time.Hour),
			want:        false,
		},
		{
			name:        "72h strategy with expired file",
			keepUpdated: "72h",
			fileModTime: time.Now().Add(-73 * time.Hour),
			want:        true,
		},
		{
			name:        "72h strategy with non-expired file",
			keepUpdated: "72h",
			fileModTime: time.Now().Add(-48 * time.Hour),
			want:        false,
		},
		{
			name:        "1d strategy with expired file",
			keepUpdated: "1d",
			fileModTime: time.Now().Add(-25 * time.Hour),
			want:        true,
		},
		{
			name:        "1d strategy with non-expired file",
			keepUpdated: "1d",
			fileModTime: time.Now().Add(-12 * time.Hour),
			want:        false,
		},
		{
			name:        "7d strategy with expired file",
			keepUpdated: "7d",
			fileModTime: time.Now().Add(-8 * 24 * time.Hour),
			want:        true,
		},
		{
			name:        "7d strategy with non-expired file",
			keepUpdated: "7d",
			fileModTime: time.Now().Add(-3 * 24 * time.Hour),
			want:        false,
		},
		{
			name:        "invalid hours format",
			keepUpdated: "abc",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "invalid hours value zero",
			keepUpdated: "0h",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "invalid hours value negative",
			keepUpdated: "-1h",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "invalid days value zero",
			keepUpdated: "0d",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "invalid days value negative",
			keepUpdated: "-1d",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "zero value time always needs update with hours",
			keepUpdated: "24h",
			fileModTime: time.Time{},
			want:        true,
		},
		{
			name:        "zero value time always needs update with days",
			keepUpdated: "7d",
			fileModTime: time.Time{},
			want:        true,
		},
		{
			name:        "case insensitive enable",
			keepUpdated: "ENABLE",
			fileModTime: time.Now(),
			want:        true,
		},
		{
			name:        "case insensitive disable",
			keepUpdated: "DISABLE",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "case insensitive hours",
			keepUpdated: "24H",
			fileModTime: time.Now().Add(-25 * time.Hour),
			want:        true,
		},
		{
			name:        "case insensitive days",
			keepUpdated: "7D",
			fileModTime: time.Now().Add(-8 * 24 * time.Hour),
			want:        true,
		},
		{
			name:        "case insensitive yes",
			keepUpdated: "YES",
			fileModTime: time.Now(),
			want:        true,
		},
		{
			name:        "case insensitive no",
			keepUpdated: "NO",
			fileModTime: time.Now(),
			want:        false,
		},
		{
			name:        "case insensitive true",
			keepUpdated: "TRUE",
			fileModTime: time.Now(),
			want:        true,
		},
		{
			name:        "case insensitive false",
			keepUpdated: "FALSE",
			fileModTime: time.Now(),
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseKeepUpdated(tt.keepUpdated, tt.fileModTime)
			if got != tt.want {
				t.Errorf("ParseKeepUpdated(%q, %v) = %v, want %v", tt.keepUpdated, tt.fileModTime, got, tt.want)
			}
		})
	}
}

// TestGetFileModTime 测试获取文件修改时间
func TestGetFileModTime(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_file_mod_time")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 测试获取已存在文件的修改时间
	modTime := GetFileModTime(testFile)
	if modTime.IsZero() {
		t.Error("GetFileModTime should return non-zero time for existing file")
	}

	// 测试获取不存在文件的修改时间
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")
	modTime = GetFileModTime(nonExistentFile)
	if !modTime.IsZero() {
		t.Error("GetFileModTime should return zero time for non-existing file")
	}
}

// TestParseKeepUpdatedWithRealFile 测试使用真实文件的更新策略
func TestParseKeepUpdatedWithRealFile(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_keep_updated_real")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 获取文件修改时间
	modTime := GetFileModTime(testFile)

	// 测试 disable 策略
	if ParseKeepUpdated("disable", modTime) {
		t.Error("disable strategy should return false")
	}

	// 测试 enable 策略
	if !ParseKeepUpdated("enable", modTime) {
		t.Error("enable strategy should return true")
	}

	// 测试 1h 策略(文件刚创建，不应更新)
	if ParseKeepUpdated("1h", modTime) {
		t.Error("1h strategy should return false for newly created file")
	}

	// 测试 1d 策略(文件刚创建，不应更新)
	if ParseKeepUpdated("1d", modTime) {
		t.Error("1d strategy should return false for newly created file")
	}
}
