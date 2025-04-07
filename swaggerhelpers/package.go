package swaggerhelpers

import (
	mapset "github.com/deckarep/golang-set/v2"
	openapi_spec "github.com/go-openapi/spec"
	"github.com/pkg/errors"
)

type Package struct {
	// name of the package, after the split
	Name string

	// List of definitions provided by this module
	Definitions []*Definition

	// list of package names this one depends on
	Dependencies mapset.Set[string]
}

func NewPackage(name string) Package {
	return Package{
		Name:         name,
		Definitions:  []*Definition{},
		Dependencies: mapset.NewSet[string](),
	}
}

func (p *Package) AddDefinitionRefactoringPlan(definition *Definition) {
	p.Definitions = append(p.Definitions, definition)
	p.Dependencies = p.Dependencies.Union(definition.dependencies)
}

func (p *Package) GenerateSwagger(swaggerVersion, kubernetesVersion, gitRepo string, interfaces *InterfaceRegistry) (openapi_spec.Swagger, error) {
	swagger := openapi_spec.Swagger{}
	swagger.Swagger = swaggerVersion

	paths := openapi_spec.Paths{}
	swagger.Paths = &paths

	info := openapi_spec.Info{}
	info.Title = "kubernetes"
	info.Version = kubernetesVersion
	swagger.Info = &info
	swagger.Definitions = make(openapi_spec.Definitions)

	for _, def := range p.Definitions {
		patchedDefinition, err := def.GeneratePatchedOpenAPIDef(
			gitRepo,
			interfaces,
		)
		if err != nil {
			return openapi_spec.Swagger{},
				errors.Wrapf(err, "cannot patch definition %s/%s", def.PackageName, def.TypeName)
		}
		swagger.Definitions[def.TypeName] = patchedDefinition
	}

	return swagger, nil
}
