package swaggerhelpers

import (
	"fmt"
	"testing"

	openapi_spec "github.com/go-openapi/spec"
)

func TestNewPropertyImportFromRef(t *testing.T) {
	cases := []struct {
		ref                 string
		expectedPackageName string
		expectedTypeName    string
		expectedAlias       string
	}{
		{
			ref:                 "#/definitions/io.k8s.api.admissionregistration.v1.MutatingWebhook",
			expectedPackageName: "api/admissionregistration/v1",
			expectedAlias:       "api_admissionregistration_v1",
			expectedTypeName:    "MutatingWebhook",
		},
		{
			ref:                 "#/definitions/io.k8s.api.apiserverinternal.v1alpha1.StorageVersionCondition",
			expectedPackageName: "api/apiserverinternal/v1alpha1",
			expectedAlias:       "api_apiserverinternal_v1alpha1",
			expectedTypeName:    "StorageVersionCondition",
		},
		{
			ref:                 "",
			expectedPackageName: "",
			expectedAlias:       "",
			expectedTypeName:    "",
		},
	}

	for _, testCase := range cases {
		ref, err := openapi_spec.NewRef(testCase.ref)
		if err != nil {
			t.Errorf("cannot create ref from url %s: %v", testCase.ref, err)
		}

		propImport, err := NewPropertyImportFromRef(&ref)
		if err != nil {
			t.Errorf("unexpected error while parsing %s: %v", testCase.ref, err)
		}

		if testCase.expectedPackageName != propImport.PackageName {
			t.Errorf("expected package name to be %s, got %s instead",
				testCase.expectedPackageName,
				propImport.PackageName)
		}

		if testCase.expectedTypeName != propImport.TypeName {
			t.Errorf("expected type name to be %s, got %s instead",
				testCase.expectedTypeName,
				propImport.TypeName)
		}

		if testCase.expectedAlias != propImport.Alias {
			t.Errorf("expected alias to be %s, got %s instead",
				testCase.expectedAlias,
				propImport.Alias)
		}
	}
}

func TestPropertyImportToMap(t *testing.T) {
	gitRepo := "github.com/kubewarden/k8s-objects"

	cases := []struct {
		ref                   string
		expectedImportPackage string
		expectedImportAlias   string
		expectedTypeName      string
	}{
		{
			ref:                   "#/definitions/io.k8s.api.admissionregistration.v1.MutatingWebhook",
			expectedImportPackage: fmt.Sprintf("%s/api/admissionregistration/v1", gitRepo),
			expectedImportAlias:   "api_admissionregistration_v1",
			expectedTypeName:      "MutatingWebhook",
		},
		{
			ref:                   "#/definitions/io.k8s.api.apiserverinternal.v1alpha1.StorageVersionCondition",
			expectedImportPackage: fmt.Sprintf("%s/api/apiserverinternal/v1alpha1", gitRepo),
			expectedImportAlias:   "api_apiserverinternal_v1alpha1",
			expectedTypeName:      "StorageVersionCondition",
		},
	}

	for _, testCase := range cases {
		ref, err := openapi_spec.NewRef(testCase.ref)
		if err != nil {
			t.Errorf("cannot create ref from url %s: %v", testCase.ref, err)
		}

		propImport, err := NewPropertyImportFromRef(&ref)
		if err != nil {
			t.Errorf("unexpected error while parsing %s: %v", testCase.ref, err)
		}
		vendorMap := propImport.ToMap(gitRepo)

		if vendorMap["type"] != testCase.expectedTypeName {
			t.Errorf("expected type to be %s, got %s instead",
				testCase.expectedTypeName,
				vendorMap["type"])
		}

		importObj, found := vendorMap["import"]
		if !found {
			t.Errorf("Cannot find `import`")
		}
		importMap := importObj.(map[string]string)
		if importMap["package"] != testCase.expectedImportPackage {
			t.Errorf("expected import package name to be %s, got %s instead",
				testCase.expectedImportPackage,
				importMap["package"])
		}
		if importMap["alias"] != testCase.expectedImportAlias {
			t.Errorf("expected import alias to be %s, got %s instead",
				testCase.expectedImportAlias,
				importMap["alias"])
		}
	}
}
