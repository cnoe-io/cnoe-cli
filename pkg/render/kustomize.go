package render

import (
	"context"
	"fmt"
	"os"

	"github.com/cnoe-io/cnoe-cli/pkg/config"
	k8sexec "k8s.io/utils/exec"
)

type Kustomize struct {
	Id        int64
	Path      string
	Name      string
	Deps      []int64
	Env       map[string]string
	executor  k8sexec.Interface
	ExecPath  string
	manifests []byte
	version   string
}

func newKustomize(component config.Component, id int64) *Kustomize {
	return &Kustomize{
		Path: component.Path,
		Id:   id,
		Deps: make([]int64, len(component.DependsOn)),
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
	return c.Id
}

func (c *Kustomize) Dependencies() []int64 {
	return c.Deps
}

func (c *Kustomize) Render(ctx context.Context) error {
	output, err := c.execKustomizeInstall(ctx)
	if err != nil {
		return err
	}
	c.manifests = output
	return nil
}

func (c *Kustomize) Install(ctx context.Context) error {
	if c.manifests == nil || len(c.manifests) == 0 {
		return fmt.Errorf("package was not rendered. path: %s, package: %s", c.Path, c.Name)
	}
	return nil
}

func (c *Kustomize) execKustomizeInstall(ctx context.Context) ([]byte, error) {
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
	return c.executor.CommandContext(ctx, "kustomize", append(args, c.Path)...), nil
}
