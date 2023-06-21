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

func init() {
	rootCmd.AddCommand(verifyCmd)
}

var (
	verifyCmd = &cobra.Command{
		Use:           "verify",
		Short:         "verify if the deployment exists",
		Long:          `verify if the required resources and controllers are working as expected`,
		RunE:          verify,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func verify(cmd *cobra.Command, args []string) error {
	cli, err := lib.NewK8sClient(kubeConfig)
	if err != nil {
		return err
	}

	return Verify(cmd.OutOrStdout(), cmd.OutOrStderr(), cli, cfg)
}

func Verify(stdout, stderr io.Writer, cli lib.IK8sClient, config lib.Config) error {
	var result error

	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	for _, op := range config.Prerequisits {
		for _, crd := range op.Crds {
			list, err := cli.CRDs(crd.Group, crd.Kind, crd.Version)
			if err != nil {
				fmt.Fprintf(stdout, "%s %s/%s, Kind=%s\n", red("X"), crd.Group, crd.Version, crd.Kind)
				result = multierror.Append(result, errors.New(fmt.Sprintf("%s/%s, Kind=%s not found", crd.Group, crd.Version, crd.Kind)))
				continue
			}

			fmt.Fprintf(stdout, "%s %s\n", green("✓"), list.GroupVersionKind().String())
		}

		pods, err := cli.Pods("")
		for _, pid := range op.Pods {
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
						fmt.Fprintf(stdout, "%s %s, Pod=%s - %s\n", green("✓"), p.GetNamespace(), p.GetName(), p.Status.Phase)
					} else {
						fmt.Fprintf(stdout, "%s %s, Pod=%s - %s\n", red("X"), p.GetNamespace(), p.GetName(), p.Status.Phase)
						result = multierror.Append(result, errors.New(fmt.Sprintf("%s, Pod=%s failed", p.GetNamespace(), p.GetName())))
					}
				}
			}

			if !found {
				fmt.Fprintf(stdout, "%s %s Pod=%s\n", red("X"), pid.Namespace, pid.Name)
				result = multierror.Append(result, errors.New(fmt.Sprintf("%s Pod=%s not found", pid.Namespace, pid.Name)))
			}
		}
	}

	return result
}
