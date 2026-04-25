package downutils

import (
	"fmt"
	"github.com/winezer0/xutils/progress"

	"github.com/schollz/progressbar/v3"
)

// createProgressBar 创建字节进度条(使用progress包)
func createProgressBar(fileSize int64, fileName string) *progressbar.ProgressBar {
	// 使用progress包的字节进度条
	return progress.NewByteProgressBar(fileSize, fmt.Sprintf("downloading %s", fileName))
}

// createSpinner 创建 spinner 动画进度条(用于文件大小未知的情况)
func createSpinner(fileName string) *progressbar.ProgressBar {
	return progress.NewSpinner(fmt.Sprintf("downloading %s", fileName))
}
