package render

import (
	"context"
	"fmt"
	"os"

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
}

func (c *Kustomize) ID() int64 {
	return c.Id
}

func (c *Kustomize) Dependencies() []int64 {
	return c.Deps
}

func (c *Kustomize) Render(ctx context.Context) error {
	output, err := c.execKustomize(ctx)
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

func (c *Kustomize) execKustomize(ctx context.Context) ([]byte, error) {
	kustomizeCmd, err := c.setup(ctx)
	if err != nil {
		return nil, err
	}
	return kustomizeCmd.CombinedOutput() // blocks
}

func (c *Kustomize) setup(ctx context.Context) (k8sexec.Cmd, error) {
	if c.ExecPath != "" {
		err := os.Setenv("PATH", c.ExecPath)
		if err != nil {
			return nil, err
		}
	}
	return c.executor.CommandContext(ctx, "kustomize", "build", c.Path), nil
}
