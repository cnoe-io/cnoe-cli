package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate Backstage templates",
}

var (
	depth          uint32
	insertionPoint string
	inputDir       string
	outputDir      string
	templatePath   string
	useOneOf       bool
)

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.PersistentFlags().StringVarP(&inputDir, "inputDir", "i", "", "input directory for CRDs and XRDs to be templatized")
	templateCmd.PersistentFlags().StringVarP(&outputDir, "outputDir", "o", "", "output directory for backstage templates to be stored in")
	templateCmd.PersistentFlags().StringVarP(&templatePath, "templatePath", "t", "", "path to the template to be augmented with backstage info")
	templateCmd.PersistentFlags().Uint32Var(&depth, "depth", 2, "depth from given directory to search for TF modules or CRDs")
	templateCmd.PersistentFlags().StringVarP(&insertionPoint, "insertAt", "p", "", "jq path within the template to insert backstage info")
	templateCmd.PersistentFlags().BoolVarP(&useOneOf, "oneTemplate", "u", false, "if set to true, items are rendered as drop down items in the specified template")

	templateCmd.MarkFlagRequired("inputDir")
	templateCmd.MarkFlagRequired("outputDir")
}

func templatePreRunE(cmd *cobra.Command, args []string) error {
	if !isDirectory(inputDir) {
		return errors.New("inputDir must be a directory")
	}

	if useOneOf && templatePath == "" && insertionPoint == "" {
		return errors.New("templatePath and inertAt flags must be specified when using the oneTemplate flag")
	}

	if insertionPoint != "" && templatePath == "" {
		return errors.New("templatePath flag must be specified")
	}
	return nil
}
