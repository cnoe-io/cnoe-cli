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
	crdCmd.Flags().StringVarP(&inputDir, "inputDir", "i", "", "input directory for CRDs and XRDs to be templatized")
	crdCmd.Flags().StringVarP(&outputDir, "outputDir", "o", "", "output directory for backstage templates to be stored in")
}
