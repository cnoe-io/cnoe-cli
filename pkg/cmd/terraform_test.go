package cmd_test

import (
	"context"
	"fmt"

	"os"
	"path/filepath"

	"github.com/cnoe-io/cnoe-cli/pkg/cmd"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Terraform Template", func() {
	var (
		tempDir   string
		outputDir string
	)

	const (
		validInputRootDir               = "./fakes/terraform/valid"
		inputDir                        = "./fakes/terraform/valid/input"
		inputDirWithRequire             = "./fakes/terraform/valid/input-require"
		expectedPropertyFile            = "./fakes/terraform/valid/output/properties.yaml"
		expectedPropertyFileWithRequire = "./fakes/terraform/valid/output/properties-require.yaml"
		expectedTemplateFile            = "./fakes/terraform/valid/output/full-template.yaml"
		expectedTemplateFileWithRequire = "./fakes/terraform/valid/output/full-template-require.yaml"
		targetTemplateFile              = "./fakes/template/input-template.yaml"
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

	Context("with valid input and no target template specified", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewTerraformModule(inputDir, outputDir, "", "", false, true))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create the skeleton file for valid definitions", func() {
			generatedData, err := os.ReadFile(fmt.Sprintf("%s/input.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			expectedData, err := os.ReadFile(expectedPropertyFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(generatedData).To(MatchYAML(expectedData))
		})

	})
	Context("with valid input and a target template specified", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewTerraformModule(inputDir, outputDir, targetTemplateFile, ".spec.parameters[0]", false, false))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create the template file with properties merged", func() {
			generatedData, err := os.ReadFile(fmt.Sprintf("%s/input.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			expectedData, err := os.ReadFile(expectedTemplateFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(generatedData).To(MatchYAML(expectedData))
		})

		It("should create the template file with properties merged and requirements updated", func() {
			generatedData, err := os.ReadFile(fmt.Sprintf("%s/input.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			expectedData, err := os.ReadFile(expectedTemplateFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(generatedData).To(MatchYAML(expectedData))
		})

	})
	Context("with valid input with required variable and a target template specified", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewTerraformModule(inputDirWithRequire, outputDir, targetTemplateFile, ".spec.parameters[0]", false, false))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create the template file with properties merged and requirements updated", func() {
			generatedData, err := os.ReadFile(fmt.Sprintf("%s/input-require.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			expectedData, err := os.ReadFile(expectedTemplateFileWithRequire)
			Expect(err).NotTo(HaveOccurred())
			Expect(generatedData).To(MatchYAML(expectedData))
		})
	})

	Context("with a root directory specified", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewTerraformModule(validInputRootDir, outputDir, "", "", false, true))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create properties files with properties merged and requirements updated", func() {
			generatedInputData, err := os.ReadFile(fmt.Sprintf("%s/input.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			generatedInputRequireData, err := os.ReadFile(fmt.Sprintf("%s/input-require.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			expectedInputData, err := os.ReadFile(expectedPropertyFile)
			Expect(err).NotTo(HaveOccurred())
			expectedInputRequireData, err := os.ReadFile(expectedPropertyFileWithRequire)
			Expect(err).NotTo(HaveOccurred())
			Expect(generatedInputData).To(MatchYAML(expectedInputData))
			Expect(generatedInputRequireData).To(MatchYAML(expectedInputRequireData))
		})
	})

	Context("with an invalid input and no target template specified", func() {
		It("should return an error", func() {
			err := cmd.Process(context.Background(), cmd.NewTerraformModule("./fakes/terraform/invalid", outputDir, "", "", false, false))
			Expect(err).Should(HaveOccurred())
		})
	})

	Context("with a root directory and oneOf flag specified", func() {
		BeforeEach(func() {
			err := cmd.Process(context.Background(), cmd.NewTerraformModule(validInputRootDir, outputDir, targetTemplateFile, ".spec.parameters[0]", true, false))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create properties files with properties merged and requirements updated", func() {
			generatedInputData, err := os.ReadFile(fmt.Sprintf("%s/resources/input.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			generatedInputRequireData, err := os.ReadFile(fmt.Sprintf("%s/resources/input-require.yaml", outputDir))
			Expect(err).NotTo(HaveOccurred())
			expectedInputData, err := os.ReadFile(expectedPropertyFile)
			Expect(err).NotTo(HaveOccurred())
			expectedInputRequireData, err := os.ReadFile(expectedPropertyFileWithRequire)
			Expect(err).NotTo(HaveOccurred())

			expectedTempalteData, err := os.ReadFile("./fakes/terraform/valid/output/full-template-oneof.yaml")
			generatedTemplateData, err := os.ReadFile(fmt.Sprintf("%s/template.yaml", outputDir))

			Expect(generatedInputData).To(MatchYAML(expectedInputData))
			Expect(generatedInputRequireData).To(MatchYAML(expectedInputRequireData))
			Expect(generatedTemplateData).To(MatchYAML(expectedTempalteData))

		})
	})
})
