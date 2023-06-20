package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	defDir  = "aws-resources"
	KindXRD = "CompositeResourceDefinition"
	KindCRD = "CustomResourceDefinition"
)

var (
	component string
)

func init() {
	rootCmd.AddCommand(verifyCmd)
}

var (
	verifyCmd = &cobra.Command{
		Use:   "verify",
		Short: "verify if the deployment exists",
		Long:  `verify if the required resources and controllers are working as expected`,
		RunE:  verify,
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
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	for _, op := range config.Prerequisits {
		for _, crd := range op.Crds {
			list, err := cli.CRDs(crd.Group, crd.Kind, crd.Version)
			if err != nil {
				fmt.Fprintf(stdout, "%s %s/%s, Kind=%s\n", red("X"), crd.Group, crd.Version, crd.Kind)
				continue
			}

			fmt.Fprintf(stdout, "%s %s\n", green("✓"), list.GroupVersionKind().String())
		}

		pods, err := cli.Pods("")
		for _, pid := range op.Pods {
			if err != nil {
				return err
			}

			found := false
			for _, p := range pods.Items {
				if pid.Namespace != "" && p.GetNamespace() != pid.Namespace {
					continue
				}

				if strings.Contains(p.GetName(), pid.Name) {
					found = true
					fmt.Fprintf(stdout, "%s %s, Pod=%s\n", green("✓"), p.GetNamespace(), p.GetName())
				}
			}

			if !found {
				fmt.Fprintf(stdout, "%s %s Pod=%s\n", red("X"), pid.Namespace, pid.Name)
			}
		}
	}

	return nil
}
