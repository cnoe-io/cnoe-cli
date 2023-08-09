package models

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Metadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

type Spec struct {
	ClaimNames *struct {
		Kind string `json:"kind"`
	} `json:"claimNames,omitempty"`
	Group string `json:"group"`
	Names struct {
		Kind string `json:"kind"`
	} `json:"names"`
	Versions []struct {
		Name   string `json:"name"`
		Schema struct {
			OpenAPIV3Schema struct {
				Properties map[string]interface{} `json:"properties"`
			} `yaml:"openAPIV3Schema"`
		} `json:"schema"`
	} `json:"versions"`
}

type Definition struct {
	Kind       string   `json:"kind"`
	APIVersion string   `json:"apiVersion"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
}

type Resources struct {
	Enum []string `json:"enum"`
}

type Config struct {
	Type       string                     `json:"type"`
	Title      string                     `json:"title"`
	Properties *unstructured.Unstructured `json:"properties,omitempty"`
}

type Props struct {
	Resources Resources `json:"resources"`
	Config    Config    `json:"config"`
}

type Wrapper struct {
	Properties Props `json:"properties"`
}

type Template struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string `yaml:"name"`
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
	} `yaml:"metadata"`
	Spec struct {
		Owner      string `yaml:"owner"`
		Type       string `yaml:"type"`
		Parameters []struct {
			Properties   map[string]interface{} `yaml:"properties"`
			Dependencies struct {
				Resources struct {
					OneOf []map[string]interface{} `yaml:"oneOf,omitempty"`
				} `yaml:"resources,omitempty"`
			} `yaml:"dependencies,omitempty"`
		} `yaml:"parameters"`

		Steps []struct {
			Id     string                 `yaml:"id"`
			Name   string                 `yaml:"name"`
			Action string                 `yaml:"action"`
			Input  map[string]interface{} `yaml:"input"`
		} `yaml:"steps"`
	} `yaml:"spec"`
}

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
