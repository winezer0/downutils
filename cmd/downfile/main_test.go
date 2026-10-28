package main

import (
	"context"
	"github.com/winezer0/downutils"
	"testing"
)

func TestResultExitCodes(t *testing.T) {
	for _, test := range []struct {
		result *downutils.DownloadResult
		want   int
	}{
		{nil, 1}, {&downutils.DownloadResult{}, 0}, {&downutils.DownloadResult{FailedItems: 1}, 1},
	} {
		if got := resultExitCode(context.Background(), test.result); got != test.want {
			t.Fatalf("exit code %d, want %d", got, test.want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := resultExitCode(ctx, &downutils.DownloadResult{}); got != 130 {
		t.Fatalf("cancel exit code %d", got)
	}
}
