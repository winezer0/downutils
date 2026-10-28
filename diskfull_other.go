//go:build !windows

package downutils

import (
	"errors"
	"syscall"
)

func isDiskFull(err error) bool {
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT)
}
