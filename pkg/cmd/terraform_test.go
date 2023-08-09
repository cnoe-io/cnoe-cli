package cmd

import (
	"fmt"

	"github.com/cnoe-io/cnoe-cli/pkg/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	"os"
	"path/filepath"
)

var _ = Describe("Terraform Template", func() {
	var (
		tempDir   string
		outputDir string
	)

	const (
		templateName        = "test-name"
		templateTitle       = "test-title"
		templateDescription = "test-description"

		inputDir             = "./fakes/terraform/valid/input"
		expectedTemplateFile = "./fakes/terraform/valid/output/properties.yaml"
		expectedResourceFile = "./fakes/terraform/valid/output"
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "test-temp")
		Expect(err).NotTo(HaveOccurred())

		outputDir = filepath.Join(tempDir, "output")
		err = os.Mkdir(outputDir, 0755)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("with valid input", func() {
		BeforeEach(func() {
			err := terraform(inputDir, outputDir)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should create the template files for valid definitions", func() {
			generatedData, err := os.ReadFile(fmt.Sprintf("%s/input.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())

			expectedData, err := os.ReadFile(expectedTemplateFile)

			var generated map[string]models.BackstageParamFields
			err = yaml.Unmarshal(generatedData, &generated)
			var expected map[string]models.BackstageParamFields
			err = yaml.Unmarshal(expectedData, &expected)
			Expect(err).NotTo(HaveOccurred())
			Expect(generated).To(Equal(expected))
		})
	})
})
