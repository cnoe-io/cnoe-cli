package config

type Configuration struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

type Spec struct {
	Target     Target       `yaml:"target"`
	Distro     Distro       `yaml:"distro"`
	Blueprints []Blueprints `yaml:"blueprints"`
	Auth       Auth         `yaml:"auth"`
}

type Blueprints struct {
	Path    string `yaml:"path"`
	Version string `yaml:"version"`
	Type    string `yaml:"type"`
	Name    string `yaml:"name"`
}

type OidcRef struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type Oidc struct {
	ClientId     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
	Callback     string `yaml:"callback"`
}

type Auth struct {
	Enabled bool    `yaml:"enabled"`
	Oidc    Oidc    `yaml:"oidc"`
	OidcRef OidcRef `yaml:"oidcRef"`
	Domain  Domain  `yaml:"domain"`
}

type Domain struct {
	PortalBase string `yaml:"portalBase"`
}

type Target struct {
	Kubernetes Kubernetes `yaml:"kubernetes"`
}

type Kubernetes struct {
	Context string `yaml:"context"`
}

type Distro struct {
	Version    string   `yaml:"version"`
	Components []string `yaml:"components"`
}
