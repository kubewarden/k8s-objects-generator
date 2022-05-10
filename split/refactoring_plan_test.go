package split

import (
	"testing"

	openapi_spec "github.com/go-openapi/spec"
)

func TestNewRefactoringPlan(t *testing.T) {
	gitRepo := "github.com/kubewarden/k8s-objects"
	swaggerVersion := "2.0"
	kubernetesVersion := "1.23"

	swagger := openapi_spec.Swagger{}
	swagger.SwaggerProps.Swagger = swaggerVersion

	paths := openapi_spec.Paths{}
	swagger.SwaggerProps.Paths = &paths

	info := openapi_spec.Info{}
	info.InfoProps.Title = "kubernetes"
	info.InfoProps.Version = kubernetesVersion
	swagger.SwaggerProps.Info = &info
	swagger.Definitions = make(openapi_spec.Definitions)

	// First definition
	labelSelectorProperties := make(map[string]openapi_spec.Schema)
	labelSelectorProperties["name"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "name desc",
			Type:        []string{"string"},
		},
	}
	swagger.Definitions["io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "LabelSelector desc",
			Properties:  labelSelectorProperties,
		},
	}

	// Another definition - from a different pacakge, plus it's an interface
	swagger.Definitions["io.k8s.core.v1.Raw"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "Raw desc",
			Type:        []string{"object"},
		},
	}

	plan, err := NewRefactoringPlan(&swagger)
	if err != nil {
		t.Errorf("Cannot create refactoring plan: %v", err)
	}

	if plan.KubernetesVersion != kubernetesVersion {
		t.Errorf("plan has wrong kubernetes version: %s", plan.KubernetesVersion)
	}
	if plan.SwaggerVersion != swaggerVersion {
		t.Errorf("plan has wrong swagger version: %s", plan.SwaggerVersion)
	}

	if !plan.Interfaces.IsInterface(gitRepo, "core/v1", "Raw") {
		t.Errorf("interface not found")
	}

	expectedPackages := []string{"apimachinery/pkg/apis/meta/v1", "core/v1"}
	for _, pkg := range expectedPackages {
		_, found := plan.Packages[pkg]
		if !found {
			t.Errorf("cannot find package %s", pkg)
		}
	}
	if len(plan.Packages) != len(expectedPackages) {
		t.Errorf("wrong number of packages found inside of the plan: %d", len(plan.Packages))
	}
}
