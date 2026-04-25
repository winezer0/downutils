package downutils

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/winezer0/xutils/logging"
	"io"
	"os"
	"strconv"
	"strings"
)

// ChecksumType 校验类型
type ChecksumType string

const (
	ChecksumMD5    ChecksumType = "md5"
	ChecksumSHA1   ChecksumType = "sha1"
	ChecksumSHA256 ChecksumType = "sha256"
)

// ParsedChecksum 解析后的校验值
type ParsedChecksum struct {
	Type   ChecksumType
	Value  string
	Enable bool
}

// ParsedCheckSize 解析后的文件大小
type ParsedCheckSize struct {
	Size   int64
	Enable bool
}

// parseChecksum 解析校验值字符串
// 格式：md5:xxx/sha1:xxx/sha256:xxx，前缀为空时默认使用md5算法
// 返回解析后的校验类型和值，以及是否启用校验
func parseChecksum(checksum string) ParsedChecksum {
	if strings.TrimSpace(checksum) == "" {
		return ParsedChecksum{Enable: false}
	}

	// 检查是否包含前缀
	parts := strings.SplitN(checksum, ":", 2)
	if len(parts) == 2 {
		// 包含前缀
		checkType := ChecksumType(strings.ToLower(strings.TrimSpace(parts[0])))
		value := strings.TrimSpace(parts[1])

		// 验证校验类型
		switch checkType {
		case ChecksumMD5, ChecksumSHA1, ChecksumSHA256:
			if value == "" {
				logging.Warnf("empty checksum value for type %s, verification disabled", checkType)
				return ParsedChecksum{Enable: false}
			}
			return ParsedChecksum{
				Type:   checkType,
				Value:  value,
				Enable: true,
			}
		default:
			logging.Warnf("unknown checksum type '%s', treating as md5", checkType)
			return ParsedChecksum{
				Type:   ChecksumMD5,
				Value:  value,
				Enable: value != "",
			}
		}
	}

	// 无前缀，默认使用md5
	value := strings.TrimSpace(parts[0])
	if value == "" {
		return ParsedChecksum{Enable: false}
	}
	return ParsedChecksum{
		Type:   ChecksumMD5,
		Value:  value,
		Enable: true,
	}
}

// parseCheckSize 解析文件大小字符串
// 格式：1KB/1MB/1GB，支持小数点
// 返回解析后的文件大小(字节)，以及是否启用校验
func parseCheckSize(checkSize string) ParsedCheckSize {
	if strings.TrimSpace(checkSize) == "" {
		return ParsedCheckSize{Enable: false}
	}

	str := strings.TrimSpace(checkSize)

	// 提取数字和单位
	var numStr string
	var unit string

	// 查找单位位置(找到第一个非数字非小数点的字符)
	for i, c := range str {
		if !(c >= '0' && c <= '9' || c == '.') {
			numStr = str[:i]
			unit = str[i:]
			break
		}
	}

	// 如果没有找到单位，整个字符串作为数字
	if numStr == "" {
		numStr = str
		unit = ""
	}

	if numStr == "" {
		logging.Warnf("invalid check-size format '%s', verification disabled", checkSize)
		return ParsedCheckSize{Enable: false}
	}

	// 解析数字
	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil || size <= 0 {
		logging.Warnf("invalid check-size value '%s', verification disabled", checkSize)
		return ParsedCheckSize{Enable: false}
	}

	// 转换单位
	unit = strings.ToUpper(strings.TrimSpace(unit))
	var bytes int64
	switch unit {
	case "KB", "K":
		bytes = int64(size * 1024)
	case "MB", "M":
		bytes = int64(size * 1024 * 1024)
	case "GB", "G":
		bytes = int64(size * 1024 * 1024 * 1024)
	case "B", "":
		bytes = int64(size)
	default:
		logging.Warnf("unknown check-size unit '%s', treating as bytes", unit)
		bytes = int64(size)
	}

	return ParsedCheckSize{
		Size:   bytes,
		Enable: bytes > 0,
	}
}

// calcFileChecksum 计算文件的校验值
func calcFileChecksum(filePath string, checkType ChecksumType) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	buf := make([]byte, DownloadBufferSize)

	// 计算校验值
	switch checkType {
	case ChecksumMD5:
		h := md5.New()
		if _, err := io.CopyBuffer(h, file, buf); err != nil {
			return "", fmt.Errorf("failed to calculate md5: %w", err)
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	case ChecksumSHA1:
		h := sha1.New()
		if _, err := io.CopyBuffer(h, file, buf); err != nil {
			return "", fmt.Errorf("failed to calculate sha1: %w", err)
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	case ChecksumSHA256:
		h := sha256.New()
		if _, err := io.CopyBuffer(h, file, buf); err != nil {
			return "", fmt.Errorf("failed to calculate sha256: %w", err)
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	default:
		return "", fmt.Errorf("unsupported checksum type: %s", checkType)
	}
}

// verifyFileChecksum 验证文件校验值
func verifyFileChecksum(filePath string, parsed ParsedChecksum) (bool, string, error) {
	if !parsed.Enable {
		return true, "", nil
	}

	// 计算文件校验值
	actual, err := calcFileChecksum(filePath, parsed.Type)
	if err != nil {
		return false, "", err
	}

	// 比较校验值(不区分大小写)
	match := strings.EqualFold(actual, parsed.Value)
	if !match {
		logging.Warnf("checksum mismatch for %s: expected %s:%s, got %s:%s",
			filePath, parsed.Type, parsed.Value, parsed.Type, actual)
	}

	return match, actual, nil
}

// verifyFileSize 验证文件大小
func verifyFileSize(filePath string, parsed ParsedCheckSize) (bool, int64, error) {
	if !parsed.Enable {
		return true, 0, nil
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get file info: %w", err)
	}

	actualSize := info.Size()
	match := actualSize == parsed.Size

	if !match {
		logging.Warnf("file size mismatch for %s: expected %d bytes, got %d bytes",
			filePath, parsed.Size, actualSize)
	}

	return match, actualSize, nil
}
