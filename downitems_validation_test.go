package downutils

import (
	"testing"
)

// TestValidateDownItem 测试配置验证功能
func TestValidateDownItem(t *testing.T) {
	tests := []struct {
		name        string
		items       []DownItem
		enableForce bool
		wantErrs    int
	}{
		{
			name: "valid enabled item",
			items: []DownItem{
				{
					Module:       "test",
					FileName:     "test.txt",
					DownloadURLs: []string{"https://example.com/file"},
					Enable:       true,
				},
			},
			enableForce: false,
			wantErrs:    0,
		},
		{
			name: "empty module name",
			items: []DownItem{
				{
					Module:       "",
					FileName:     "test.txt",
					DownloadURLs: []string{"https://example.com/file"},
					Enable:       true,
				},
			},
			enableForce: false,
			wantErrs:    1,
		},
		{
			name: "empty file name",
			items: []DownItem{
				{
					Module:       "test",
					FileName:     "",
					DownloadURLs: []string{"https://example.com/file"},
					Enable:       true,
				},
			},
			enableForce: false,
			wantErrs:    1,
		},
		{
			name: "invalid file name characters",
			items: []DownItem{
				{
					Module:       "test",
					FileName:     "test<file>.txt",
					DownloadURLs: []string{"https://example.com/file"},
					Enable:       true,
				},
			},
			enableForce: false,
			wantErrs:    1,
		},
		{
			name: "empty download URLs",
			items: []DownItem{
				{
					Module:       "test",
					FileName:     "test.txt",
					DownloadURLs: []string{},
					Enable:       true,
				},
			},
			enableForce: false,
			wantErrs:    1,
		},
		{
			name: "empty URL in list",
			items: []DownItem{
				{
					Module:       "test",
					FileName:     "test.txt",
					DownloadURLs: []string{"https://example.com/file", ""},
					Enable:       true,
				},
			},
			enableForce: false,
			wantErrs:    1,
		},
		{
			name: "disabled item skipped when enableForce is false",
			items: []DownItem{
				{
					Module:       "",
					FileName:     "",
					DownloadURLs: []string{},
					Enable:       false,
				},
			},
			enableForce: false,
			wantErrs:    0,
		},
		{
			name: "disabled item validated when enableForce is true",
			items: []DownItem{
				{
					Module:       "",
					FileName:     "",
					DownloadURLs: []string{},
					Enable:       false,
				},
			},
			enableForce: true,
			wantErrs:    3,
		},
		{
			name: "multiple errors in one item",
			items: []DownItem{
				{
					Module:       "",
					FileName:     "test<file>.txt",
					DownloadURLs: []string{},
					Enable:       true,
				},
			},
			enableForce: false,
			wantErrs:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateDownItem(tt.items, tt.enableForce)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateDownItem() got %d errors, want %d errors, errors: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

// TestValidateDownConfig 测试整个配置验证功能
func TestValidateDownConfig(t *testing.T) {
	config := DownConfig{
		"group1": []DownItem{
			{
				Module:       "test1",
				FileName:     "test1.txt",
				DownloadURLs: []string{"https://example.com/file1"},
				Enable:       true,
			},
		},
		"group2": []DownItem{
			{
				Module:       "",
				FileName:     "",
				DownloadURLs: []string{},
				Enable:       true,
			},
		},
	}

	errs := ValidateDownConfig(config, false)
	if len(errs) != 3 {
		t.Errorf("ValidateDownConfig() got %d errors, want 3 errors, errors: %v", len(errs), errs)
	}

	// 验证错误信息包含组名
	for _, err := range errs {
		if err[:7] != "[group1" && err[:7] != "[group2" {
			t.Errorf("error message should contain group name, got: %s", err)
		}
	}
}
