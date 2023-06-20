package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	kubeConfig string
	configPath string
	cfg        lib.Config

	rootCmd = &cobra.Command{
		Use:   "cnoe",
		Short: "cnoe cli for building your developer platform",

		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "./config.yaml", "path to config file")
	rootCmd.PersistentFlags().StringVarP(&kubeConfig, "kubeconfig", "k", "~/.kube/config", "path to the kubeconfig file")
}

func Execute() {
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	kubeConfig = os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		kubeConfig = fmt.Sprintf("%s/.kube/config", homeDir)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
