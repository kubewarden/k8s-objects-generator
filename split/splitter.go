package split

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	openapi_spec "github.com/go-openapi/spec"
	"github.com/pkg/errors"
)

// Splitter takes care of splitting the single big swagger file of Kubernetes
// into smaller ones, one package.
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

func (s *Splitter) GenerateSwaggerFiles(project Project, plan *RefactoringPlan) error {
	swaggerFiles, err := plan.RenderNewSwaggerFiles(project.GitRepo)
	if err != nil {
		return err
	}

	for pkgName, jsonData := range swaggerFiles {
		slog.Info("Generating models for package", "package", pkgName)

		pathToSwagger := filepath.Join(project.OutputDir,
			"src",
			project.GitRepo,
			pkgName)
		if err := os.MkdirAll(pathToSwagger, 0o750); err != nil {
			return errors.Wrapf(err, "cannot create directory %s", pathToSwagger)
		}

		fileName := filepath.Join(pathToSwagger, "swagger.json")
		if err := os.WriteFile(fileName, []byte(jsonData), 0o600); err != nil {
			return errors.Wrapf(err, "cannot write %s", fileName)
		}

		if err := project.InvokeSwaggerModelGenerator(pkgName); err != nil {
			return fmt.Errorf("swagger execution failed for module %s: %+w", pkgName, err)
		}
	}

	return nil
}
