package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate Backstage templates",
}

const (
	DefinitionsDir = "resources"
)

var (
	depth          uint32
	insertionPoint string
	inputDir       string
	outputDir      string
	templatePath   string
	collapsed      bool
	raw            bool
)

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.PersistentFlags().StringVarP(&inputDir, "inputDir", "i", "", "input directory for CRDs and XRDs to be templatized")
	templateCmd.PersistentFlags().StringVarP(&outputDir, "outputDir", "o", "", "output directory for backstage templates to be stored in")
	templateCmd.PersistentFlags().StringVarP(&templatePath, "templatePath", "t", "", "path to the template to be augmented with backstage info")
	templateCmd.PersistentFlags().Uint32Var(&depth, "depth", 2, "depth from given directory to search for TF modules or CRDs")
	templateCmd.PersistentFlags().StringVarP(&insertionPoint, "insertAt", "p", ".spec.parameters[0]", "jq path within the template to insert backstage info")
	templateCmd.PersistentFlags().BoolVarP(&collapsed, "collapse", "c", false, "if set to true, items are rendered and collapsed as drop down items in a single specified template")
	templateCmd.PersistentFlags().BoolVarP(&raw, "raw", "", false, "prints the raw open API output without putting it into a template (ignoring `templatePath` and `insertAt`)")

	templateCmd.MarkFlagRequired("inputDir")
	templateCmd.MarkFlagRequired("outputDir")
}

func templatePreRunE(cmd *cobra.Command, args []string) error {
	if !isDirectory(inputDir) {
		return errors.New("inputDir must be a directory")
	}

	if collapsed && templatePath == "" && !raw {
		return errors.New("templatePath flag must be specified when using the `collapse` flag (optionally you can use `insertAt` as well.)")
	}

	if templatePath == "" && !raw {
		return errors.New("you either need to use the `raw` flag to generate raw OpenAPI files or define a `templatePath` for the tool to populate.")
	}

	return nil
}
