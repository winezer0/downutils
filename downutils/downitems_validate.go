package downutils

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateDownConfig 验证整个下载配置
// 当 enableForce 为 true 时，验证所有项目；否则仅验证已启用的项目
// 返回验证错误信息列表，空列表表示验证通过
func ValidateDownConfig(config DownConfig, enableForce bool) []string {
	var allErrs []string

	for groupName, items := range config {
		errs := validateDownItem(items, enableForce)
		for _, err := range errs {
			allErrs = append(allErrs, fmt.Sprintf("[%s] %s", groupName, err))
		}
	}

	return allErrs
}

// validateDownItem 验证单个下载项配置
// 当 enableForce 为 true 时，验证所有项目；否则仅验证已启用的项目
// 返回验证错误信息列表，空列表表示验证通过
func validateDownItem(items []DownItem, enableForce bool) []string {
	var errs []string

	// Windows 文件名非法字符
	invalidCharsPattern := `[<>:"/\\|?*]`
	invalidCharsRegex := regexp.MustCompile(invalidCharsPattern)

	for _, item := range items {
		// 如果未启用且未强制启用，则跳过验证
		if !item.Enable && !enableForce {
			continue
		}

		// 验证 Module 字段
		if strings.TrimSpace(item.Module) == "" {
			errs = append(errs, fmt.Sprintf("module name is empty for item"))
		}

		// 验证 FileName 字段
		if strings.TrimSpace(item.FileName) == "" {
			errs = append(errs, fmt.Sprintf("file name is empty for module '%s'", item.Module))
		} else if invalidCharsRegex.MatchString(item.FileName) {
			errs = append(errs, fmt.Sprintf("file name '%s' contains invalid characters for module '%s'", item.FileName, item.Module))
		}

		// 验证 DownloadURLs 字段
		if len(item.DownloadURLs) == 0 {
			errs = append(errs, fmt.Sprintf("download URLs is empty for module '%s'", item.Module))
		} else {
			for i, url := range item.DownloadURLs {
				if strings.TrimSpace(url) == "" {
					errs = append(errs, fmt.Sprintf("download URL at index %d is empty for module '%s'", i, item.Module))
				}
			}
		}
	}

	return errs
}
