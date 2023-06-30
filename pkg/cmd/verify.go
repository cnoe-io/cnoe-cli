package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"github.com/fatih/color"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
)

const (
	Version = "v1alpha1"
	Group   = "cnoe.io"
)

var (
	configPaths []string

	verifyCmd = &cobra.Command{
		Use:           "verify",
		Short:         "verify if the deployment exists",
		Long:          `verify if the required resources and controllers are working as expected`,
		RunE:          verify,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func init() {
	rootCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().StringArrayVarP(&configPaths, "config", "c", []string{}, "list of prerequisit configurations")
	verifyCmd.MarkFlagRequired("config")
}

func verify(cmd *cobra.Command, args []string) error {
	cli, err := lib.NewK8sClient(kubeConfig)
	if err != nil {
		return err
	}

	configs, err := load()
	if err != nil {
		return err
	}

	return Verify(cmd.OutOrStdout(), cmd.OutOrStderr(), cli, configs)
}

func Verify(stdout, stderr io.Writer, cli lib.IK8sClient, configs []lib.Config) error {
	var result error

	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	for _, config := range configs {

		if config.ApiVersion != fmt.Sprintf("%s/%s", Group, Version) {
			result = multierror.Append(result, errors.New(fmt.Sprintf("apiVersion not matching %s/%s", Group, Version)))
			continue
		}

		if config.Metadata.Name == "" {
			result = multierror.Append(result, errors.New("missing metadata.name"))
		}

		for _, crd := range config.Spec.Crds {
			_, err := cli.CRDs(crd.Group, crd.Kind, crd.Version)
			if err != nil {
				fmt.Fprintf(stdout, "%s %s - %s/%s, Kind=%s\n", red("X"), config.Metadata.Name, crd.Group, crd.Version, crd.Kind)
				result = multierror.Append(result, errors.New(fmt.Sprintf("%s/%s, Kind=%s not found", crd.Group, crd.Version, crd.Kind)))
				continue
			}

			fmt.Fprintf(stdout, "%s %s - %s/%s, Kind=%s\n", green("✓"), config.Metadata.Name, crd.Group, crd.Version, crd.Kind)
		}

		pods, err := cli.Pods("")
		for _, pid := range config.Spec.Pods {
			if err != nil {
				return multierror.Append(result, err)
			}

			found := false
			for _, p := range pods.Items {
				if pid.Namespace != "" && p.GetNamespace() != pid.Namespace {
					continue
				}

				if strings.Contains(p.GetName(), pid.Name) {
					found = true
					if p.Status.Phase == v1.PodRunning {
						fmt.Fprintf(stdout, "%s %s - %s, Pod=%s - %s\n", green("✓"), config.Metadata.Name, p.GetNamespace(), p.GetName(), p.Status.Phase)
					} else {
						fmt.Fprintf(stdout, "%s %s - %s, Pod=%s - %s\n", red("X"), config.Metadata.Name, p.GetNamespace(), p.GetName(), p.Status.Phase)
						result = multierror.Append(result, errors.New(fmt.Sprintf("%s, Pod=%s failed", p.GetNamespace(), p.GetName())))
					}
				}
			}

			if !found {
				fmt.Fprintf(stdout, "%s %s - %s Pod=%s\n", red("X"), config.Metadata.Name, pid.Namespace, pid.Name)
				result = multierror.Append(result, errors.New(fmt.Sprintf("%s Pod=%s not found", pid.Namespace, pid.Name)))
			}
		}
	}

	return result
}
