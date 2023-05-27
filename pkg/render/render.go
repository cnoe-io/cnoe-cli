package render

import (
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func NewComponent(id int64) *Component {
	return &Component{
		id: id,
	}
}

type Component struct {
	id            int64
	path          string
	name          string
	componentType string
	neighbors     map[int64]int64
}

func (c Component) ID() int64 {
	return c.id
}

func Sort(components map[int64]Component) ([]Component, error) {
	g := simple.NewDirectedGraph()
	for _, c := range components {
		if len(c.neighbors) == 0 {
			g.AddNode(simple.Node(c.id))
		} else {
			for f, t := range c.neighbors {
				g.SetEdge(simple.Edge{
					F: simple.Node(f),
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
