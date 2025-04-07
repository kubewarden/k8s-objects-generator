package swaggerhelpers

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

	properties["required_prop"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "required prop desc",
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

	properties["array_of_strings"] = openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Description: "a list of strings",
			Type:        []string{"array"},
			Items: &openapi_spec.SchemaOrArray{
				Schema: &openapi_spec.Schema{
					SchemaProps: openapi_spec.SchemaProps{
						Type: []string{"string"},
					},
				},
			},
		},
	}

	defSchema := openapi_spec.Schema{
		SchemaProps: openapi_spec.SchemaProps{
			Properties: properties,
			Required:   []string{"required_prop"},
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

	cases := []struct {
		PropName       string
		XGoTypeIsSet   bool
		RefIsNull      bool
		NewRef         string
		IsNullableSet  bool
		IsOmitEmptySet bool
	}{
		{
			PropName:       "outside",
			XGoTypeIsSet:   true,
			RefIsNull:      true,
			NewRef:         "",
			IsNullableSet:  true,
			IsOmitEmptySet: true,
		},
		{
			PropName:       "same",
			XGoTypeIsSet:   false,
			RefIsNull:      false,
			NewRef:         "/definitions/MutatingWebhookSpec",
			IsNullableSet:  true,
			IsOmitEmptySet: true,
		},
		{
			PropName:       "interface",
			XGoTypeIsSet:   true,
			RefIsNull:      true,
			NewRef:         "",
			IsNullableSet:  false,
			IsOmitEmptySet: true,
		},
		{
			PropName:       "array_of_strings",
			XGoTypeIsSet:   false,
			RefIsNull:      true,
			NewRef:         "",
			IsNullableSet:  false,
			IsOmitEmptySet: true,
		},
		{
			PropName:       "required_prop",
			XGoTypeIsSet:   false,
			RefIsNull:      false,
			NewRef:         "/definitions/MutatingWebhookSpec",
			IsNullableSet:  false,
			IsOmitEmptySet: false,
		},
	}

	for _, testCase := range cases {
		prop, found := patchedProperties[testCase.PropName]
		if !found {
			t.Errorf("cannot find %s property", testCase.PropName)
			continue
		}
		extensions := prop.Extensions
		_, found = extensions["x-go-type"]
		if testCase.XGoTypeIsSet && !found {
			t.Errorf("x-go-type is not set for %s property", testCase.PropName)
		}
		if !testCase.XGoTypeIsSet && found {
			t.Errorf("x-go-type is was not supposed to be set for %s property", testCase.PropName)
		}

		ref := prop.Ref.GetPointer()
		refIsNull := ref.IsEmpty()
		if testCase.RefIsNull && !refIsNull {
			t.Errorf("%s property should not have a ref set anymore", testCase.PropName)
		}
		if !testCase.RefIsNull && refIsNull {
			t.Errorf("%s property should have a ref set", testCase.PropName)
		}

		if testCase.NewRef != "" {
			if ref.String() != testCase.NewRef {
				t.Errorf("%s property: Ref is pointing to the wrong location: %s instead of %s",
					testCase.PropName,
					ref.String(),
					testCase.NewRef,
				)
			}
		}

		checkBoolExtension(
			t,
			testCase.PropName,
			extensions,
			"x-nullable",
			testCase.IsNullableSet)

		checkBoolExtension(
			t,
			testCase.PropName,
			extensions,
			"x-omitempty",
			testCase.IsOmitEmptySet)
	}
}

func checkBoolExtension(t *testing.T, propName string, extensions openapi_spec.Extensions, extensionName string, expectedValue bool) {
	value, found := extensions[extensionName]
	if !found && expectedValue == false {
		return
	}
	if !found {
		t.Errorf("property %s does not have %s extension set",
			propName, extensionName)
		return
	}
	if value != expectedValue {
		t.Errorf("property %s: %s extension is not %v as expected",
			propName, extensionName, expectedValue)
	}
}
