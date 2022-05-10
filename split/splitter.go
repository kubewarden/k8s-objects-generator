package split

import (
	"fmt"
	"os"
	"path/filepath"

	openapi_spec "github.com/go-openapi/spec"
	"github.com/pkg/errors"
)

// Takes care of splitting the single big swagger file of Kubernetes
// into smaller ones, one package
type Splitter struct {
	vanillaSwagger openapi_spec.Swagger
}

func NewSplitter(swaggerFile string) (Splitter, error) {
	data, err := os.ReadFile(swaggerFile)
	if err != nil {
		return Splitter{}, errors.Wrapf(err, "cannot read swagger file %s", swaggerFile)
	}

	swagger := openapi_spec.Swagger{}
	if err = swagger.UnmarshalJSON(data); err != nil {
		return Splitter{}, errors.Wrapf(err, "cannot decode swagger file %s", swaggerFile)
	}

	return Splitter{
		vanillaSwagger: swagger,
	}, nil
}

func (s *Splitter) ComputeRefactoringPlan() (*RefactoringPlan, error) {
	return NewRefactoringPlan(&s.vanillaSwagger)
}

type walkerStateSwaggerData struct {
	swaggerFiles map[string]string
	project      Project
}

func (s *Splitter) GenerateSwaggerFiles(project Project, plan *RefactoringPlan) error {
	swaggerFiles, err := plan.RenderNewSwaggerFiles(project.GitRepo)
	if err != nil {
		return err
	}

	dependenciesGraph, err := plan.DependenciesGraph()
	if err != nil {
		return err
	}

	stateData := walkerStateSwaggerData{
		project:      project,
		swaggerFiles: swaggerFiles,
	}
	state := NewGeneratorState(dependenciesGraph, stateData)

	if err := WalkGraph(&state, swaggerGenerateModelsVisitorFn); err != nil {
		return errors.Wrapf(err, "cannot generate swagger files")
	}

	return nil
}

func swaggerGenerateModelsVisitorFn(nodeID string, state *GeneratorState) error {
	if state.VisitedNodes.Contains(nodeID) {
		return nil
	}

	// Fist, ensure all the dependencies are generated
	ancestors, err := state.DependenciesGraph.GetOrderedAncestors(nodeID)
	if err != nil {
		return errors.Wrapf(err, "cannot compute dependency graph of package %s", nodeID)
	}

	fmt.Printf("Ordered list of dependencies of packge %s: %+v\n", nodeID, ancestors)
	for _, ancestor := range ancestors {
		if err := swaggerGenerateModelsVisitorFn(ancestor, state); err != nil {
			return err
		}
	}

	// Now let's generate the actual namespace
	fmt.Printf("Generating models for package %s\n", nodeID)

	stateData := state.Data.(walkerStateSwaggerData)

	jsonData, found := stateData.swaggerFiles[nodeID]
	if !found {
		return fmt.Errorf("Cannot find %s inside of list of patched swagger files", nodeID)
	}

	// write file
	pathToSwagger := filepath.Join(stateData.project.OutputDir,
		"src",
		stateData.project.GitRepo,
		nodeID)
	if err := os.MkdirAll(pathToSwagger, 0777); err != nil {
		return errors.Wrapf(err, "cannot create directory %s", pathToSwagger)
	}

	fileName := filepath.Join(pathToSwagger, "swagger.json")
	if err := os.WriteFile(fileName, []byte(jsonData), 0644); err != nil {
		return errors.Wrapf(err, "cannot write %s", fileName)
	}

	if err := stateData.project.InvokeSwaggerModelGenerator(nodeID); err != nil {
		return fmt.Errorf("swagger execution failed for module %s: %+v", nodeID, err)
	}
	state.VisitedNodes.Add(nodeID)

	return nil
}
