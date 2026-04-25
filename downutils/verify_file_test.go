package downutils

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseChecksum 测试校验值解析
func TestParseChecksum(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   ChecksumType
		wantValue  string
		wantEnable bool
	}{
		{"empty string", "", "", "", false},
		{"md5 prefix", "md5:abc123", ChecksumMD5, "abc123", true},
		{"sha1 prefix", "sha1:def456", ChecksumSHA1, "def456", true},
		{"sha256 prefix", "sha256:ghi789", ChecksumSHA256, "ghi789", true},
		{"no prefix defaults to md5", "jkl012", ChecksumMD5, "jkl012", true},
		{"unknown type treated as md5", "crc32:mno345", ChecksumMD5, "mno345", true},
		{"case insensitive", "MD5:ABC123", ChecksumMD5, "ABC123", true},
		{"empty value disabled", "md5:", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseChecksum(tt.input)
			if got.Enable != tt.wantEnable {
				t.Errorf("parseChecksum(%q).Enable = %v, want %v", tt.input, got.Enable, tt.wantEnable)
			}
			if got.Enable && got.Type != tt.wantType {
				t.Errorf("parseChecksum(%q).Type = %v, want %v", tt.input, got.Type, tt.wantType)
			}
			if got.Enable && got.Value != tt.wantValue {
				t.Errorf("parseChecksum(%q).Value = %v, want %v", tt.input, got.Value, tt.wantValue)
			}
		})
	}
}

// TestParseCheckSize 测试文件大小解析
func TestParseCheckSize(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSize   int64
		wantEnable bool
	}{
		{"empty string", "", 0, false},
		{"1KB", "1KB", 1024, true},
		{"1MB", "1MB", 1024 * 1024, true},
		{"1GB", "1GB", 1024 * 1024 * 1024, true},
		{"512B", "512B", 512, true},
		{"1.5MB", "1.5MB", 1572864, true},
		{"invalid format", "abc", 0, false},
		{"negative value", "-1KB", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCheckSize(tt.input)
			if got.Enable != tt.wantEnable {
				t.Errorf("parseCheckSize(%q).Enable = %v, want %v", tt.input, got.Enable, tt.wantEnable)
			}
			if got.Enable && got.Size != tt.wantSize {
				t.Errorf("parseCheckSize(%q).Size = %v, want %v", tt.input, got.Size, tt.wantSize)
			}
		})
	}
}

// TestCalculateFileChecksum 测试文件校验值计算
func TestCalculateFileChecksum(t *testing.T) {
	// 创建临时测试文件
	tmpDir, err := os.MkdirTemp("", "test_checksum")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "test content for checksum"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 测试MD5
	md5Sum, err := calcFileChecksum(testFile, ChecksumMD5)
	if err != nil {
		t.Fatalf("calcFileChecksum MD5 failed: %v", err)
	}
	if len(md5Sum) != 32 {
		t.Errorf("MD5 checksum length = %v, want 32", len(md5Sum))
	}

	// 测试SHA1
	sha1Sum, err := calcFileChecksum(testFile, ChecksumSHA1)
	if err != nil {
		t.Fatalf("calcFileChecksum SHA1 failed: %v", err)
	}
	if len(sha1Sum) != 40 {
		t.Errorf("SHA1 checksum length = %v, want 40", len(sha1Sum))
	}

	// 测试SHA256
	sha256Sum, err := calcFileChecksum(testFile, ChecksumSHA256)
	if err != nil {
		t.Fatalf("calcFileChecksum SHA256 failed: %v", err)
	}
	if len(sha256Sum) != 64 {
		t.Errorf("SHA256 checksum length = %v, want 64", len(sha256Sum))
	}
}

// TestVerifyFileChecksum 测试文件校验值验证
func TestVerifyFileChecksum(t *testing.T) {
	// 创建临时测试文件
	tmpDir, err := os.MkdirTemp("", "test_verify_checksum")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "test content for verification"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 计算正确的MD5
	correctMD5, _ := calcFileChecksum(testFile, ChecksumMD5)

	// 测试正确校验值
	parsed := ParsedChecksum{Type: ChecksumMD5, Value: correctMD5, Enable: true}
	match, _, err := verifyFileChecksum(testFile, parsed)
	if err != nil {
		t.Fatalf("verifyFileChecksum failed: %v", err)
	}
	if !match {
		t.Error("verifyFileChecksum should match with correct checksum")
	}

	// 测试错误校验值
	parsedWrong := ParsedChecksum{Type: ChecksumMD5, Value: "wrongchecksum", Enable: true}
	match, _, err = verifyFileChecksum(testFile, parsedWrong)
	if err != nil {
		t.Fatalf("verifyFileChecksum failed: %v", err)
	}
	if match {
		t.Error("verifyFileChecksum should not match with wrong checksum")
	}

	// 测试禁用校验
	parsedDisabled := ParsedChecksum{Enable: false}
	match, _, err = verifyFileChecksum(testFile, parsedDisabled)
	if err != nil {
		t.Fatalf("verifyFileChecksum failed: %v", err)
	}
	if !match {
		t.Error("verifyFileChecksum should return true when disabled")
	}
}

// TestVerifyFileSize 测试文件大小验证
func TestVerifyFileSize(t *testing.T) {
	// 创建临时测试文件
	tmpDir, err := os.MkdirTemp("", "test_verify_size")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "1234567890" // 10 bytes
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 测试正确大小
	parsed := ParsedCheckSize{Size: 10, Enable: true}
	match, _, err := verifyFileSize(testFile, parsed)
	if err != nil {
		t.Fatalf("verifyFileSize failed: %v", err)
	}
	if !match {
		t.Error("verifyFileSize should match with correct size")
	}

	// 测试错误大小
	parsedWrong := ParsedCheckSize{Size: 100, Enable: true}
	match, _, err = verifyFileSize(testFile, parsedWrong)
	if err != nil {
		t.Fatalf("verifyFileSize failed: %v", err)
	}
	if match {
		t.Error("verifyFileSize should not match with wrong size")
	}

	// 测试禁用校验
	parsedDisabled := ParsedCheckSize{Enable: false}
	match, _, err = verifyFileSize(testFile, parsedDisabled)
	if err != nil {
		t.Fatalf("verifyFileSize failed: %v", err)
	}
	if !match {
		t.Error("verifyFileSize should return true when disabled")
	}
}
