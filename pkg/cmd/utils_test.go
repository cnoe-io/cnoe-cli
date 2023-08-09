package cmd

import (
	"context"
	"log"
	"testing"

	"github.com/cnoe-io/cnoe-cli/pkg/models"
)

func Test(t *testing.T) {
	input := insertAtInput{
		templatePath:     "fakes/template/input-template.yaml",
		jqPathExpression: ".spec.parameters[0]",
		properties: map[string]models.BackstageParamFields{
			"test": {
				Type:  "string",
				Title: "hello",
			},
		},
		required: []string{"my"},
	}

	_, err := insertAt(context.Background(), input)
	if err != nil {
		log.Fatal(err)
	}
}
