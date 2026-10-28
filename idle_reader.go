package downutils

import (
	"context"
	"io"
	"time"
)

// idleReadCloser limits only time spent waiting for response-body data, not throttling.
type idleReadCloser struct {
	io.ReadCloser
	timeout time.Duration
	cancel  context.CancelCauseFunc
}

func (r *idleReadCloser) Read(p []byte) (int, error) {
	done := make(chan struct{})
	timer := time.AfterFunc(r.timeout, func() {
		r.cancel(context.DeadlineExceeded)
		close(done)
	})
	n, err := r.ReadCloser.Read(p)
	if !timer.Stop() {
		<-done
		return n, context.DeadlineExceeded
	}
	return n, err
}
