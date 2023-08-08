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
		templateFile         = "./fakes/template/input-template.yaml"
		expectedTemplateFile = "./fakes/template/output-template.yaml"
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
			expectedTemplateData, err := os.ReadFile(expectedTemplateFile)
			Expect(err).NotTo(HaveOccurred())

			var expectedTemplate models.Template
			err = yaml.Unmarshal(expectedTemplateData, &expectedTemplate)
			Expect(err).NotTo(HaveOccurred())

			generatedTemplateData, err := os.ReadFile(fmt.Sprintf("%s/%s", outputDir, "template.yaml"))
			Expect(err).NotTo(HaveOccurred())

			var generatedTemplate models.Template
			err = yaml.Unmarshal(generatedTemplateData, &generatedTemplate)
			Expect(err).NotTo(HaveOccurred())
			Expect(generatedTemplate).To(Equal(expectedTemplate))
		})
	})
})
