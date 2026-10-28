//go:build windows

package downutils

import (
	"golang.org/x/sys/windows"
	"testing"
)

func TestPlatformDiskFullClassification(t *testing.T) {
	for _, err := range []error{windows.ERROR_DISK_FULL, windows.ERROR_HANDLE_DISK_FULL} {
		assertDiskFullClassification(t, err, true)
	}
	assertDiskFullClassification(t, windows.ERROR_FILE_NOT_FOUND, false)
}
