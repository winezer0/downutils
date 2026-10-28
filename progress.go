package downutils

import (
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
)

// createProgressBar 创建字节进度条(使用progress包)
func createProgressBar(fileSize int64, fileName string) *progressbar.ProgressBar {
	// 使用progress包的字节进度条
	return newDownloadProgress(fileSize, fileName, time.Second,
		progressbar.OptionShowBytes(true), progressbar.OptionSetPredictTime(true))
}

// createSpinner 创建 spinner 动画进度条(用于文件大小未知的情况)
func createSpinner(fileName string) *progressbar.ProgressBar {
	return newDownloadProgress(-1, fileName, 100*time.Millisecond, progressbar.OptionShowCount())
}

func newDownloadProgress(total int64, fileName string, throttle time.Duration, extra ...progressbar.Option) *progressbar.ProgressBar {
	options := []progressbar.Option{
		progressbar.OptionSetDescription(fmt.Sprintf("downloading %s", fileName)),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(throttle),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer: "=", SaucerHead: ">", SaucerPadding: " ", BarStart: "[", BarEnd: "]",
		}),
	}
	return progressbar.NewOptions64(total, append(options, extra...)...)
}
