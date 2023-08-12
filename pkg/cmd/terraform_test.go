package cmd

import (
	"context"
	"fmt"

	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Terraform Template", func() {
	var (
		tempDir   string
		outputDir string
	)

	const (
		inputDir                        = "./fakes/terraform/valid/input"
		inputDirWithRequire             = "./fakes/terraform/valid/input-require"
		expectedPropertyFile            = "./fakes/terraform/valid/output/properties.yaml"
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
			err := terraform(context.Background(), inputDir, outputDir, "", "")
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
			err := terraform(context.Background(), inputDir, outputDir, targetTemplateFile, ".spec.parameters[0]")
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
			err := terraform(context.Background(), inputDirWithRequire, outputDir, targetTemplateFile, ".spec.parameters[0]")
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
})