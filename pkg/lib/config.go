package lib

type CRD struct {
	Group   string `yaml:"group"`
	Kind    string `yaml:"kind"`
	Version string `yaml:"version"`
}

type Pod struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	State     string `yaml:"state"`
}

type Spec struct {
	Crds []CRD `yaml:"crds"`
	Pods []Pod `yaml:"pods"`
}

type Config struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata",required`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string            `yaml:"name",required`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}
