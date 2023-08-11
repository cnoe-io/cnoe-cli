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
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/spf13/cobra"
)

var (
	tfCmd = &cobra.Command{
		Use:   "tf",
		Short: "Generate backstage templates from Terraform variables",
		Long: "Generate backstage templates by walking the given input directory, find TF modules," +
			"then create output file per module.\n" +
			"If the templatePath and insertionPoint flags are set, generated objects are merged into the given template at given insertion point.\n" +
			"Otherwise a yaml file with two keys are generated. The properties key contains the generated form input. " +
			"The required key contains the TF variable names that do not have defaults.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !isDirectory(inputDir) {
				return errors.New("inputDir and ouputDir entries need to be directories")
			}
			return nil
		},
		RunE: tfE,
	}
	depth          uint32
	insertionPoint string
)

func init() {
	templateCmd.AddCommand(tfCmd)
}

func tfE(cmd *cobra.Command, args []string) error {
	return terraform(cmd.Context(), inputDir, outputDir, templatePath, insertionPoint)
}

func terraform(ctx context.Context, inputDir, outputDir, templatePath, insertionPoint string) error {
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
		params := make(map[string]models.BackstageParamFields)
		required := make([]string, 0)
		for j := range mod.Variables {
			params[j] = convertVariable(*mod.Variables[j])
			if mod.Variables[j].Required {
				required = append(required, j)
			}
		}
		filePath := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", filepath.Base(inputDir)))
		err := handleOutput(ctx, filePath, templatePath, insertionPoint, params, required)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func handleOutput(ctx context.Context, outputFile, templatePath, insertionPoint string, properties map[string]models.BackstageParamFields, required []string) error {
	if templatePath != "" && insertionPoint != "" {
		input := insertAtInput{
			templatePath:     templatePath,
			jqPathExpression: insertionPoint,
			properties:       properties,
			required:         required,
		}
		t, err := insertAt(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to insert to given template: %s", err)
		}
		return writeOutput(t, outputFile)
	}
	t := map[string]any{
		"properties": properties,
		"required":   required,
	}
	return writeOutput(t, outputFile)
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

func convertVariable(tfVar tfconfig.Variable) models.BackstageParamFields {
	tfType := cleanString(tfVar.Type)
	t := mapType(tfType)
	if isPrimitive(tfType) {
		b := models.BackstageParamFields{
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
	return models.BackstageParamFields{}
}

func convertArray(tfVar tfconfig.Variable) models.BackstageParamFields {
	tfType := cleanString(tfVar.Type)
	nestedType := getNestedType(tfType)
	nestedTfVar := tfconfig.Variable{
		Name: fmt.Sprintf("%s-a", tfVar.Name),
		Type: nestedType,
	}
	nestedItems := convertVariable(nestedTfVar)
	out := models.BackstageParamFields{
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

func convertObjectDefaults(tfVar tfconfig.Variable) map[string]*models.BackstageParamFields {
	// build default values by taking default's key and type. Must be done for primitives only.
	properties := make(map[string]*models.BackstageParamFields)
	nestedType := getNestedType(cleanString(tfVar.Type))
	if tfVar.Default != nil {
		d, ok := tfVar.Default.(map[string]any)
		if !ok {
			log.Fatalf("could not determine default type of %s\n", tfVar.Default)
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

func convertObject(tfVar tfconfig.Variable) models.BackstageParamFields {
	out := models.BackstageParamFields{
		Title:       tfVar.Name,
		Type:        mapType(cleanString(tfVar.Type)),
		Description: tfVar.Description,
	}

	nestedType := getNestedType(cleanString(tfVar.Type))
	if isPrimitive(nestedType) {
		p := models.AdditionalProperties{Type: mapType(nestedType)}
		out.AdditionalProperties = &p
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
		out.Properties = map[string]*models.BackstageParamFields{
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
