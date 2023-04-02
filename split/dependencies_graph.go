package split

import (
	"fmt"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/heimdalr/dag"
)

// Helper function that walks a graph and invokes the provided `visitorFn`
// against each node.
// Nodes are visited starting from their anchestors, the given function is
// invoked only once.
func WalkGraph(state *GeneratorState, visitorFn VisitNodeFn) error {
	vertices := state.DependenciesGraph.GetVertices()
	verticesCount := len(vertices)

	verticesIds := []string{}
	for id := range vertices {
		verticesIds = append(verticesIds, id)
	}
	sort.Strings(verticesIds)

	for _, pkgName := range verticesIds {
		msg := fmt.Sprintf("Processing entry %s (visited %d/%d)",
			pkgName, state.VisitedNodes.Cardinality(), verticesCount)
		fmt.Println(msg)
		separator := strings.Repeat("=", len(msg))
		fmt.Println(separator)

		if err := visitorFn(pkgName, state); err != nil {
			return err
		}
	}

	return nil
}

// used to keep track of the navigation inside of the graph
type GeneratorState struct {
	VisitedNodes      mapset.Set[string]
	DependenciesGraph *dag.DAG
	Data              interface{}
}

func NewGeneratorState(dependencies *dag.DAG, data interface{}) GeneratorState {
	return GeneratorState{
		VisitedNodes:      mapset.NewSet[string](),
		DependenciesGraph: dependencies,
		Data:              data,
	}
}

type VisitNodeFn func(nodeID string, state *GeneratorState) error
