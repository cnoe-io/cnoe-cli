package render

import (
	"context"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type Component interface {
	ID() int64
	Dependencies() []int64
	Render(ctx context.Context) error
	Install(ctx context.Context) error
}

type Renderer interface {
	Render()
}

type DefaultRenderer struct {
	components map[int64]Component
}

func (r DefaultRenderer) Render(ctx context.Context) error {
	err := r.Validate(ctx)
	if err != nil {
		return err
	}
	sorted, err := Sort(r.components)
	if err != nil {
		return err
	}

	// install one by one.
	for i := range sorted {
		err = r.components[sorted[i].ID()].Install(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r DefaultRenderer) Validate(ctx context.Context) error {
	for _, v := range r.components {
		err := v.Render(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r DefaultRenderer) AddComponent(component Component) {
	r.components[component.ID()] = component
}

func Sort(components map[int64]Component) ([]Component, error) {
	g := simple.NewDirectedGraph()
	for _, c := range components {
		if len(c.Dependencies()) == 0 {
			g.AddNode(simple.Node(c.ID()))
		} else {
			for _, t := range c.Dependencies() {
				g.SetEdge(simple.Edge{
					F: simple.Node(c.ID()),
					T: simple.Node(t),
				})
			}
		}
	}
	nodes, err := topo.Sort(g)
	if err != nil {
		return nil, err
	}

	out := make([]Component, len(components))
	for i := range nodes {
		out[i] = components[nodes[i].ID()]
	}
	return out, nil
}
