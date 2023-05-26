package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var (
	EnvPrefix = "CNOE_"
	Delimiter = "."
)

func Parse(f string) (Configuration, error) {
	k := koanf.New(Delimiter)

	err := parseConfigFile(k, f)
	if err != nil {
		return Configuration{}, err
	}
	err = parseEnv(k)
	if err != nil {
		return Configuration{}, err
	}
	var out Configuration
	err = k.Unmarshal("", &out)
	return out, nil
}

func parseConfigFile(k *koanf.Koanf, f string) error {
	info, err := os.Stat(f)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("give path must be a file. Given path: %s", f)
	}
	err = k.Load(file.Provider(f), yaml.Parser())
	if err != nil {
		return err
	}
	return nil
}

func parseEnv(k *koanf.Koanf) error {
	err := k.Load(env.ProviderWithValue(EnvPrefix, Delimiter, envParser), nil)
	if err != nil {
		return err
	}
	return nil
}

func envParser(k string, v string) (string, interface{}) {
	key := strings.Replace(strings.ToLower(strings.TrimPrefix(k, EnvPrefix)), "_", Delimiter, -1)
	if strings.Contains(v, " ") {
		return key, strings.Split(v, " ")
	}
	return key, v
}
