package cmd

import (
	"context"

	"github.com/cnoe-io/cnoe-cli/pkg/render"
)

type cnoeCmd struct {
	renderer render.Renderer
}

func (c cnoeCmd) install(ctx context.Context) error {
	err := c.renderer.Validate(ctx)
	if err != nil {
		log.Error(err, "failed to validate given configuration")
		return err
	}
	err = c.renderer.Install(ctx)
	if err != nil {
		log.Error(err, "failed to install")
	}
	return nil
}
