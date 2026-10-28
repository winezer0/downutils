package downutils

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func assertDiskFullClassification(t *testing.T, cause error, want bool) {
	t.Helper()
	for _, err := range []error{cause, &os.PathError{Op: "write", Path: "target", Err: cause}, fmt.Errorf("save failed: %w", &os.PathError{Op: "write", Path: "target", Err: cause})} {
		var got DownloadError
		classified := classifyError(err, 0)
		isFull := errors.As(classified, &got) && got.Type == ErrDiskFull
		if isFull != want {
			t.Errorf("classifyError(%v) = %v, disk-full=%v, want %v", err, classified, isFull, want)
		}
	}
}
