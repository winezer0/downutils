package downutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// fileExists 检查文件是否存在
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// convertGitHubURL 转换GitHub URL为原始内容URL
func convertGitHubURL(url string) string {
	// 只转换blob URL，不转换releases下载链接
	if strings.Contains(url, "/releases/") {
		return url
	}
	return strings.Replace(strings.Replace(url, "github.com", "raw.githubusercontent.com", 1), "/blob/", "/", 1)
}

// formatDuration 格式化时间为易读格式
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	if d.Hours() >= 1 {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	} else if d.Minutes() >= 1 {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// formatSize 格式化大小为易读格式
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// makeDirs 创建目录
func makeDirs(path string, isFile bool) error {
	dir := path
	if isFile {
		dir = filepath.Dir(path)
	}
	_, err := os.Stat(dir)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, os.ModePerm)
	}
	return err
}

// getItemFilePath 获取文件存储路径
func getItemFilePath(filename, downloadDir string) string {
	storePath := filename
	if !filepath.IsAbs(filename) {
		storePath = filepath.Join(downloadDir, filename)
	}
	return storePath
}

// filterEnableItems 仅保留 enable=true 的配置项
func filterEnableItems(items []DownItem) []DownItem {
	var enabledItems []DownItem
	if len(items) > 0 {
		for _, item := range items {
			if item.Enable {
				enabledItems = append(enabledItems, item)
			}
		}
	}
	return enabledItems
}

// FindFilesBySuffix 递归查找指定目录下所有以后缀 suffix 结尾的文件
func FindFilesBySuffix(root, suffix string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), suffix) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
