package lib

type CRD struct {
	Group   string `yaml:"group"`
	Kind    string `yaml:"kind"`
	Version string `yaml:"version"`
}

type Pod struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type Operator struct {
	Name string `yaml:"name"`
	Crds []CRD  `yaml:"crds"`
	Pods []Pod  `yaml:"pods"`
}

type Config struct {
	Prerequisits []Operator `yaml:"prerequisits"`
}
