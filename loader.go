package downutils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig 加载YAML配置文件
func LoadConfig(filename string) (DownConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config DownConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return config, nil
}
