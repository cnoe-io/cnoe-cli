package cmd

import (
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate Backstage templates",
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.PersistentFlags().StringVarP(&inputDir, "inputDir", "i", "", "input directory for CRDs and XRDs to be templatized")
	templateCmd.PersistentFlags().StringVarP(&outputDir, "outputDir", "o", "", "output directory for backstage templates to be stored in")
	templateCmd.PersistentFlags().StringVarP(&templatePath, "templatePath", "t", "scaffolding/template.yaml", "path to the template to be augmented with backstage info")
	templateCmd.PersistentFlags().Uint32Var(&depth, "depth", 2, "depth from given directory to search for TF modules or CRDs")
	templateCmd.PersistentFlags().StringVarP(&insertionPoint, "insertAt", "p", "", "jq path within the template to insert backstage info")

	templateCmd.MarkFlagRequired("inputDir")
	templateCmd.MarkFlagRequired("outputDir")
}
