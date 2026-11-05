package downutils

import (
	"io"
	"time"
)

const progressCallbackInterval = 100 * time.Millisecond

type downloadProgressWriter struct {
	writer       io.Writer
	callback     func(Progress)
	progress     Progress
	lastReported time.Time
}

func newDownloadProgressWriter(writer io.Writer, progress Progress, callback func(Progress)) *downloadProgressWriter {
	w := &downloadProgressWriter{
		writer:   writer,
		callback: callback,
		progress: progress,
	}
	w.report()
	return w
}

func (w *downloadProgressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.progress.Downloaded += int64(n)
	if err != nil || time.Since(w.lastReported) >= progressCallbackInterval {
		w.report()
	}
	return n, err
}

func (w *downloadProgressWriter) report() {
	w.lastReported = time.Now()
	w.callback(w.progress)
}
