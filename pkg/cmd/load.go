//go:build !embed
// +build !embed

package cmd

import (
	"os"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"gopkg.in/yaml.v2"
)

func load() ([]lib.Config, error) {
	var configs []lib.Config
	for _, configPath := range configPaths {
		yamlFile, err := os.ReadFile(configPath)
		if err != nil {
			return configs, err
		}

		var config lib.Config
		err = yaml.Unmarshal(yamlFile, &config)
		if err != nil {
			return configs, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}
