package swaggerhelpers

import (
	"fmt"
	"log/slog"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
)

// InterfaceRegistry keeps track of all the `interface` objects that are
// defined inside of the project.
type InterfaceRegistry struct {
	interfacesByModule map[string]mapset.Set[string]
}

func NewInterfaceRegistry() InterfaceRegistry {
	return InterfaceRegistry{
		interfacesByModule: make(map[string]mapset.Set[string]),
	}
}

// RegisterInterface keeps track of the interfaced called `name`, defined
// inside of the `module` module.
func (r *InterfaceRegistry) RegisterInterface(module, name string) {
	interfaces, known := r.interfacesByModule[module]
	if !known {
		interfaces = mapset.NewSet[string]()
	}
	interfaces.Add(name)

	r.interfacesByModule[module] = interfaces
}

// IsInterface returns true if this is a known interface, false otherwise.
// * `gitRepo`: repository that will contain the generated code: i.e. `github.com/kubewarden/k8s-objects`.
// * `module`: name of the module where the type is defined.
// * `name`: name of the type.
func (r *InterfaceRegistry) IsInterface(gitRepo, module, name string) bool {
	module = strings.TrimPrefix(module, fmt.Sprintf("%s/", gitRepo))
	interfaces, known := r.interfacesByModule[module]

	if !known {
		return false
	}

	return interfaces.Contains(name)
}

func (r *InterfaceRegistry) Dump() {
	for module, interfaces := range r.interfacesByModule {
		slog.Info("interfaces for module", "module", module, "interfaces", interfaces)
	}
}
