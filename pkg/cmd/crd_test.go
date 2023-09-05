package cmd_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cnoe-io/cnoe-cli/pkg/cmd"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Template CRDs", func() {
	var (
		tempDir   string
		outputDir string

		stdout *gbytes.Buffer
	)

	const (
		templateName        = "test-name"
		templateTitle       = "test-title"
		templateDescription = "test-description"

		inputDir             = "./fakes/crd/valid/input"
		validOutputDir       = "./fakes/crd/valid/output"
		invalidInputDir      = "./fakes/crd/invalid/input"
		templateFile         = "./fakes/template/input-template.yaml"
		expectedTemplateFile = "./fakes/crd/valid/output/full-template-oneof.yaml"
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "test")
		Expect(err).NotTo(HaveOccurred())

		outputDir = filepath.Join(tempDir, "output")
		err = os.Mkdir(outputDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		stdout = gbytes.NewBuffer()
		log.SetOutput(stdout)
	})

	AfterEach(func() {
		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("with valid input with oneof", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewCRDModule(inputDir, outputDir, templateFile, ".spec.parameters[0]", true, false,
				[]string{}, templateName, templateTitle, templateDescription,
			))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create the template files for valid definitions", func() {
			expectedTemplateData, err := os.ReadFile(expectedTemplateFile)
			Expect(err).NotTo(HaveOccurred())
			generatedTemplateData, err := os.ReadFile(fmt.Sprintf("%s/%s", outputDir, "template.yaml"))
			Expect(err).NotTo(HaveOccurred())

			Expect(expectedTemplateData).To(MatchYAML(generatedTemplateData))
		})

		It("should create valid resources", func() {
			resourceDir := filepath.Join(outputDir, "resources")
			files, err := os.ReadDir(resourceDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(2))

			for i := range files {
				filePath := filepath.Join(resourceDir, files[i].Name())
				genrated, err := os.ReadFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				propFile := fmt.Sprintf("properties-%s", files[i].Name())
				expected, err := os.ReadFile(filepath.Join(validOutputDir, propFile))
				Expect(err).NotTo(HaveOccurred())
				Expect(genrated).To(MatchYAML(expected))
			}
		})
	})

	Context("with valid input and specify template file and jq path", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewCRDModule(inputDir, outputDir, templateFile, ".spec.parameters[0]", false, false,
				[]string{}, templateName, templateTitle, templateDescription,
			))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create valid backstage template for each definition", func() {
			files, err := os.ReadDir(outputDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(2))
			for i := range files {
				filePath := filepath.Join(outputDir, files[i].Name())
				generated, err := os.ReadFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				propFile := fmt.Sprintf("full-template-%s", files[i].Name())
				expected, err := os.ReadFile(filepath.Join(validOutputDir, propFile))
				Expect(err).NotTo(HaveOccurred())
				Expect(generated).To(MatchYAML(expected))
			}
		})
	})

	Context("with invalid input only", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewCRDModule(invalidInputDir, outputDir, "", "", false, false,
				[]string{}, templateName, templateTitle, templateDescription,
			))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not create any files", func() {
			files, err := os.ReadDir(outputDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(0))
		})
	})
})
