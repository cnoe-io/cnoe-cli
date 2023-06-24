package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	kubeConfig string
	configPath string

	rootCmd = &cobra.Command{
		Use:   "cnoe",
		Short: "cnoe cli for building your developer platform",

		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&kubeConfig, "kubeconfig", "k", "~/.kube/config", "path to the kubeconfig file")
}

func Execute() {
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
