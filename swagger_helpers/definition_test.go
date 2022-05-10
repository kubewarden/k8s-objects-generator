package swagger_helpers

import (
	"fmt"
	"testing"

	openapi_spec "github.com/go-openapi/spec"
)

func TestParseDefinitionID(t *testing.T) {
	cases := []struct {
		id                  string
		expectedPackageName string
		expectedTypeName    string
	}{
		{
			id:                  "io.k8s.api.admissionregistration.v1.MutatingWebhook",
			expectedPackageName: "api/admissionregistration/v1",
			expectedTypeName:    "MutatingWebhook",
		},
		{
			id:                  "io.k8s.api.apiserverinternal.v1alpha1.StorageVersionCondition",
			expectedPackageName: "api/apiserverinternal/v1alpha1",
			expectedTypeName:    "StorageVersionCondition",
		},
	}

	emptySchema := openapi_spec.Schema{}

	for _, testCase := range cases {
		definition, err := NewDefinition(emptySchema, testCase.id)
		if err != nil {
			t.Errorf("unexpected error while parsing %s: %v", testCase.id, err)
		}

		if testCase.expectedPackageName != definition.PackageName {
			t.Errorf("expected package name to be %s, got %s instead",
				testCase.expectedPackageName,
				definition.PackageName)
		}

		if testCase.expectedTypeName != definition.TypeName {
			t.Errorf("expected type name to be %s, got %s instead",
				testCase.expectedTypeName,
				definition.TypeName)
		}
	}
}

func TestComputeDependenciesSimpleSchema(t *testing.T) {
	cases := []struct {
		refs         []string
		expectedDeps []string
	}{
		{
			refs:         []string{"#/definitions/io.k8s.api.admissionregistration.v1.MutatingWebhookSpec"},
			expectedDeps: []string{},
		},
		{
			refs:         []string{"#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector"},
			expectedDeps: []string{"apimachinery/pkg/apis/meta/v1"},
		},
		{
			refs: []string{
				"#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
				"#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
			},
			expectedDeps: []string{"apimachinery/pkg/apis/meta/v1"},
		},
		{
			refs: []string{
				"#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
				"#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
				"#/definitions/io.k8s.core.v1.PodSpec",
			},
			expectedDeps: []string{
				"apimachinery/pkg/apis/meta/v1",
				"core/v1",
			},
		},
	}

	for _, testCase := range cases {
		properties := make(map[string]openapi_spec.Schema)
		for counter, refUrl := range testCase.refs {
			ref, err := openapi_spec.NewRef(refUrl)
			if err != nil {
				t.Errorf("Cannot create ref: %v", err)
			}

			properties[fmt.Sprintf("prop%d", counter)] = openapi_spec.Schema{
				SchemaProps: openapi_spec.SchemaProps{
					Description: fmt.Sprintf("prop %d desc", counter),
					Ref:         ref,
				},
			}
		}

		defSchema := openapi_spec.Schema{
			SchemaProps: openapi_spec.SchemaProps{
				Properties: properties,
			},
		}

		definition, err := NewDefinition(defSchema,
			"io.k8s.api.admissionregistration.v1.MutatingWebhook",
		)
		if err != nil {
			t.Errorf("cannot generate definition: %v", err)
		}

		if len(testCase.expectedDeps) != definition.dependencies.Cardinality() {
			t.Errorf("wrong number of dependencies, expected %d got %d",
				len(testCase.expectedDeps),
				definition.dependencies.Cardinality())
		}

		for _, dep := range testCase.expectedDeps {
			if !definition.dependencies.Contains(dep) {
				t.Errorf("cannot find expected dependency %s inside of %v",
					dep, definition.dependencies)
			}
		}
	}
}

func TestComputeDependenciesSchemaWithAdditionalProperties(t *testing.T) {
	cases := []struct {
		refUrl       string
		expectedDeps []string
	}{
		{
			refUrl:       "#/definitions/io.k8s.api.admissionregistration.v1.MutatingWebhookSpec",
			expectedDeps: []string{},
		},
		{
			refUrl:       "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
			expectedDeps: []string{"apimachinery/pkg/apis/meta/v1"},
		},
	}

	for _, testCase := range cases {
		ref, err := openapi_spec.NewRef(testCase.refUrl)
		if err != nil {
			t.Errorf("Cannot create ref: %v", err)
		}

		// A more comples schema, one property that has additional properties with refs
		properties := make(map[string]openapi_spec.Schema)
		properties["default"] = openapi_spec.Schema{
			SchemaProps: openapi_spec.SchemaProps{
				Description: "desc",
				Type:        []string{"object"},
				AdditionalProperties: &openapi_spec.SchemaOrBool{
					Schema: &openapi_spec.Schema{
						SchemaProps: openapi_spec.SchemaProps{
							Ref: ref,
						},
					},
				},
			},
		}

		// A simple schema, just one property with its own ref
		defSchema := openapi_spec.Schema{
			SchemaProps: openapi_spec.SchemaProps{
				Properties: properties,
			},
		}

		definition, err := NewDefinition(defSchema,
			"io.k8s.api.admissionregistration.v1.MutatingWebhook",
		)
		if err != nil {
			t.Errorf("cannot generate definition: %v", err)
		}

		if len(testCase.expectedDeps) != definition.dependencies.Cardinality() {
			t.Errorf("wrong number of dependencies, expected %d got %d",
				len(testCase.expectedDeps),
				definition.dependencies.Cardinality())
		}

		for _, dep := range testCase.expectedDeps {
			if !definition.dependencies.Contains(dep) {
				t.Errorf("cannot find expected dependency %s inside of %v",
					dep, definition.dependencies)
			}
		}
	}
}

func TestComputeDependenciesSchemaWithItems(t *testing.T) {
	cases := []struct {
		refUrl       string
		expectedDeps []string
	}{
		{
			refUrl:       "#/definitions/io.k8s.api.admissionregistration.v1.MutatingWebhookSpec",
			expectedDeps: []string{},
		},
		{
			refUrl:       "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
			expectedDeps: []string{"apimachinery/pkg/apis/meta/v1"},
		},
	}

	for _, testCase := range cases {
		ref, err := openapi_spec.NewRef(testCase.refUrl)
		if err != nil {
			t.Errorf("Cannot create ref: %v", err)
		}

		// A more comples schema, one property that has additional properties with refs
		properties := make(map[string]openapi_spec.Schema)
		properties["default"] = openapi_spec.Schema{
			SchemaProps: openapi_spec.SchemaProps{
				Description: "desc",
				Type:        []string{"object"},
				Items: &openapi_spec.SchemaOrArray{
					Schema: &openapi_spec.Schema{
						SchemaProps: openapi_spec.SchemaProps{
							Ref: ref,
						},
					},
				},
			},
		}

		// A simple schema, just one property with its own ref
		defSchema := openapi_spec.Schema{
			SchemaProps: openapi_spec.SchemaProps{
				Properties: properties,
			},
		}

		definition, err := NewDefinition(defSchema,
			"io.k8s.api.admissionregistration.v1.MutatingWebhook",
		)
		if err != nil {
			t.Errorf("cannot generate definition: %v", err)
		}

		if len(testCase.expectedDeps) != definition.dependencies.Cardinality() {
			t.Errorf("wrong number of dependencies, expected %d got %d",
				len(testCase.expectedDeps),
				definition.dependencies.Cardinality())
		}

		for _, dep := range testCase.expectedDeps {
			if !definition.dependencies.Contains(dep) {
				t.Errorf("cannot find expected dependency %s inside of %v",
					dep, definition.dependencies)
			}
		}
	}
}

func TestPatchSchema(t *testing.T) {
	interfaces := NewInterfaceRegistry()
	properties := make(map[string]openapi_spec.Schema)

	refOtherPackage, err := openapi_spec.NewRef(
		"#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector")
	if err != nil {
		t.Errorf("Cannot create ref: %v", err)
	}

	properties["outside"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "outside desc",
			Ref:         refOtherPackage,
		},
	}

	refSamePackage, err := openapi_spec.NewRef(
		"#/definitions/io.k8s.api.admissionregistration.v1.MutatingWebhookSpec")
	if err != nil {
		t.Errorf("Cannot create ref: %v", err)
	}
	properties["same"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "same desc",
			Ref:         refSamePackage,
		},
	}

	refInterfaceFromOtherPackage, err := openapi_spec.NewRef(
		"#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.Raw")
	if err != nil {
		t.Errorf("Cannot create ref: %v", err)
	}
	properties["interface"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "interface desc",
			Ref:         refInterfaceFromOtherPackage,
		},
	}
	interfaces.RegisterInterface("apimachinery/pkg/apis/meta/v1", "Raw")

	defSchema := openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Properties: properties,
		},
	}

	definition, err := NewDefinition(defSchema,
		"io.k8s.api.admissionregistration.v1.MutatingWebhook",
	)
	if err != nil {
		t.Errorf("cannot generate definition: %v", err)
	}

	patchedSchema, err := definition.GeneratePatchedOpenAPIDef(
		"github.com/kubewarden/k8s-objects",
		&interfaces)
	if err != nil {
		t.Errorf("cannot generate patched schema: %v", err)
	}

	patchedProperties := patchedSchema.Properties

	// outside property should not be a ref anymore, it should
	// instead be a x-go-type
	outsideProp, found := patchedProperties["outside"]
	if !found {
		t.Errorf("cannot find outside prop")
	}
	extensions := outsideProp.VendorExtensible.Extensions
	_, found = extensions["x-go-type"]
	if !found {
		t.Errorf("cannot find x-go-type for outside prop")
	}
	if !outsideProp.SchemaProps.Ref.GetPointer().IsEmpty() {
		t.Errorf("outside prop should not have a Ref anymore")
	}

	// Ref inside of the same package should still be a ref, but
	// pointing to the new internal address
	internalProp, found := patchedProperties["same"]
	if !found {
		t.Errorf("cannot find internal prop")
	}
	extensions = internalProp.VendorExtensible.Extensions
	_, found = extensions["x-go-type"]
	if found {
		t.Errorf("internal prop should NOT have x-go-type")
	}
	internalRef := internalProp.SchemaProps.Ref.GetPointer()
	if internalRef.IsEmpty() {
		t.Errorf("internal prop should not have a Ref")
	}
	if internalRef.String() != "/definitions/MutatingWebhookSpec" {
		t.Errorf("internal prop Ref is pointing to the wrong location: %s", internalRef.String())
	}

	// Ref to an external Interface type should be replaced with
	// a x-go-type and a x-nullable
	// The Ref should also be empty
	interfaceProp, found := patchedProperties["interface"]
	if !found {
		t.Errorf("cannot find interface prop")
	}
	extensions = interfaceProp.VendorExtensible.Extensions
	_, found = extensions["x-go-type"]
	if !found {
		t.Errorf("cannot find x-go-type for interface prop")
	}
	nullable, found := extensions["x-nullable"]
	if !found {
		t.Errorf("cannot find x-nullable for interface prop")
	}
	if nullable.(bool) != false {
		t.Errorf("interface prop, x-nullable is not set to `false`: %v", nullable)
	}
	if !interfaceProp.SchemaProps.Ref.GetPointer().IsEmpty() {
		t.Errorf("interface prop should not have a Ref anymore")
	}
}
