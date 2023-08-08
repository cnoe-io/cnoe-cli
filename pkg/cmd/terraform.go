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

// tfCmd represents the tf command
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
	Properties           map[string]*BackstageParamFields `yaml:",omitempty"`
	AdditionalProperties AdditionalProperties             `yaml:",omitempty"`
}

type AdditionalProperties struct {
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
			field := convertVariable(*mod.Variables[j])
			properties[j] = field
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
	b := BackstageParamFields{
		Type: t,
	}
	if tfVar.Description != "" {
		b.Description = tfVar.Description
	}
	if tfVar.Default != nil {
		b.Default = tfVar.Default
	}

	if !isPrimitive(tfType) {
		if strings.HasPrefix(tfType, "list") {
			b.Items = convertArray(tfVar)
		}
		if strings.HasPrefix(tfType, "map") || strings.HasPrefix(tfType, "object") {
			properties := convertObject(tfVar)
			b.Properties = map[string]*BackstageParamFields{
				tfVar.Name: properties,
			}
		}
	}
	return b
}

func convertArray(tfVar tfconfig.Variable) *BackstageParamFields {
	nestedType := getNestedType(cleanString(tfVar.Type))
	nestedTfVar := tfconfig.Variable{
		Type: nestedType,
	}
	nestedItems := convertVariable(nestedTfVar)

	return &BackstageParamFields{
		Type:  "array",
		Items: &nestedItems,
	}
}

func convertObject(tfVar tfconfig.Variable) *BackstageParamFields {
	nestedType := getNestedType(cleanString(tfVar.Type))
	name := fmt.Sprintf("%s-n", tfVar.Name)
	nestedTfVar := tfconfig.Variable{
		Name: name,
		Type: nestedType,
	}
	converted := convertVariable(nestedTfVar)
	return &converted
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
		fmt.Println(strings.SplitAfterN(s, "map(", 2))
		return strings.TrimSuffix(strings.SplitAfterN(s, "map(", 2)[1], ")")
	}
	if strings.HasPrefix(s, "list(") {
		fmt.Println(strings.SplitAfterN(s, "list(", 2))
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
	case strings.HasPrefix(tfType, "list"):
		return "array"
	default:
		return "string"
	}
}
