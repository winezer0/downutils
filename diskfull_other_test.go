//go:build !windows

package downutils

import (
	"syscall"
	"testing"
)

func TestPlatformDiskFullClassification(t *testing.T) {
	for _, err := range []error{syscall.ENOSPC, syscall.EDQUOT} {
		assertDiskFullClassification(t, err, true)
	}
	// These numeric values were previously mistaken for Windows disk-full errors.
	for _, err := range []error{syscall.Errno(39), syscall.Errno(112), syscall.ENOENT} {
		assertDiskFullClassification(t, err, false)
	}
}
