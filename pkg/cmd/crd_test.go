package cmd_test

import (
	"fmt"
	"path"

	"github.com/cnoe-io/cnoe-cli/pkg/cmd"
	"github.com/cnoe-io/cnoe-cli/pkg/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"gopkg.in/yaml.v3"

	"os"
	"path/filepath"
)

var _ = Describe("Template", func() {
	var (
		tempDir   string
		outputDir string

		stdout *gbytes.Buffer
		stderr *gbytes.Buffer
	)

	const (
		templateName        = "test-name"
		templateTitle       = "test-title"
		templateDescription = "test-description"

		inputDir             = "./fakes/in-resource"
		invalidInputDir      = "./fakes/invalid-in-resource"
		templateFile         = "./fakes/template/input-template.yaml"
		expectedTemplateFile = "./fakes/template/output-template.yaml"
		expectedResourceFile = "./fakes/out-resource/output-resource.yaml"
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "test")
		Expect(err).NotTo(HaveOccurred())

		outputDir = filepath.Join(tempDir, "output")
		err = os.Mkdir(outputDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
	})

	AfterEach(func() {
		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("with valid input", func() {
		BeforeEach(func() {
			err := cmd.Crd(stdout, stderr, inputDir, outputDir, templateFile,
				[]string{}, false, templateName, templateTitle, templateDescription,
			)
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

		It("should create valid resources", func() {
			resourceDir := fmt.Sprintf("%s/%s", outputDir, "resources")
			files, err := os.ReadDir(resourceDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(1))

			filePath := path.Join(resourceDir, files[0].Name())
			Expect(err).NotTo(HaveOccurred())
			generatedResourceData, err := os.ReadFile(filePath)
			Expect(err).NotTo(HaveOccurred())

			var generatedResource models.Definition
			err = yaml.Unmarshal(generatedResourceData, &generatedResource)
			Expect(err).NotTo(HaveOccurred())

			expectedResourceData, err := os.ReadFile(expectedResourceFile)
			Expect(err).NotTo(HaveOccurred())
			var expectedResource models.Definition
			err = yaml.Unmarshal(expectedResourceData, &expectedResource)
			Expect(err).NotTo(HaveOccurred())

			Expect(generatedResource).To(Equal(expectedResource))
		})
	})

	Context("with invalid input files", func() {
		BeforeEach(func() {
			err := cmd.Crd(stdout, stderr, invalidInputDir, outputDir, templateFile,
				[]string{}, false, templateName, templateTitle, templateDescription,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create the template files for valid definitions only", func() {
			resourceDir := fmt.Sprintf("%s/%s", outputDir, "resources")
			files, err := os.ReadDir(resourceDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(1))
		})
	})
})
