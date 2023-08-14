package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchyny/gojq"
	yamlv3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/yaml"
)

type finder func(file os.DirEntry, currentDepth uint32, base string) ([]string, error)

type NotFoundError struct {
	Err error
}

func (n NotFoundError) Error() string {
	return n.Error()
}

type insertAtInput struct {
	templatePath     string
	jqPathExpression string
	properties       supportedFields
	required         []string
}

type supportedFields struct {
	Properties   any `yaml:",omitempty"`
	Dependencies any `yaml:",omitempty"`
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

func checkAndCreateDir(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// return absolute path for given input and output dirs, if output path does not exist, create it.
func prepDirectories(inputDir, outputDir string, oneOf bool) (string, string, error) {
	input, err := filepath.Abs(inputDir)
	if err != nil {
		return "", "", err
	}
	output, err := filepath.Abs(outputDir)
	if err != nil {
		return "", "", err
	}
	m := output
	if oneOf {
		m = fmt.Sprintf("%s/resources", output)
	}
	err = checkAndCreateDir(m)
	if err != nil {
		return "", "", err
	}
	return input, output, nil
}

func handleOneOf(ctx context.Context, outputFile, templatePath, insertionPoint string, resourceFiles []string) error {
	input := insertAtInput{
		templatePath:     templatePath,
		jqPathExpression: insertionPoint,
	}
	t, err := oneOf(ctx, resourceFiles, input)
	if err != nil {
		return err
	}
	return writeOutput(t, outputFile)
}

func oneOf(ctx context.Context, resourceFiles []string, input insertAtInput) (any, error) {
	resourcesPath := fmt.Sprintf(input.templatePath, "/resources")
	err := os.MkdirAll(resourcesPath, 0755)
	if err != nil {
		return nil, err
	}
	n := make([]string, len(resourceFiles))
	m := make([]map[string]string, len(resourceFiles))
	for i := range resourceFiles {
		fileName := filepath.Base(resourceFiles[i])
		n[i] = fileName
		m[i] = map[string]string{
			"$yaml": fmt.Sprintf("resources/%s.yaml", fileName),
		}
	}
	props := map[string]any{
		"resources": map[string]any{
			"type": "string",
			"enum": n,
		},
	}
	deps := map[string]any{
		"resources": map[string][]map[string]string{
			"oneOf": m,
		},
	}
	input.properties.Properties = props
	input.properties.Dependencies = deps

	return insertAt(ctx, input)
}

func jsonFromObject(obj any) ([]byte, error) {
	b, err := yamlv3.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return yaml.YAMLToJSON(b)
}

// inserts Backstage parameters at the specified path. Path is specified in the same format as jq.
func insertAt(ctx context.Context, input insertAtInput) (any, error) {
	b, err := os.ReadFile(input.templatePath)
	if err != nil {
		return nil, err
	}
	var targetTemplate map[string]any
	err = yaml.Unmarshal(b, &targetTemplate)
	if err != nil {
		return nil, err
	}
	jqProp, err := jsonFromObject(input.properties)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	// update the properties field. merge (*) then assign (=)
	sb.WriteString(fmt.Sprintf("%s = %s * %s", input.jqPathExpression, input.jqPathExpression, string(jqProp)))
	if len(input.required) > 0 {
		jqReq, err := jsonFromObject(input.required)
		if err != nil {
			return nil, err
		}
		// update the required field by feeding the new query the output from previous step (|)
		sb.WriteString("| ")
		sb.WriteString(fmt.Sprintf("%s.required = (%s.required + %s)", input.jqPathExpression, input.jqPathExpression, string(jqReq)))
	}
	query, err := gojq.Parse(sb.String())
	if err != nil {
		return nil, err
	}
	iter := query.RunWithContext(ctx, targetTemplate)
	v, _ := iter.Next()
	if err, ok := v.(error); ok {
		return nil, err
	}
	return v, nil
}

func writeOutput(content any, path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yamlv3.NewEncoder(f)
	defer enc.Close()
	enc.SetIndent(2)
	return enc.Encode(content)
}

func getRelevantFiles(inputDir string, currentDepth uint32, f finder) ([]string, error) {
	base, err := filepath.Abs(inputDir)
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, file := range files {
		o, err := f(file, currentDepth, base)
		if err != nil {
			return nil, err
		}
		out = append(out, o...)
	}
	return out, nil
}
