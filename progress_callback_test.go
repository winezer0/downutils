package downutils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteDownloadsReportsProgress(t *testing.T) {
	const payload = "progress-callback-payload"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	var reports []Progress
	options := NewDefaultDownOptions()
	options.ShowProgress = false
	options.MaxRetries = 1
	options.OnProgress = func(progress Progress) {
		reports = append(reports, progress)
	}

	config := DownConfig{
		"test": {
			{
				Module:       "progress-module",
				FileName:     "asset.bin",
				DownloadURLs: []string{server.URL},
				StorageDir:   t.TempDir(),
				Enable:       true,
			},
		},
	}
	result, err := ExecuteDownloadsWithClientContext(context.Background(), options, config, server.Client())
	if err != nil {
		t.Fatalf("ExecuteDownloadsWithClientContext() error = %v", err)
	}
	if len(result.TaskInfos) != 1 || result.TaskInfos[0].Status != TaskStatusSuccess {
		t.Fatalf("download result = %+v, want one successful task", result.TaskInfos)
	}
	if len(reports) < 2 {
		t.Fatalf("progress callback count = %d, want initial and final reports", len(reports))
	}

	first := reports[0]
	if first.Module != "progress-module" || first.FileName != "asset.bin" || first.Downloaded != 0 {
		t.Fatalf("initial progress = %+v", first)
	}
	last := reports[len(reports)-1]
	if last.Module != "progress-module" || last.FileName != "asset.bin" {
		t.Fatalf("final progress identity = %+v", last)
	}
	if last.Downloaded != int64(len(payload)) || last.Total != int64(len(payload)) {
		t.Fatalf("final progress = %+v, want downloaded and total %d", last, len(payload))
	}
}
