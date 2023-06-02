package render

import (
	"context"
	"fmt"
	"os"

	"github.com/cnoe-io/cnoe-cli/pkg/config"
	k8sexec "k8s.io/utils/exec"
)

type Kustomize struct {
	Path     string
	Deps     []int64
	Env      map[string]string
	executor k8sexec.Interface
	ExecPath string
	version  string
	name     string
	id       int64
}

func newKustomize(component config.Component, id int64) *Kustomize {
	return &Kustomize{
		name:     component.Name,
		Path:     component.Path,
		id:       id,
		Deps:     make([]int64, len(component.DependsOn)),
		executor: k8sexec.New(),
	}
}

// generating int64 representations of strings is impossible. Will need to look into rolling our own topological sort
// this is bad.
func NewKustomizeComponents(config config.Configuration) (map[int64]Component, error) {
	components := make(map[int64]Component, len(config.Spec.Distro.Components))
	tmpMap := make(map[string]int64, len(config.Spec.Distro.Components))

	for id, v := range config.Spec.Distro.Components {
		tmpMap[v.Name] = int64(id) // could be ""
	}

	for i, v := range config.Spec.Distro.Components {
		id := int64(i)
		kComponent := newKustomize(v, id)
		for j := range v.DependsOn {
			depId := tmpMap[v.DependsOn[j].Name] // could be non-existent
			kComponent.Deps[j] = depId
		}
		components[id] = kComponent
	}
	return components, nil
}

func (c *Kustomize) ID() int64 {
	return c.id
}

func (c *Kustomize) Name() string {
	return c.name
}

func (c *Kustomize) Dependencies() []int64 {
	return c.Deps
}

func (c *Kustomize) Render(ctx context.Context) error {
	output, err := c.execKustomizeBuild(ctx)
	if err != nil {
		return fmt.Errorf("%w : %s", err, output)
	}
	return os.WriteFile(manifestPath(c), output, 0640)
}

func (c *Kustomize) Install(ctx context.Context) error {
	return nil
}

func (c *Kustomize) execKustomizeBuild(ctx context.Context) ([]byte, error) {
	kustomizeCmd, err := c.setup(ctx, "build")
	if err != nil {
		return nil, err
	}
	return kustomizeCmd.CombinedOutput() // blocks
}

func (c *Kustomize) setup(ctx context.Context, args ...string) (k8sexec.Cmd, error) {
	if c.ExecPath != "" {
		err := os.Setenv("PATH", c.ExecPath)
		if err != nil {
			return nil, err
		}
	}
	return c.executor.CommandContext(context.Background(), "kustomize", append(args, c.Path)...), nil
}
