package split

import (
	"fmt"

	openapi_spec "github.com/go-openapi/spec"
	"github.com/heimdalr/dag"
	"github.com/kubewarden/k8s-objects-generator/swaggerhelpers"
	"github.com/pkg/errors"
)

// Holds information about how the big swagger file is going to be split
type RefactoringPlan struct {
	Packages          map[string]swaggerhelpers.Package
	Interfaces        swaggerhelpers.InterfaceRegistry
	SwaggerVersion    string
	KubernetesVersion string
}

func NewRefactoringPlan(swagger *openapi_spec.Swagger) (*RefactoringPlan, error) {
	packages := make(map[string]swaggerhelpers.Package)
	interfaces := swaggerhelpers.NewInterfaceRegistry()

	kubernetesVersion := "undefined"
	if swagger.Info != nil {
		kubernetesVersion = swagger.Info.Version
	}

	for id, definition := range swagger.Definitions {
		newDefinitionRefactoringPlan, err := swaggerhelpers.NewDefinition(definition, id)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot parse definition with id %s", id)
		}

		if ((len(definition.Type) == 1 && definition.Type[0] == "object") || (len(definition.Type) == 0)) &&
			len(definition.Properties) == 0 &&
			definition.AdditionalProperties == nil {
			// this is a go interface
			interfaces.RegisterInterface(newDefinitionRefactoringPlan.PackageName, newDefinitionRefactoringPlan.TypeName)
		}

		pkg, pkgKnown := packages[newDefinitionRefactoringPlan.PackageName]
		if !pkgKnown {
			pkg = swaggerhelpers.NewPackage(newDefinitionRefactoringPlan.PackageName)
		}

		pkg.AddDefinitionRefactoringPlan(newDefinitionRefactoringPlan)
		packages[newDefinitionRefactoringPlan.PackageName] = pkg
	}

	return &RefactoringPlan{
		SwaggerVersion:    swagger.Swagger,
		KubernetesVersion: kubernetesVersion,
		Packages:          packages,
		Interfaces:        interfaces,
	}, nil
}

func (r *RefactoringPlan) DependenciesGraph() (*dag.DAG, error) {
	dependenciesGraph := dag.NewDAG()

	for pkgName, pkg := range r.Packages {
		if _, err := dependenciesGraph.GetVertex(pkgName); err != nil {
			// the package is not yet known by the DAG
			err := dependenciesGraph.AddVertexByID(pkgName, pkgName)
			if err != nil {
				return nil, err
			}
		}

		for name := range pkg.Dependencies.Iterator().C {
			// ensure the dependency is known by the DAG
			if _, err := dependenciesGraph.GetVertex(name); err != nil {
				_, found := r.Packages[name]
				if !found {
					return nil, fmt.Errorf("unsolved dependency: cannot find package %s inside of list of known packages", name)
				}

				// the package is not yet known by the DAG
				err := dependenciesGraph.AddVertexByID(name, name)
				if err != nil {
					return nil, err
				}
			}

			// register the dependency relation
			// this namespace depends on `depName`
			if err := dependenciesGraph.AddEdge(name, pkgName); err != nil {
				return nil, errors.Wrapf(err,
					"Cannot register the dependency relation that %s has against %s",
					pkgName, name)
			}
		}
	}

	return dependenciesGraph, nil
}

func (r *RefactoringPlan) RenderNewSwaggerFiles(githubRepo string) (map[string]string, error) {
	renderedFiles := make(map[string]string)

	for pkgName, pkg := range r.Packages {
		swaggerFile, err := pkg.GenerateSwagger(
			r.SwaggerVersion,
			r.KubernetesVersion,
			githubRepo,
			&r.Interfaces,
		)
		if err != nil {
			return make(map[string]string), errors.Wrapf(err, "cannot render swagger file for package %s", pkgName)
		}

		jsonBytes, err := swaggerFile.MarshalJSON()
		if err != nil {
			return make(map[string]string), errors.Wrapf(err, "cannot render swagger file for package %s to JSON", pkgName)
		}

		renderedFiles[pkgName] = string(jsonBytes)
	}

	return renderedFiles, nil
}
