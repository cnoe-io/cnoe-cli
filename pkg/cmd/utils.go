package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cnoe-io/cnoe-cli/pkg/models"
	"github.com/itchyny/gojq"
	yamlv3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/yaml"
)

type NotFoundError struct {
	Err error
}

func (n NotFoundError) Error() string {
	return n.Error()
}

type insertAtInput struct {
	templatePath     string
	jqPathExpression string
	properties       map[string]models.BackstageParamFields
	required         []string
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
	sb.WriteString(fmt.Sprintf("%s.properties = %s.properties * %s", input.jqPathExpression, input.jqPathExpression, string(jqProp)))
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
		log.Fatalln(err)
	}
	iter := query.RunWithContext(ctx, targetTemplate)
	v, _ := iter.Next()
	if err, ok := v.(error); ok {
		log.Fatalln(err)
	}
	return v, nil
}