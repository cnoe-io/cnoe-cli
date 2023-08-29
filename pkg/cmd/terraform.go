package cmd

import (
	"context"
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
		PreRunE: templatePreRunE,
		RunE:    tfE,
	}
)

func init() {
	templateCmd.AddCommand(tfCmd)
}

func tfE(cmd *cobra.Command, args []string) error {
	return Process(cmd.Context(), NewTerraformModule(inputDir, outputDir, templatePath, insertionPoint, collapsed, raw))
}

type TerraformModule struct {
	EntityConfig
}

func NewTerraformModule(inputDir, outputDir, templatePath, insertionPoint string, collapsed, raw bool) Entity {
	return &TerraformModule{
		EntityConfig: EntityConfig{
			InputDir:       inputDir,
			OutputDir:      outputDir,
			TemplateFile:   templatePath,
			InsertionPoint: insertionPoint,
			Collapsed:      collapsed,
			Raw:            raw,
		},
	}
}

func (t *TerraformModule) Config() EntityConfig {
	return t.EntityConfig
}

func (t *TerraformModule) HandleEntries(ctx context.Context, c EntryConfig) (ProcessOutput, error) {
	out := ProcessOutput{
		Templates: make([]string, 0),
		Resources: make([]string, 0),
	}

	for _, def := range c.Definitions {
		log.Printf("processing module at %s", def)
		mod, diag := tfconfig.LoadModule(def)
		if diag.HasErrors() {
			return out, diag.Err()
		}

		if len(mod.Variables) == 0 {
			log.Printf("module %s does not have variables", def)
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

		fileName := filepath.Join(c.ExpectedOutDir, fmt.Sprintf("%s.yaml", filepath.Base(def)))
		err := t.handleModuleOutput(ctx, fileName, c.TemplateFile, params, required)
		if err != nil {
			log.Printf("failed to write %s: %s \n", def, err.Error())
			return ProcessOutput{}, err
		}

		out.Templates = append(out.Templates, fileName)
	}

	return out, nil
}

func (t *TerraformModule) handleModuleOutput(ctx context.Context, outputFile, templateFile string, properties map[string]models.BackstageParamFields, required []string) error {
	if !t.Raw && !t.Collapsed {
		input := insertAtInput{
			templatePath:     t.TemplateFile,
			jqPathExpression: insertionPoint,
			fields: map[string]interface{}{
				"properties": properties,
			},
		}
		if len(required) > 0 {
			input.required = required
		}
		content, err := insertAt(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to insert to given template: %s", err)
		}
		return writeOutput(content, outputFile)
	}

	content := map[string]interface{}{
		"properties": properties,
		"required":   required,
	}
	return writeOutput(content, outputFile)
}

func (t *TerraformModule) GetDefinitions(inputDir string, currentDepth uint32) ([]string, error) {
	if currentDepth > depth {
		return nil, nil
	}
	if tfconfig.IsModuleDir(inputDir) {
		return []string{inputDir}, nil
	}
	out, err := getRelevantFiles(inputDir, currentDepth, t.findModule)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (t *TerraformModule) findModule(file os.DirEntry, currentDepth uint32, base string) ([]string, error) {
	f := filepath.Join(base, file.Name())
	stat, err := os.Stat(f)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		mods, err := t.GetDefinitions(f, currentDepth+1)
		if err != nil {
			return nil, err
		}
		return mods, nil
	}
	return nil, nil
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
		// we will enable this once it's fixed in Backstage.
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
