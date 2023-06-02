package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cnoe-io/cnoe-cli/pkg/render"
)

type cnoeCmd struct {
	renderer render.Renderer
}

func (c cnoeCmd) install(ctx context.Context) error {
	err := setup()
	if err != nil {
		return err
	}
	err = c.renderer.Validate(ctx)
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

func setup() error {
	err := checkCreateDir(render.CNOEDir)
	if err != nil {
		return err
	}
	return checkCreateDir(fmt.Sprintf("%s/%s", render.CNOEDir, render.CNOEManifestsDir))
}

func checkCreateDir(path string) error {
	err := os.Mkdir(path, 0740)
	if !errors.Is(err, os.ErrExist) {
		return err
	}
	return nil
}
