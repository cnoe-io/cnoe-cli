package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	kubeConfig string

	k8sCmd = &cobra.Command{
		Use:   "k8s",
		Short: "Run against a kubernets cluster",
		Long:  "Commands that assume a kubernetes cluster as the backend",
	}
)

func init() {
	rootCmd.AddCommand(k8sCmd)
	k8sCmd.PersistentFlags().StringVarP(&kubeConfig, "kubeconfig", "k", "~/.kube/config", "path to the kubeconfig file")

	kubeConfig = os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		kubeConfig = fmt.Sprintf("%s/.kube/config", homeDir)
	}
}
