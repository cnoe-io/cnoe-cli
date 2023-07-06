package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnoe-io/cnoe-cli/pkg/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defDir  = "aws-resources"
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
	rootCmd.AddCommand(templateCmd)
	templateCmd.Flags().StringVarP(&inputDir, "inputDir", "i", "", "input directory for CRDs and XRDs to be templatized")
	templateCmd.Flags().StringVarP(&outputDir, "outputDir", "o", "", "output directory for backstage templates to be stored in")
	templateCmd.Flags().StringVarP(&templatePath, "templatePath", "t", "scaffolding/template.yaml", "path to the template to be augmented with backstage info")
	templateCmd.Flags().StringArrayVarP(&verifiers, "verifier", "v", []string{}, "list of verifiers to test the resource against")
	templateCmd.Flags().BoolVarP(&namespaced, "namespaced", "n", false, "whether or not resources are namespaced")

	templateCmd.Flags().StringVarP(&templateName, "templateName", "", "", "sets the name of the template")
	templateCmd.Flags().StringVarP(&templateTitle, "templateTitle", "", "", "sets the title of the template")
	templateCmd.Flags().StringVarP(&templateDescription, "templateDescription", "", "", "sets the description of the template")

	templateCmd.MarkFlagRequired("inputDir")
	templateCmd.MarkFlagRequired("outputDir")
	templateCmd.MarkFlagRequired("templatePath")
}

var (
	templateCmd = &cobra.Command{
		Use:   "template",
		Short: "Generate backstage templates from CRD/XRD",
		Long:  `Generate backstage templates from supplied CRD and XRD definitions`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !isDirectory(inputDir) || !isDirectory(outputDir) {
				return errors.New("inputDir and ouputDir entries need to be directories")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			defs := defs(inputDir, 0)

			output, err := writeSchema(
				outputDir,
				defs,
			)
			if err != nil {
				return err
			}

			err = writeToTemplate(
				templatePath,
				outputDir,
				output.Resources, 0)

			if err != nil {
				return err
			}

			return nil
		},
	}
)

type cmdOutput struct {
	Templates []string
	Resources []string
}

func defs(dir string, depth int) []string {
	if depth > 2 {
		return nil
	}

	var out []string
	base, _ := filepath.Abs(dir)
	files, _ := ioutil.ReadDir(base)
	for _, file := range files {
		f := filepath.Join(base, file.Name())
		stat, _ := os.Stat(f)
		if stat.IsDir() {
			out = append(out, defs(f, depth+1)...)
			continue
		}

		out = append(out, f)
	}

	return out
}

func writeSchema(outputPath string, defs []string) (cmdOutput, error) {
	out := cmdOutput{
		Templates: make([]string, 0),
		Resources: make([]string, 0),
	}

	templateOutputDir := fmt.Sprintf("%s/%s", outputPath, defDir)
	_, err := os.Stat(templateOutputDir)
	if os.IsNotExist(err) {
		// Directory doesn't exist, so create it
		err := os.MkdirAll(templateOutputDir, 0755)
		if err != nil {
			return cmdOutput{}, err
		}
		fmt.Println("Directory created successfully!")
	} else if err != nil {
		return cmdOutput{}, err
	}

	for _, def := range defs {
		data, err := ioutil.ReadFile(def)
		if err != nil {
			continue
		}

		var doc models.Definition
		err = yaml.Unmarshal(data, &doc)
		if err != nil {
			continue
		}

		if !isXRD(doc) && !isCRD(doc) {
			continue
		}

		fmt.Printf("foud: %s\n", def)
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
				fmt.Printf("failed %s: %s \n", def, err.Error())
				continue
			}
		}

		obj := &unstructured.Unstructured{
			Object: make(map[string]interface{}, 0),
		}
		unstructured.SetNestedSlice(obj.Object, ConvertSlice([]string{resourceName}), "properties", "awsResources", "enum")
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
			fmt.Printf("failed %s: %s \n", def, err.Error())
			continue
		}

		template := fmt.Sprintf("%s/%s.yaml", templateOutputDir, strings.ToLower(resourceName))
		err = ioutil.WriteFile(template, []byte(wrapperData), 0644)
		if err != nil {
			fmt.Printf("failed %s: %s \n", def, err.Error())
			continue
		}

		out.Templates = append(out.Templates, template)
		out.Resources = append(out.Resources, resourceName)
	}

	return out, nil
}

func writeToTemplate(templateFile string, outputPath string, resources []string, position int) error {
	templateData, err := ioutil.ReadFile(templateFile)
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
		AwsResources struct {
			OneOf []map[string]interface{} `yaml:"oneOf,omitempty"`
		} `yaml:"awsResources,omitempty"`
	}{}

	awsResources := struct {
		Type string   `yaml:"type"`
		Enum []string `yaml:"enum"`
	}{
		Type: "string",
		Enum: resources,
	}

	for _, resource := range resources {
		dependencies.AwsResources.OneOf = append(dependencies.AwsResources.OneOf, map[string]interface{}{
			"$yaml": fmt.Sprintf("aws-resources/%s.yaml", strings.ToLower(resource)),
		})
	}

	doc.Spec.Parameters[position].Properties["awsResources"] = awsResources
	doc.Spec.Parameters[position].Dependencies = dependencies

	outputData, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s/template.yaml", outputPath), outputData, 0644)
	if err != nil {
		return err
	}

	fmt.Println("Template successfully written.")
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
	originalMap, ok := originalData.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("failed to convert to interface map")
	}

	convertedMap := make(map[string]interface{})

	for key, value := range originalMap {
		strKey, ok := key.(string)
		if !ok {
			// Skip the key if it cannot be converted to string
			continue
		}

		switch v := value.(type) {
		case map[interface{}]interface{}:
			// If the value is a nested map, recursively convert it
			var err error
			convertedMap[strKey], err = ConvertMap(v)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("failed to convert for key %s", strKey))
			}
		case int:
			convertedMap[strKey] = int64(v)
		case int32:
			convertedMap[strKey] = int64(v)
		case []interface{}:
			dv := make([]interface{}, len(v))
			for i, ve := range v {
				switch ive := ve.(type) {
				case map[interface{}]interface{}:
					ivec, err := ConvertMap(ive)
					if err != nil {
						return nil, errors.New(fmt.Sprintf("failed to convert for key %s", strKey))
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
			convertedMap[strKey] = dv
		default:
			// Otherwise, add the key-value pair to the converted map
			convertedMap[strKey] = v
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

func isDirectory(path string) bool {
	// Get file information
	info, err := os.Stat(path)
	if err != nil {
		// Error occurred, path does not exist or cannot be accessed
		return false
	}

	// Check if the path is a directory
	return info.Mode().IsDir()
}
