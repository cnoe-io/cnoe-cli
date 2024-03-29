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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const (
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
}

var (
	crdCmd = &cobra.Command{
		Use:     "crd",
		Short:   "Generate backstage templates from CRD/XRD",
		Long:    `Generate backstage templates from supplied CRD and XRD definitions`,
		PreRunE: templatePreRunE,
		RunE:    crd,
	}
)

func crd(cmd *cobra.Command, args []string) error {
	return Process(
		cmd.Context(),
		NewCRDModule(
			inputDir, outputDir, templatePath, insertionPoint, collapsed, raw,
			verifiers, templateName, templateTitle, templateDescription,
		),
	)
}

type CRDModule struct {
	EntityConfig
	verifiers           []string
	templateName        string
	templateTitle       string
	templateDescription string
}

func NewCRDModule(
	inputDir, outputDir, templatePath, insertionPoint string, collapsed, raw bool,
	verifiers []string, templateName, templateTitle, templateDescription string,
) Entity {
	return &CRDModule{
		EntityConfig: EntityConfig{
			InputDir:       inputDir,
			OutputDir:      outputDir,
			TemplateFile:   templatePath,
			InsertionPoint: insertionPoint,
			Collapsed:      collapsed,
			Raw:            raw,
		},
		verifiers:           verifiers,
		templateName:        templateName,
		templateTitle:       templateTitle,
		templateDescription: templateDescription,
	}
}

func (c *CRDModule) Config() EntityConfig {
	return c.EntityConfig
}

func (c *CRDModule) GetDefinitions(inputDir string, currentDepth uint32) ([]string, error) {
	if currentDepth > depth {
		return nil, nil
	}
	out, err := getRelevantFiles(inputDir, currentDepth, c.findDefs)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *CRDModule) findDefs(file os.DirEntry, currentDepth uint32, base string) ([]string, error) {
	f := filepath.Join(base, file.Name())
	stat, err := os.Stat(f)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		df, err := c.GetDefinitions(f, currentDepth+1)
		if err != nil {
			return nil, err
		}
		return df, nil
	}
	return []string{f}, nil
}

func (c *CRDModule) HandleEntry(ctx context.Context, def, expectedOutDir, templateFile string) (any, string, error) {
	log.Printf("processing resource at %s", def)
	converted, resourceName, err := c.convert(def)
	if err != nil {
		var e NotSupported
		if errors.As(err, &e) {
			return nil, "", nil
		}
		return nil, "", err
	}

	fileName := filepath.Join(expectedOutDir, fmt.Sprintf("%s.yaml", strings.ToLower(resourceName)))
	content, err := c.createContent(ctx, converted, templateFile)
	if err != nil {
		log.Printf("failed to write %s: %s \n", def, err.Error())
		return nil, "", err
	}

	return content, fileName, nil
}

func (c *CRDModule) createContent(ctx context.Context, converted any, templateFile string) (any, error) {
	if shouldCreateNonCollapsedTemplate(c) {
		input := insertAtInput{
			templatePath:     templateFile,
			jqPathExpression: c.InsertionPoint,
		}
		props := converted.(map[string]any)
		if v, reqOk := props["required"]; reqOk {
			if reqs, ok := v.([]string); ok {
				input.required = reqs
			}
		}
		input.fields = props
		converted, err := insertAt(ctx, input)
		if err != nil {
			return nil, err
		}
		return converted, nil
	}

	return converted, nil
}

func (c *CRDModule) convert(def string) (any, string, error) {
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
		resourceName = fmt.Sprintf("%s.%s", doc.Spec.Group, doc.Spec.ClaimNames.Kind)
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
			return nil, "", err
		}
	}

	obj := &unstructured.Unstructured{
		Object: make(map[string]interface{}, 0),
	}

	if shouldCreateCollapsedTemplate(c) {
		unstructured.SetNestedSlice(obj.Object, ConvertSlice([]string{strings.ToLower(resourceName)}), "properties", "resources", "enum")
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
		kind := doc.Spec.Names.Kind
		if doc.Spec.ClaimNames != nil {
			kind = doc.Spec.ClaimNames.Kind
		}
		unstructured.SetNestedMap(obj.Object, map[string]interface{}{
			"type":        "string",
			"description": "Kind for the resource",
			"default":     kind,
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
				return nil, fmt.Errorf("failed to convert for key %s", key)
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
						return nil, fmt.Errorf("failed to convert for key %s", key)
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
