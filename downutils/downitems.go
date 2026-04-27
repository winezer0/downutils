package downutils

import (
	"context"
	"errors"
	"fmt"
	"github.com/winezer0/xutils/logging"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// CheckDuplicatePaths 检查配置中是否存在重复的保存路径
func CheckDuplicatePaths(config DownConfig, outputForce string) {
	pathMap := make(map[string]string) // path -> module name

	for groupName, items := range config {
		for _, item := range items {
			storePath := GetDownItemFinalPath(item.FileName, item.StorageDir, outputForce)

			// 检查是否已存在相同路径
			if existingModule, exists := pathMap[storePath]; exists {
				logging.Warnf("duplicate save path detected: %s (used by both '%s' and '%s' in group '%s')",
					storePath, existingModule, item.Module, groupName)
			} else {
				pathMap[storePath] = item.Module
			}
		}
	}
}

// ProcessDownItems 处理配置组，返回每个任务的详细信息
func ProcessDownItems(client *http.Client, items []DownItem, downOptions *DownOptions) []TaskInfo {
	if downOptions == nil {
		downOptions = NewDefaultDownOptions()
	}

	// 预分配切片容量，避免多次扩容
	taskInfos := make([]TaskInfo, 0, len(items))

	// 如果并发数为1，使用串行下载
	if downOptions.MaxConcurrent <= 1 {
		for _, item := range items {
			taskInfo := processSingleItem(client, item, downOptions)
			taskInfos = append(taskInfos, taskInfo)
		}
		return taskInfos
	}

	// 并发下载
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, downOptions.MaxConcurrent)
		results = make([]TaskInfo, len(items))
	)

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(idx int, it DownItem) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			taskInfo := processSingleItem(client, it, downOptions)
			mu.Lock()
			results[idx] = taskInfo
			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

	// 按原始顺序收集结果
	for _, taskInfo := range results {
		taskInfos = append(taskInfos, taskInfo)
	}

	return taskInfos
}

// processSingleItem 处理单个下载项
func processSingleItem(client *http.Client, item DownItem, downOptions *DownOptions) TaskInfo {
	// 确定最终存储目录，优先级：OutputForce > StorageDir
	storePath := GetDownItemFinalPath(item.FileName, item.StorageDir, downOptions.OutputForce)

	// 检查文件是否存在
	fileExists := fileExists(storePath)

	// 如果强制更新，则忽略更新策略，直接下载
	if downOptions.UpdateForce {
		logging.Infof("force update enabled for %s", item.FileName)
	} else {
		// 获取文件修改时间
		fileModTime := GetFileModTime(storePath)

		// 根据更新策略判断是否需要更新
		needsUpdate := ParseKeepUpdated(item.KeepUpdated, fileModTime)

		// 如果文件已存在且不需要更新，则跳过
		if fileExists && !needsUpdate {
			logging.Infof("file %s already exists and does not need update, skipping", item.FileName)
			return TaskInfo{
				Module:      item.Module,
				FileName:    item.FileName,
				StoragePath: storePath,
				Status:      TaskStatusSkipped,
			}
		}
	}

	// 创建目录
	if err := makeDirs(storePath, true); err != nil {
		logging.Errorf("directory [%s] initialization failed: %v", item.FileName, err)
		return TaskInfo{
			Module:      item.Module,
			FileName:    item.FileName,
			StoragePath: storePath,
			Status:      TaskStatusConfigErr,
			ErrorMsg:    fmt.Sprintf("directory initialization failed: %v", err),
		}
	}

	logging.Infof("starting download %s...", item.Module)

	// 初始化任务信息
	taskInfo := TaskInfo{
		Module:      item.Module,
		FileName:    item.FileName,
		StoragePath: storePath,
		Status:      TaskStatusFailed,
	}

	success := false
	allUrlsNotFound := true
	hasTriedUrls := false

	// 尝试从每个URL下载
	for _, url := range item.DownloadURLs {
		// 处理GitHub URL
		downloadURL := url
		if strings.Contains(url, "github.com") && strings.Contains(url, "/blob/") {
			downloadURL = convertGitHubURL(url)
			logging.Debugf("converted GitHub URL: %s -> %s", url, downloadURL)
		}

		hasTriedUrls = true
		urlFailed404 := false

		// 尝试下载，支持重试
	retryLoop:
		for attempt := 1; attempt <= downOptions.MaxRetries; attempt++ {
			if attempt > 1 {
				logging.Infof("retry %d downloading...", attempt)
			} else {
				logging.Infof("downloading %s...", downloadURL)
			}

			// 创建带超时的context
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(downOptions.IdleTimeout)*time.Second)

			// 执行下载
			taskConfig := &DownloadTaskConfig{
				Checksum:     item.Checksum,
				CheckSize:    item.CheckSize,
				ShowProgress: downOptions.ShowProgress,
				MaxSpeed:     downOptions.MaxSpeed,
			}
			if err := downloadFile(ctx, client, downloadURL, storePath, taskConfig); err != nil {
				cancel()

				// 对错误进行分类
				var downloadErr DownloadError
				logging.Errorf("download failed: %v", err)

				if errors.As(err, &downloadErr) {
					switch downloadErr.Type {
					case ErrResourceNotFound:
						logging.Warnf("resource not found (404) for URL: %s", downloadURL)
						urlFailed404 = true
						taskInfo.ErrorMsg = fmt.Sprintf("resource not found (404): %s", downloadURL)
						break retryLoop

					case ErrDownloadCancelled:
						logging.Warnf("download cancelled: %v", err)
						taskInfo.ErrorMsg = fmt.Sprintf("download cancelled: %v", err)
						break retryLoop

					case ErrTimeout:
						logging.Warnf("timeout error for URL: %s", downloadURL)
						taskInfo.ErrorMsg = fmt.Sprintf("timeout error: %v", err)

					case ErrNetworkError:
						logging.Warnf("network error for URL: %s", downloadURL)
						taskInfo.ErrorMsg = fmt.Sprintf("network error: %v", err)

					case ErrDiskFull:
						logging.Errorf("disk full, cannot download to %s", storePath)
						taskInfo.ErrorMsg = fmt.Sprintf("disk full: %v", err)
						allUrlsNotFound = false
						break retryLoop

					case ErrPermissionDenied:
						logging.Errorf("permission denied for path: %s", storePath)
						taskInfo.ErrorMsg = fmt.Sprintf("permission denied: %v", err)
						allUrlsNotFound = false
						break retryLoop

					case ErrChecksumMismatch:
						logging.Warnf("checksum/size verification failed for URL: %s", downloadURL)
						taskInfo.ErrorMsg = fmt.Sprintf("verification failed: %v", err)

					default:
						logging.Warnf("unknown error type: %s", downloadErr.Type)
						taskInfo.ErrorMsg = fmt.Sprintf("unknown error: %v", err)
					}
				} else {
					taskInfo.ErrorMsg = fmt.Sprintf("download failed: %v", err)
				}

				// 如果不是最后一次尝试，则等待后重试
				if attempt < downOptions.MaxRetries {
					waitTime := time.Duration(attempt) * 2 * time.Second
					logging.Infof("waiting %v before retry...", waitTime)
					select {
					case <-time.After(waitTime):
					case <-ctx.Done():
						logging.Warnf("retry wait cancelled: %v", ctx.Err())
						taskInfo.ErrorMsg = fmt.Sprintf("retry wait cancelled: %v", ctx.Err())
						allUrlsNotFound = false
						break retryLoop
					}
					continue
				}
				break
			} else {
				cancel()
				logging.Infof("successfully downloaded %s to %s", item.Module, storePath)
				taskInfo.Status = TaskStatusSuccess
				taskInfo.DownloadURL = downloadURL
				taskInfo.ErrorMsg = ""
				taskInfo.Resumed = checkFileExistsForResume(storePath, downloadURL)
				success = true
				allUrlsNotFound = false
				break retryLoop
			}
		}

		if success {
			break
		}
		if urlFailed404 {
			continue
		}
		allUrlsNotFound = false
	}

	if !success {
		if allUrlsNotFound && hasTriedUrls {
			logging.Warnf("all URLs returned 404 for %s, please check the URLs in the configuration file", item.Module)
			if taskInfo.ErrorMsg == "" {
				taskInfo.ErrorMsg = "all URLs returned 404"
			}
		} else {
			logging.Errorf("all download sources failed, unable to download %s", item.Module)
			if taskInfo.ErrorMsg == "" {
				taskInfo.ErrorMsg = "all download sources failed"
			}
		}
	}

	return taskInfo
}

// checkFileExistsForResume 检查文件是否存在(用于判断是否断点续传)
func checkFileExistsForResume(storePath string, downloadUrl string) bool {
	existingTempFile := findIncompleteTempFile(storePath, downloadUrl)
	if existingTempFile != "" {
		info, statErr := os.Stat(existingTempFile)
		if statErr == nil && info.Size() > 0 {
			return true
		}
	}
	return false
}
