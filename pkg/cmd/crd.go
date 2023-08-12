package cmd

import (
	"errors"
	"fmt"
	"io"
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
	inputDir     string
	outputDir    string
	templatePath string
	verifiers    []string
	namespaced   bool

	templateName        string
	templateTitle       string
	templateDescription string
)

func init() {
	templateCmd.AddCommand(crdCmd)
	crdCmd.Flags().StringArrayVarP(&verifiers, "verifier", "v", []string{}, "list of verifiers to test the resource against")
	crdCmd.Flags().BoolVarP(&namespaced, "namespaced", "n", false, "whether or not resources are namespaced")

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
			if !isDirectory(inputDir) || !isDirectory(outputDir) {
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
		cmd.OutOrStdout(), cmd.OutOrStderr(),
		inputDir, outputDir, templatePath,
		verifiers, namespaced,
		templateName, templateTitle, templateDescription,
	)
}

func Crd(
	stdout, stderr io.Writer,
	inputDir, outputDir, templatePath string,
	verifiers []string, namespaced bool,
	templateName, templateTitle, templateDescription string,
) error {
	defs, err := getDefs(inputDir, 0)
	if err != nil {
		return err
	}

	output, err := writeSchema(
		stdout, stderr,
		outputDir,
		defs,
	)
	if err != nil {
		return err
	}

	err = writeToTemplate(
		stdout, stderr,
		templatePath,
		outputDir,
		output.Resources, 0,
		templateName,
		templateTitle,
		templateDescription,
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

func writeSchema(stdout, stderr io.Writer, outputDir string, defs []string) (cmdOutput, error) {
	out := cmdOutput{
		Templates: make([]string, 0),
		Resources: make([]string, 0),
	}

	templateOutputDir := fmt.Sprintf("%s/%s", outputDir, defDir)
	_, err := os.Stat(templateOutputDir)
	if os.IsNotExist(err) {
		// Directory doesn't exist, so create it
		err := os.MkdirAll(templateOutputDir, 0755)
		if err != nil {
			return cmdOutput{}, err
		}
		fmt.Fprintf(stdout, "Directory created successfully!")
	} else if err != nil {
		return cmdOutput{}, err
	}

	for _, def := range defs {
		data, err := os.ReadFile(def)
		if err != nil {
			continue
		}

		var doc models.Definition
		err = yaml.Unmarshal(data, &doc)
		if err != nil {
			fmt.Printf("failed to read %s. This file will be excluded. %s", def, err)
			continue
		}

		if !isXRD(doc) && !isCRD(doc) {
			continue
		}

		fmt.Fprintf(stdout, "foud: %s\n", def)
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
				return cmdOutput{}, err
			}
		}

		obj := &unstructured.Unstructured{
			Object: make(map[string]interface{}, 0),
		}
		unstructured.SetNestedSlice(obj.Object, ConvertSlice([]string{resourceName}), "properties", "resources", "enum")
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
		if namespaced {
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

		wrapperData, err := yaml.Marshal(obj.Object)
		if err != nil {
			fmt.Fprintf(stdout, "failed %s: %s \n", def, err.Error())
			continue
		}

		template := fmt.Sprintf("%s/%s.yaml", templateOutputDir, strings.ToLower(resourceName))
		err = os.WriteFile(template, []byte(wrapperData), 0644)
		if err != nil {
			fmt.Fprintf(stdout, "failed %s: %s \n", def, err.Error())
			continue
		}

		out.Templates = append(out.Templates, template)
		out.Resources = append(out.Resources, resourceName)
	}

	return out, nil
}

func writeToTemplate(
	stdout, stderr io.Writer,
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

	resources := struct {
		Type string   `yaml:"type"`
		Enum []string `yaml:"enum"`
	}{
		Type: "string",
		Enum: identifiedResources,
	}

	for _, r := range identifiedResources {
		dependencies.Resources.OneOf = append(dependencies.Resources.OneOf, map[string]interface{}{
			"$yaml": fmt.Sprintf("resources/%s.yaml", strings.ToLower(r)),
		})
	}

	if len(doc.Spec.Parameters) <= position {
		return errors.New("not the right template or input format")
	}

	doc.Spec.Parameters[position].Properties["resources"] = resources
	doc.Spec.Parameters[position].Dependencies = dependencies

	outputData, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/template.yaml", outputPath), outputData, 0644)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Template successfully written.")
	return nil
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
