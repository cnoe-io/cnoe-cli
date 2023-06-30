//go:build embed
// +build embed

package cmd

import (
	"embed"
	"io/fs"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"gopkg.in/yaml.v2"
)

//go:embed yaml/*
var embeddedFiles embed.FS

func load() ([]lib.Config, error) {

	var configs []lib.Config

	files := make(map[string]lib.Config)
	dirEntries, _ := embeddedFiles.ReadDir("yaml")
	for _, entry := range dirEntries {
		data, err := fs.ReadFile(embeddedFiles, "yaml/"+entry.Name())
		if err != nil {
			return configs, err
		}

		var config lib.Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return configs, err
		}
		files[config.Metadata.Name] = config
	}

	for _, configName := range configPaths {
		if val, ok := files[configName]; ok {
			configs = append(configs, val)
		}
	}

	return configs, nil
}
