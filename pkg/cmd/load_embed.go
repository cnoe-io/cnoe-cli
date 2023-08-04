//go:build embed
// +build embed

package cmd

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v3"
)

//go:embed prereq/*
var embeddedFiles embed.FS

func load() ([]lib.Config, error) {

	var (
		configs []lib.Config
		result  error
	)

	files := make(map[string]lib.Config)
	dirEntries, _ := embeddedFiles.ReadDir("prereq")
	for _, entry := range dirEntries {
		data, err := fs.ReadFile(embeddedFiles, "prereq/"+entry.Name())
		if err != nil {
			return configs, multierror.Append(result, err)
		}

		var config lib.Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return configs, multierror.Append(result, err)
		}
		files[config.Metadata.Name] = config
	}

	for _, configName := range configPaths {
		if val, ok := files[configName]; ok {
			configs = append(configs, val)
		} else {
			return configs, multierror.Append(result, errors.New(fmt.Sprintf("verifier not found: %s", configName)))
		}
	}

	return configs, nil
}
