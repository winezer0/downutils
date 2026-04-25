package downutils

import (
	"context"
	"io"
)

// CountingWriter 是一个包装io.Writer的结构，用于跟踪写入的字节数
type CountingWriter struct {
	Writer io.Writer
}

// Write 实现io.Writer接口
func (w *CountingWriter) Write(p []byte) (n int, err error) {
	return w.Writer.Write(p)
}

// copyWithContext 带context支持的复制函数
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf == nil {
		buf = make([]byte, 32*1024)
	}

	for {
		// 检查context是否已取消
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				return
			}
			if nr != nw {
				err = io.ErrShortWrite
				return
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			return
		}
	}
}
