package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnoe-io/cnoe-cli/pkg/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defDir  = "resources"
	KindXRD = "CompositeResourceDefinition"
	KindCRD = "CustomResourceDefinition"
)

var (
	verifiers []string

	templateName        string
	templateTitle       string
	templateDescription string
)

func init() {
	templateCmd.AddCommand(crdCmd)
	crdCmd.Flags().StringArrayVarP(&verifiers, "verifier", "v", []string{}, "list of verifiers to test the resource against")

	crdCmd.Flags().StringVarP(&templateName, "templateName", "", "", "sets the name of the template")
	crdCmd.Flags().StringVarP(&templateTitle, "templateTitle", "", "", "sets the title of the template")
	crdCmd.Flags().StringVarP(&templateDescription, "templateDescription", "", "", "sets the description of the template")

	crdCmd.MarkFlagRequired("templatePath")
}

var (
	crdCmd = &cobra.Command{
		Use:   "crd",
		Short: "Generate backstage templates from CRD/XRD",
		Long:  `Generate backstage templates from supplied CRD and XRD definitions`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !isDirectory(inputDir) {
				return errors.New("inputDir and ouputDir entries need to be directories")
			}
			return nil
		},
		RunE: crd,
	}
)

type cmdOutput struct {
	Templates []string
	Resources []string
}

func crd(cmd *cobra.Command, args []string) error {
	return Crd(
		cmd.Context(),
		inputDir, outputDir, templatePath, insertionPoint,
		verifiers, useOneOf,
		templateName, templateTitle, templateDescription,
	)
}

func Crd(
	ctx context.Context, inputDir, outputDir, templatePath, insertionPoint string,
	verifiers []string, useOneOf bool,
	templateName, templateTitle, templateDescription string,
) error {
	inDir, outDir, template, err := prepDirectories(inputDir, outputDir, templatePath, useOneOf)
	if err != nil {
		return err
	}
	defs, err := getDefs(inDir, 0)
	if err != nil {
		return err
	}
	log.Printf("processing %d definitions", len(defs))

	schemaDir := outDir
	if useOneOf {
		schemaDir = filepath.Join(outDir, defDir)
		output, err := writeSchema(
			ctx,
			schemaDir,
			"",
			"",
			defs,
		)
		if err != nil {
			return err
		}
		templateFile := filepath.Join(outDir, "template.yaml")
		return writeOneOf(ctx, templateFile, template, insertionPoint, output.Templates)
	}
	_, err = writeSchema(
		ctx,
		schemaDir,
		insertionPoint,
		template,
		defs,
	)
	if err != nil {
		return err
	}

	return nil
}

func getDefs(inputDir string, currentDepth uint32) ([]string, error) {
	if currentDepth > depth {
		return nil, nil
	}
	out, err := getRelevantFiles(inputDir, currentDepth, findDefs)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func findDefs(file os.DirEntry, currentDepth uint32, base string) ([]string, error) {
	f := filepath.Join(base, file.Name())
	stat, err := os.Stat(f)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		df, err := getDefs(f, currentDepth+1)
		if err != nil {
			return nil, err
		}
		return df, nil
	}
	return []string{f}, nil
}

func writeSchema(ctx context.Context, outputDir, insertionPoint, templateFile string, defs []string) (cmdOutput, error) {
	out := cmdOutput{
		Templates: make([]string, 0),
		Resources: make([]string, 0),
	}

	for _, def := range defs {
		converted, resourceName, err := convert(def)
		if err != nil {
			var e NotSupported
			if errors.As(err, &e) {
				continue
			}
			return cmdOutput{}, err
		}
		filename := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", strings.ToLower(resourceName)))
		if templateFile != "" && insertionPoint != "" {
			input := insertAtInput{
				templatePath:     templateFile,
				jqPathExpression: insertionPoint,
			}
			props := converted.(map[string]any)
			if v, reqOk := props["required"]; reqOk {
				if reqs, ok := v.([]string); ok {
					input.required = reqs
				}
			}
			input.fields = props
			t, err := insertAt(ctx, input)
			if err != nil {
				return cmdOutput{}, err
			}
			converted = t
		}
		err = writeOutput(converted, filename)
		if err != nil {
			log.Printf("failed to write %s: %s \n", def, err.Error())
			return cmdOutput{}, err
		}
		out.Templates = append(out.Templates, filename)
		out.Resources = append(out.Resources, resourceName)
	}

	return out, nil
}

func writeToTemplate(
	templateFile string, outputPath string, identifiedResources []string, position int,
	templateName, templateTitle, templateDescription string,
) error {
	templateData, err := os.ReadFile(templateFile)
	if err != nil {
		return err
	}

	var doc models.Template
	err = yaml.Unmarshal(templateData, &doc)
	if err != nil {
		return err
	}

	if templateName != "" {
		doc.Metadata.Name = templateName
	}

	if templateTitle != "" {
		doc.Metadata.Title = templateTitle
	}

	if templateDescription != "" {
		doc.Metadata.Description = templateDescription
	}

	dependencies := struct {
		Resources struct {
			OneOf []map[string]interface{} `yaml:"oneOf,omitempty"`
		} `yaml:"resources,omitempty"`
	}{}

	for _, r := range identifiedResources {
		dependencies.Resources.OneOf = append(dependencies.Resources.OneOf, map[string]interface{}{
			"$yaml": fmt.Sprintf("resources/%s.yaml", strings.ToLower(r)),
		})
	}

	if len(doc.Spec.Parameters) <= position {
		return errors.New("not the right template or input format")
	}

	doc.Spec.Parameters[position].Dependencies = dependencies

	outputData, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/template.yaml", outputPath), outputData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func convert(def string) (any, string, error) {
	data, err := os.ReadFile(def)
	if err != nil {
		return nil, "", err
	}
	var doc models.Definition
	err = yaml.Unmarshal(data, &doc)
	if err != nil {
		log.Printf("failed to read %s. This file will be excluded. %s", def, err)
		return nil, "", NotSupported{
			fmt.Errorf("%s is not a kubernetes file", def),
		}
	}

	if !isXRD(doc) && !isCRD(doc) {
		return nil, "", NotSupported{
			fmt.Errorf("%s is not a CRD or XRD", def),
		}
	}
	var resourceName string
	if doc.Spec.ClaimNames != nil {
		resourceName = doc.Spec.ClaimNames.Kind
	} else {
		resourceName = fmt.Sprintf("%s.%s", doc.Spec.Group, doc.Spec.Names.Kind)
	}

	var value map[string]interface{}

	v := doc.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	if v == nil {
		value = doc.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties
	} else {
		value, err = ConvertMap(v)
		if err != nil {
			return cmdOutput{}, "", err
		}
	}

	obj := &unstructured.Unstructured{
		Object: make(map[string]interface{}, 0),
	}
	unstructured.SetNestedMap(obj.Object, value, "properties", "config")
	unstructured.SetNestedField(obj.Object, fmt.Sprintf("%s configuration options", resourceName), "properties", "config", "title")

	// setting GVK for the resource
	if len(doc.Spec.Versions) > 0 {
		unstructured.SetNestedMap(obj.Object, map[string]interface{}{
			"type":        "string",
			"description": "APIVersion for the resource",
			"default":     fmt.Sprintf("%s/%s", doc.Spec.Group, doc.Spec.Versions[0].Name),
		},
			"properties", "apiVersion")
		unstructured.SetNestedMap(obj.Object, map[string]interface{}{
			"type":        "string",
			"description": "Kind for the resource",
			"default":     doc.Spec.Names.Kind,
		},
			"properties", "kind")
	}

	// add a property to define the namespace for the resource
	if doc.Spec.Scope == "Namespaced" {
		unstructured.SetNestedMap(obj.Object, map[string]interface{}{
			"type":        "string",
			"description": "Namespace for the resource",
			"namespace":   "default",
		},
			"properties", "namespace")
	}

	// add verifiers to the resource
	if len(verifiers) > 0 {
		var convertedVerifiers []interface{} = make([]interface{}, len(verifiers))
		for i, v := range verifiers {
			convertedVerifiers[i] = v
		}

		unstructured.SetNestedMap(obj.Object, map[string]interface{}{
			"type":        "array",
			"description": "verifiers to be used against the resource",
			"items":       map[string]interface{}{"type": "string"},
			"default":     convertedVerifiers,
		},
			"properties", "verifiers")
	}
	return obj.Object, resourceName, nil
}

func ConvertSlice(strSlice []string) []interface{} {
	var ifaceSlice []interface{}
	for _, s := range strSlice {
		ifaceSlice = append(ifaceSlice, s)
	}
	return ifaceSlice
}

func ConvertMap(originalData interface{}) (map[string]interface{}, error) {
	originalMap, ok := originalData.(map[string]interface{})
	if !ok {
		return nil, errors.New("conversion failed: data is not map[string]interface{}")
	}

	convertedMap := make(map[string]interface{})

	for key, value := range originalMap {
		switch v := value.(type) {
		case map[interface{}]interface{}:
			// If the value is a nested map, recursively convert it
			var err error
			convertedMap[key], err = ConvertMap(v)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("failed to convert for key %s", key))
			}
		case int:
			convertedMap[key] = int64(v)
		case int32:
			convertedMap[key] = int64(v)
		case []interface{}:
			dv := make([]interface{}, len(v))
			for i, ve := range v {
				switch ive := ve.(type) {
				case map[interface{}]interface{}:
					ivec, err := ConvertMap(ive)
					if err != nil {
						return nil, errors.New(fmt.Sprintf("failed to convert for key %s", key))
					}
					dv[i] = ivec
				case int:
					dv[i] = int64(ive)
				case int32:
					dv[i] = int64(ive)
				default:
					dv[i] = ive
				}
			}
			convertedMap[key] = dv
		default:
			// Otherwise, add the key-value pair to the converted map
			convertedMap[key] = v
		}
	}

	return convertedMap, nil
}

func isXRD(m models.Definition) bool {
	return m.Kind == KindXRD
}

func isCRD(m models.Definition) bool {
	return m.Kind == KindCRD
}
