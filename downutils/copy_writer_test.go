package downutils

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

// TestCopyWithContext 测试带context的复制功能
func TestCopyWithContext(t *testing.T) {
	// 创建测试数据
	testData := []byte("test data for copy with context")
	src := bytes.NewReader(testData)
	dst := &bytes.Buffer{}
	buf := make([]byte, 32*1024)

	// 创建context
	ctx := context.Background()

	// 测试复制
	written, err := copyWithContext(ctx, dst, src, buf)
	if err != nil {
		t.Fatalf("copyWithContext failed: %v", err)
	}

	if written != int64(len(testData)) {
		t.Errorf("expected to write %d bytes, wrote %d", len(testData), written)
	}

	if dst.String() != string(testData) {
		t.Errorf("copied data mismatch: got %s, want %s", dst.String(), string(testData))
	}
}

// TestCopyWithContextCancel 测试context取消
func TestCopyWithContextCancel(t *testing.T) {
	// 创建慢速reader
	slowReader := &slowReader{
		data:     make([]byte, 1000000),
		delay:    100 * time.Millisecond,
		position: 0,
	}

	dst := &bytes.Buffer{}
	buf := make([]byte, 1024)

	// 创建短超时context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 测试复制应该被取消
	_, err := copyWithContext(ctx, dst, slowReader, buf)
	if err == nil {
		t.Error("copyWithContext should fail when context is cancelled")
	}

	if ctx.Err() == nil {
		t.Error("context should be cancelled")
	}
}

// TestCopyWithNilBuffer 测试nil buffer
func TestCopyWithNilBuffer(t *testing.T) {
	// 创建测试数据
	testData := []byte("test data with nil buffer")
	src := bytes.NewReader(testData)
	dst := &bytes.Buffer{}

	// 创建context
	ctx := context.Background()

	// 测试复制(buffer为nil)
	written, err := copyWithContext(ctx, dst, src, nil)
	if err != nil {
		t.Fatalf("copyWithContext with nil buffer failed: %v", err)
	}

	if written != int64(len(testData)) {
		t.Errorf("expected to write %d bytes, wrote %d", len(testData), written)
	}

	if dst.String() != string(testData) {
		t.Errorf("copied data mismatch: got %s, want %s", dst.String(), string(testData))
	}
}

// TestCopyWithLargeData 测试大数据复制
func TestCopyWithLargeData(t *testing.T) {
	// 创建大数据(1MB)
	testData := make([]byte, 1024*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	src := bytes.NewReader(testData)
	dst := &bytes.Buffer{}
	buf := make([]byte, 32*1024)

	// 创建context
	ctx := context.Background()

	// 测试复制
	written, err := copyWithContext(ctx, dst, src, buf)
	if err != nil {
		t.Fatalf("copyWithContext with large data failed: %v", err)
	}

	if written != int64(len(testData)) {
		t.Errorf("expected to write %d bytes, wrote %d", len(testData), written)
	}

	if !bytes.Equal(dst.Bytes(), testData) {
		t.Error("copied large data mismatch")
	}
}

// TestCopyWithEOF 测试EOF处理
func TestCopyWithEOF(t *testing.T) {
	// 创建空reader
	src := bytes.NewReader([]byte{})
	dst := &bytes.Buffer{}
	buf := make([]byte, 1024)

	// 创建context
	ctx := context.Background()

	// 测试复制空数据
	written, err := copyWithContext(ctx, dst, src, buf)
	if err != nil {
		t.Fatalf("copyWithContext with empty data failed: %v", err)
	}

	if written != 0 {
		t.Errorf("expected to write 0 bytes, wrote %d", written)
	}

	if dst.Len() != 0 {
		t.Errorf("expected empty buffer, got %d bytes", dst.Len())
	}
}

// slowReader 慢速reader，用于测试context取消
type slowReader struct {
	data     []byte
	delay    time.Duration
	position int
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	if sr.position >= len(sr.data) {
		return 0, io.EOF
	}

	// 模拟延迟
	time.Sleep(sr.delay)

	n = copy(p, sr.data[sr.position:])
	sr.position += n
	return n, nil
}
