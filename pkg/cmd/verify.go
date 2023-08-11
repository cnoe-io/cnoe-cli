package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

const (
	Version = "v1alpha1"
	Group   = "cnoe.io"
	Kind    = "Prerequisite"
)

var (
	configPaths []string

	verifyCmd = &cobra.Command{
		Use:           "verify",
		Short:         "Verify if the deployment exists",
		Long:          `Verify if the required resources and controllers are working as expected`,
		RunE:          verify,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func init() {
	k8sCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().StringArrayVarP(&configPaths, "config", "c", []string{}, "list of prerequisit configurations (samples under config/prereq)")
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

	for _, config := range configs {

		if config.ApiVersion != fmt.Sprintf("%s/%s", Group, Version) || config.Kind != Kind {
			result = multierror.Append(result, errors.New(fmt.Sprintf("apiVersion or kind not matching %s/%s:%s", Group, Version, Kind)))
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
					if string(p.Status.Phase) == pid.State {
						fmt.Fprintf(stdout, "%s %s - %s, Pod=%s - %s\n", green("✓"), config.Metadata.Name, p.GetNamespace(), p.GetName(), p.Status.Phase)
					} else {
						fmt.Fprintf(stdout, "%s %s - %s, Pod=%s - %s != %s \n", red("X"), config.Metadata.Name, p.GetNamespace(), p.GetName(), p.Status.Phase, pid.State)
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
