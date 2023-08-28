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

	for i := range c.Definitions {
		path := c.Definitions[i]
		log.Printf("processing module at %s", path)
		mod, diag := tfconfig.LoadModule(path)
		if diag.HasErrors() {
			return out, diag.Err()
		}

		if len(mod.Variables) == 0 {
			log.Printf("module %s does not have variables", path)
			continue
		}

		params, required := prepareParamsAndRequired(mod)

		fileName := fmt.Sprintf("%s.yaml", filepath.Base(path))
		outputFile := filepath.Join(c.ExpectedOutDir, fileName)
		err := handleModuleOutput(ctx, outputFile, c.TemplateFile, c.InsertionPoint, params, required, c.Collapsed, c.Raw)
		if err != nil {
			return ProcessOutput{}, err
		}

		out.Templates = append(out.Templates, fileName)
	}

	return out, nil
}

func prepareParamsAndRequired(mod *tfconfig.Module) (map[string]models.BackstageParamFields, []string) {
	params := make(map[string]models.BackstageParamFields)
	required := make([]string, 0)
	for j := range mod.Variables {
		params[j] = convertVariable(*mod.Variables[j])
		if mod.Variables[j].Required {
			required = append(required, j)
		}
	}
	return params, required
}

func handleModuleOutput(ctx context.Context, outputFile, templateFile, insertionPoint string, properties map[string]models.BackstageParamFields, required []string, collapsed, raw bool) error {
	props := make(map[string]interface{}, len(properties))
	for k := range properties {
		props[k] = properties[k]
	}

	if !raw && !collapsed {
		input := insertAtInput{
			templatePath:     templateFile,
			jqPathExpression: insertionPoint,
			fields: map[string]interface{}{
				"properties": props,
			},
		}
		if len(required) > 0 {
			input.required = required
		}
		t, err := insertAt(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to insert to given template: %s", err)
		}
		return writeOutput(t, outputFile)
	}

	t := map[string]interface{}{
		"properties": props,
		"required":   required,
	}
	return writeOutput(t, outputFile)
}

func handleOutput(ctx context.Context, outputFile, templateFile, insertionPoint string, properties map[string]models.BackstageParamFields, required []string, collapsed, raw bool) error {
	props := make(map[string]any, len(properties))
	for k := range properties {
		props[k] = properties[k]
	}
	if !raw && !collapsed {
		input := insertAtInput{
			templatePath:     templateFile,
			jqPathExpression: insertionPoint,
			fields: map[string]any{
				"properties": props,
			},
		}
		if len(required) > 0 {
			input.required = required
		}
		t, err := insertAt(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to insert to given template: %s", err)
		}
		return writeOutput(t, outputFile)
	}
	t := map[string]any{
		"properties": props,
		"required":   required,
	}
	return writeOutput(t, outputFile)
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
