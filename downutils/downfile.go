package downutils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/winezer0/xutils/logging"
)

// classifyError 对错误进行分类，返回 DownloadError
func classifyError(err error, statusCode int) error {
	// 检查是否是 HTTP 状态码错误
	if statusCode >= 400 {
		if statusCode == http.StatusNotFound {
			return DownloadError{
				StatusCode: statusCode,
				Message:    fmt.Sprintf("HTTP %d: resource not found", statusCode),
				Type:       ErrResourceNotFound,
			}
		}
		return DownloadError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("HTTP %d error", statusCode),
			Type:       "HTTP_ERROR",
		}
	}

	// 检查是否是超时错误
	if os.IsTimeout(err) {
		return DownloadError{
			Message: fmt.Sprintf("timeout: %v", err),
			Type:    ErrTimeout,
		}
	}

	// 检查是否是权限错误
	if os.IsPermission(err) {
		return DownloadError{
			Message: fmt.Sprintf("permission denied: %v", err),
			Type:    ErrPermissionDenied,
		}
	}

	// 检查是否是磁盘空间不足(Windows 错误码 112)
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			// Windows ERROR_DISK_FULL = 112, ERROR_HANDLE_DISK_FULL = 39
			if errno == 112 || errno == 39 {
				return DownloadError{
					Message: fmt.Sprintf("disk full: %v", err),
					Type:    ErrDiskFull,
				}
			}
		}
	}

	// 检查错误信息中是否包含网络相关关键字
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "reset") ||
		strings.Contains(errMsg, "refused") {
		return DownloadError{
			Message: fmt.Sprintf("network error: %v", err),
			Type:    ErrNetworkError,
		}
	}

	// 未知错误，包装为通用错误
	return fmt.Errorf("download error: %w", err)
}

// downloadFile 下载文件，支持context取消、进度条显示和断点续传
func downloadFile(ctx context.Context, client *http.Client, downloadUrl, storePath string, taskConfig *DownloadTaskConfig) error {
	// 创建目标文件的目录(如果不存在)
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 生成精确的临时文件名(基于URL哈希，避免不同任务冲突)
	tempFile := generateTempFileName(storePath, downloadUrl)

	// 检查是否存在未完成的临时文件(支持断点续传)
	var downloadedBytes int64 = 0
	var outFile *os.File
	var err error

	existingTempFile := findIncompleteTempFile(storePath, downloadUrl)
	if existingTempFile != "" {
		info, statErr := os.Stat(existingTempFile)
		if statErr == nil && info.Size() > 0 {
			logging.Infof("found incomplete file %s, resuming download from %d bytes", existingTempFile, info.Size())
			downloadedBytes = info.Size()
			outFile, err = os.OpenFile(existingTempFile, os.O_APPEND|os.O_WRONLY, 0644)
			tempFile = existingTempFile
		}
	}

	// 如果没有找到可续传的临时文件，创建新的
	if outFile == nil {
		outFile, err = os.Create(tempFile)
		if err != nil {
			return DownloadError{
				Message: fmt.Sprintf("failed to create temp file: %v", err),
				Type:    ErrCreateTempFile,
			}
		}
	}

	// 使用defer确保在函数退出时处理临时文件
	var downloadSuccess bool
	var fileClosed bool
	defer func() {
		if !fileClosed {
			if err := outFile.Close(); err != nil {
				logging.Warnf("failed to close file: %v", err)
			}
			fileClosed = true
		}
		if !downloadSuccess {
			// 下载失败，保留临时文件以便后续续传
			logging.Infof("download failed, keeping temp file %s for resume", tempFile)
		}
	}()

	// 尝试下载，支持断点续传失败后降级为完整下载
	resp, actualDownloadedBytes, err := downloadWithResumeFallback(ctx, client, downloadUrl, downloadedBytes)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 如果发生了降级(从断点续传降级为完整下载)，需要重置文件
	if actualDownloadedBytes != downloadedBytes {
		logging.Infof("resume failed, falling back to full download")
		// 关闭当前文件并重新创建
		if err := outFile.Close(); err != nil {
			logging.Warnf("failed to close file before fallback: %v", err)
		}
		outFile, err = os.Create(tempFile)
		if err != nil {
			return fmt.Errorf("failed to recreate temp file for full download: %w", err)
		}
		fileClosed = false
		downloadedBytes = 0

		// 重新发送完整下载请求
		resp, err = HttpGetWithRange(ctx, client, downloadUrl, 0)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	// 获取文件大小
	var fileSize int64
	if resp.StatusCode == http.StatusPartialContent {
		// 断点续传响应，解析 Content-Range 头
		contentRange := resp.Header.Get("Content-Range")
		if contentRange != "" {
			// 格式：bytes start-end/total
			var total int64
			_, err := fmt.Sscanf(contentRange, "bytes %d-%d/%d", nil, nil, &total)
			if err == nil {
				fileSize = total
			}
		}
	} else {
		fileSize = resp.ContentLength
	}
	fileName := filepath.Base(storePath)

	// 根据showProgress选项和文件大小决定使用何种进度显示
	var writer io.Writer = outFile
	var barWrapper *progressbar.ProgressBar
	if taskConfig.ShowProgress {
		if fileSize > 0 {
			// 文件大小已知，使用字节进度条
			barWrapper = createProgressBar(fileSize, fileName)
			writer = io.MultiWriter(outFile, barWrapper)
		} else {
			// 文件大小未知，使用 spinner 动画显示进度
			barWrapper = createSpinner(fileName)
			writer = io.MultiWriter(outFile, barWrapper)
		}
	}

	// 应用下载速度限制
	var reader io.Reader = resp.Body
	if taskConfig.MaxSpeed > 0 {
		reader = NewLimitReader(resp.Body, taskConfig.MaxSpeed)
	}

	// 复制内容，支持context取消
	buf := make([]byte, DownloadBufferSize)
	_, err = copyWithContext(ctx, writer, reader, buf)

	// 检查是否是因为context取消导致的错误
	if ctx.Err() != nil {
		return DownloadError{
			Message: fmt.Sprintf("download cancelled: %v", ctx.Err()),
			Type:    ErrDownloadCancelled,
		}
	}

	// 检查其他错误
	if err != nil {
		return classifyError(err, resp.StatusCode)
	}

	// 关闭进度条(触发换行)
	if barWrapper != nil {
		barWrapper.Finish()
	}

	// 关闭文件，确保内容写入磁盘
	if err := outFile.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	fileClosed = true

	// 标记下载成功，避免在defer中删除临时文件
	downloadSuccess = true

	// 删除旧文件(如果存在)
	if fileExists(storePath) {
		if err := os.Remove(storePath); err != nil {
			return fmt.Errorf("error: failed to delete old file: %w", err)
		}
	}

	// 重命名临时文件为最终文件名
	if err := os.Rename(tempFile, storePath); err != nil {
		return fmt.Errorf("error: failed to rename temp file: %w", err)
	}

	// 验证文件大小(如果配置了)
	parsedSize := parseCheckSize(taskConfig.CheckSize)
	if parsedSize.Enable {
		match, actualSize, verifyErr := verifyFileSize(storePath, parsedSize)
		if verifyErr != nil {
			return fmt.Errorf("file size verification failed: %w", verifyErr)
		}
		if !match {
			return DownloadError{
				Message: fmt.Sprintf("file size mismatch: expected %d bytes, got %d bytes", parsedSize.Size, actualSize),
				Type:    ErrChecksumMismatch,
			}
		}
		logging.Infof("file size verification passed: %d bytes", actualSize)
	}

	// 验证文件校验值(如果配置了)
	parsedChecksum := parseChecksum(taskConfig.Checksum)
	if parsedChecksum.Enable {
		match, actual, verifyErr := verifyFileChecksum(storePath, parsedChecksum)
		if verifyErr != nil {
			return fmt.Errorf("checksum verification failed: %w", verifyErr)
		}
		if !match {
			return DownloadError{
				Message: fmt.Sprintf("checksum mismatch: expected %s:%s, got %s:%s",
					parsedChecksum.Type, parsedChecksum.Value, parsedChecksum.Type, actual),
				Type: ErrChecksumMismatch,
			}
		}
		logging.Infof("file checksum verification passed: %s:%s", parsedChecksum.Type, actual)
	}

	return nil
}

// generateTempFileName 生成唯一的临时文件名(基于URL哈希，避免不同任务冲突)
func generateTempFileName(storePath, downloadUrl string) string {
	hash := sha256.Sum256([]byte(downloadUrl))
	hashStr := hex.EncodeToString(hash[:8]) // 使用前16位哈希
	return storePath + "." + hashStr + ".download"
}

// findIncompleteTempFile 查找未完成的临时文件(基于URL哈希精确匹配)
func findIncompleteTempFile(storePath, downloadUrl string) string {
	expectedTempFile := generateTempFileName(storePath, downloadUrl)
	if fileExists(expectedTempFile) {
		return expectedTempFile
	}
	return ""
}

// downloadWithResumeFallback 尝试断点续传，失败后自动降级为完整下载
// 返回响应、实际下载的起始字节数、错误
func downloadWithResumeFallback(ctx context.Context, client *http.Client, downloadUrl string, resumeFrom int64) (*http.Response, int64, error) {
	if resumeFrom == 0 {
		// 无需续传，直接下载
		resp, err := HttpGetWithRange(ctx, client, downloadUrl, 0)
		return resp, 0, err
	}

	// 先发送一个轻量请求检查服务器是否支持 Range 请求
	checkResp, err := HttpHeadRequest(ctx, client, downloadUrl)
	if err != nil {
		logging.Warnf("failed to check server range support, falling back to full download: %v", err)
		return nil, 0, nil // 返回nil响应，由调用方处理
	}

	// 检查 Accept-Ranges 头
	acceptRanges := checkResp.Header.Get("Accept-Ranges")
	if acceptRanges != "bytes" {
		logging.Warnf("server does not support range requests (Accept-Ranges: %s), falling back to full download", acceptRanges)
		checkResp.Body.Close()
		return nil, 0, nil
	}
	checkResp.Body.Close()

	// 服务器支持 Range 请求，尝试断点续传
	resp, err := HttpGetWithRange(ctx, client, downloadUrl, resumeFrom)
	if err != nil {
		logging.Warnf("resume request failed, falling back to full download: %v", err)
		return nil, 0, nil
	}

	// 检查响应状态
	if resp.StatusCode == http.StatusPartialContent {
		// 续传成功
		return resp, resumeFrom, nil
	}

	if resp.StatusCode == http.StatusOK {
		// 服务器返回完整文件，说明续传失败(可能文件已变更)
		logging.Warnf("server returned 200 OK instead of 206, file may have changed, falling back to full download")
		return resp, 0, nil
	}

	// 其他错误状态码
	logging.Warnf("resume request returned status %d, falling back to full download", resp.StatusCode)
	return resp, 0, nil
}
