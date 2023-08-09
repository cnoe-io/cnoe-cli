package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	tfCmd = &cobra.Command{
		Use:   "tf",
		Short: "Generate backstage templates from Terraform variables",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !isDirectory(inputDir) {
				return errors.New("inputDir and ouputDir entries need to be directories")
			}
			return nil
		},
		RunE: tfE,
	}
	depth uint32
)

type BackstageParamFields struct {
	Title                string `yaml:",omitempty"`
	Type                 string
	Description          string                           `yaml:",omitempty"`
	Default              any                              `yaml:",omitempty"`
	Items                *BackstageParamFields            `yaml:",omitempty"`
	UIWidget             string                           `yaml:"ui:widget,omitempty"`
	Properties           map[string]*BackstageParamFields `yaml:"UiWidget,omitempty"`
	AdditionalProperties AdditionalProperties             `yaml:"additionalProperties,omitempty"`
	UniqueItems          *bool                            `yaml:",omitempty"` // This does not guarantee a set. Works for primitives only.
}

type AdditionalProperties struct { // technically any but for our case, it should be a type: string
	Type string `yaml:",omitempty"`
}

func init() {
	tfCmd.Flags().Uint32Var(&depth, "depth", 2, "depth from given directory to search for TF modules")
	templateCmd.AddCommand(tfCmd)
}

func tfE(cmd *cobra.Command, args []string) error {
	return terraform(inputDir, outputDir)
}

func terraform(inputDir string, outputDir string) error {
	mods := getModules(inputDir, 0)
	if len(mods) == 0 {
		return fmt.Errorf("could not find any TF modules in given directorr: %s", inputDir)
	}

	for i := range mods {
		path := mods[i]
		mod, diag := tfconfig.LoadModule(path)
		if diag.HasErrors() {
			return diag.Err()
		}
		if len(mod.Variables) == 0 {
			fmt.Println(fmt.Sprintf("module %s does not have variables", path))
			continue
		}
		properties := make(map[string]BackstageParamFields)
		for j := range mod.Variables {
			properties[j] = convertVariable(*mod.Variables[j])
		}
		b, err := yaml.Marshal(properties)
		if err != nil {
			fmt.Println(fmt.Sprintf("failed to marshal %s: %s", path, err))
			continue
		}
		fileName := fmt.Sprintf("%s.yaml", filepath.Base(inputDir))
		filePath := filepath.Join(outputDir, fileName)
		err = os.WriteFile(filePath, b, 0644)
		if err != nil {
			fmt.Println(fmt.Sprintf("failed to write %s: %s", path, err))
			continue
		}
	}
	return nil
}

func getModules(inputDir string, currentDepth uint32) []string {
	if currentDepth > depth {
		return nil
	}
	if tfconfig.IsModuleDir(inputDir) {
		return []string{inputDir}
	}
	base, _ := filepath.Abs(inputDir)
	files, _ := os.ReadDir(base)
	out := make([]string, 1)
	for _, file := range files {
		f := filepath.Join(base, file.Name())
		mods := getModules(f, currentDepth+1)
		out = append(out, mods...)
	}
	return out
}

func convertVariable(tfVar tfconfig.Variable) BackstageParamFields {
	tfType := cleanString(tfVar.Type)
	t := mapType(tfType)
	if isPrimitive(tfType) {
		b := BackstageParamFields{
			Type: t,
		}
		if tfVar.Description != "" {
			b.Description = tfVar.Description
		}
		if tfVar.Default != nil {
			b.Default = tfVar.Default
		}
		return b
	}

	if t == "array" {
		return convertArray(tfVar)
	}
	if t == "object" {
		return convertObject(tfVar)
	}
	return BackstageParamFields{}
}

func convertArray(tfVar tfconfig.Variable) BackstageParamFields {
	tfType := cleanString(tfVar.Type)
	nestedType := getNestedType(tfType)
	nestedTfVar := tfconfig.Variable{
		Name: fmt.Sprintf("%s-a", tfVar.Name),
		Type: nestedType,
	}
	nestedItems := convertVariable(nestedTfVar)
	out := BackstageParamFields{
		Type:        "array",
		Description: tfVar.Description,
		Default:     tfVar.Default,
		Items:       &nestedItems,
	}
	if strings.HasPrefix(tfType, "set") {
		u := true
		out.UniqueItems = &u
	}
	return out
}

func convertObjectDefaults(tfVar tfconfig.Variable) map[string]*BackstageParamFields {
	// build default values by taking default's key and type. Must be done for primitives only.
	properties := make(map[string]*BackstageParamFields)
	nestedType := getNestedType(cleanString(tfVar.Type))
	if tfVar.Default != nil {
		d, ok := tfVar.Default.(map[string]any)
		if !ok {
			fmt.Println(fmt.Sprintf("could not determine default type of %s", tfVar.Default))
		}
		for k := range d {
			defaultProp := convertVariable(tfconfig.Variable{
				Name:    k,
				Type:    nestedType,
				Default: d[k],
			})
			properties[k] = &defaultProp
		}
	}
	return properties
}

func convertObject(tfVar tfconfig.Variable) BackstageParamFields {
	out := BackstageParamFields{
		Title:       tfVar.Name,
		Type:        mapType(cleanString(tfVar.Type)),
		Description: tfVar.Description,
	}

	nestedType := getNestedType(cleanString(tfVar.Type))
	if isPrimitive(nestedType) {
		out.AdditionalProperties = AdditionalProperties{Type: mapType(nestedType)}
		// defaults for object type is broken in Backstage atm. In the UI, the default values cannot be removed.
		//properties := convertObjectDefaults(tfVar)
		//if len(properties) > 0 {
		//	out.Properties = properties
		//}
	} else {
		name := fmt.Sprintf("%s-n", tfVar.Name)
		nestedTfVar := tfconfig.Variable{
			Name: name,
			Type: nestedType,
		}
		converted := convertVariable(nestedTfVar)
		out.Properties = map[string]*BackstageParamFields{
			name: &converted,
		}
	}
	return out
}

func cleanString(input string) string {
	return strings.ReplaceAll(input, " ", "")
}

func isPrimitive(s string) bool {
	return s == "string" || s == "number" || s == "bool"
}

func isNestedPrimitive(s string) bool {
	nested := strings.HasPrefix(s, "object(") || strings.HasPrefix(s, "map(") || strings.HasPrefix(s, "list(")
	if nested {
		return isPrimitive(getNestedType(s))
	}
	return false
}

func getNestedType(s string) string {
	if strings.HasPrefix(s, "object(") {
		return strings.TrimSuffix(strings.SplitAfterN(s, "object(", 1)[1], ")")
	}
	if strings.HasPrefix(s, "map(") {
		return strings.TrimSuffix(strings.SplitAfterN(s, "map(", 2)[1], ")")
	}
	if strings.HasPrefix(s, "list(") {
		return strings.TrimSuffix(strings.SplitAfterN(s, "list(", 2)[1], ")")
	}
	return s
}

func mapType(tfType string) string {
	switch {
	case tfType == "string":
		return "string"
	case tfType == "number":
		return "number"
	case tfType == "bool":
		return "boolean"
	case strings.HasPrefix(tfType, "object"), strings.HasPrefix(tfType, "map"):
		return "object"
	case strings.HasPrefix(tfType, "list"), strings.HasPrefix(tfType, "set"):
		return "array"
	default:
		return "string"
	}
}
