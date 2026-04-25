package downutils

import (
	"fmt"
	"github.com/winezer0/xutils/logging"
	"os"
)

// cleanupIncompleteDownloads 清理下载目录下未完成的下载文件(以 .download 结尾的文件)
func cleanupIncompleteDownloads(downloadDir string) error {
	// 检查下载目录是否存在
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		return nil // 目录不存在，直接返回
	}
	// 查找所有 .download 文件
	files, err := FindFilesBySuffix(downloadDir, ".download")
	if err != nil {
		return fmt.Errorf("failed to find incomplete download files: %w", err)
	}
	// 删除每个文件
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			logging.Warnf("failed to delete incomplete download file %s: %v", file, err)
		}
	}
	return nil
}

// cleanupAllIncompleteDownloads 清理所有配置目录中未完成的下载文件
func cleanupAllIncompleteDownloads(config DownConfig, outputForce string) {
	dirs := collectConfigDirs(config, outputForce)
	for _, dir := range dirs {
		if err := cleanupIncompleteDownloads(dir); err != nil {
			logging.Warnf("failed to cleanup incomplete downloads in %s: %v", dir, err)
		} else {
			logging.Infof("cleaned up incomplete downloads in %s", dir)
		}
	}
}

// collectConfigDirs 收集配置中涉及的所有目录
func collectConfigDirs(config DownConfig, outputForce string) []string {
	dirSet := make(map[string]bool)

	for _, items := range config {
		for _, item := range items {
			var outputDir string
			if outputForce != "" {
				outputDir = outputForce
			} else if item.StorageDir != "" {
				outputDir = item.StorageDir
			} else {
				continue
			}
			dirSet[outputDir] = true
		}
	}

	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	return dirs
}
