package swaggerhelpers

import (
	"fmt"
	"path/filepath"
	"strings"

	openapi_spec "github.com/go-openapi/spec"
)

type PropertyImport struct {
	PackageName string
	Alias       string
	TypeName    string
}

func (p *PropertyImport) IsEmpty() bool {
	return p.PackageName == "" && p.Alias == "" && p.TypeName == ""
}

// Convert PropertyImport into a swagger x-go-type interface
// * `gitRepo`: name of the repository that is going to host the code, e.g. `github.com/kubewarden/k8s-objects`
func (p *PropertyImport) ToMap(gitRepo string) map[string]interface{} {
	outerObj := make(map[string]interface{})

	fullPackageImport := filepath.Join(gitRepo, p.PackageName)

	importObj := make(map[string]string)
	importObj["package"] = fullPackageImport
	importObj["alias"] = p.Alias

	outerObj["import"] = importObj
	outerObj["type"] = p.TypeName

	return outerObj
}

// Given a `ref` string like:
// `/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector`
// return:
//
//	propertyImport"{
//	   package_name: "apimachinery/pkg/apis/meta/v1",
//	   alias: "apimachinery_pkgs_apis_meta_v1",
//	   type_name: "LabelSelector",
//	}
func NewPropertyImportFromRef(ref *openapi_spec.Ref) (PropertyImport, error) {
	refPointer := ref.GetPointer()
	if refPointer == nil || refPointer.IsEmpty() {
		return PropertyImport{}, nil
	}

	namespace := strings.TrimPrefix(refPointer.String(), "/definitions/io.k8s.")
	chunks := strings.Split(namespace, ".")
	if len(chunks) < 2 {
		return PropertyImport{},
			fmt.Errorf("ref -> chunk: not enough chunks for %s: %+v", ref, chunks)
	}

	namespaceChunks := chunks[0 : len(chunks)-1]
	typeName := chunks[len(chunks)-1]

	alias := strings.Join(namespaceChunks, "_")
	alias = strings.ReplaceAll(alias, "-", "")

	return PropertyImport{
		TypeName:    typeName,
		Alias:       alias,
		PackageName: strings.Join(namespaceChunks, "/"),
	}, nil
}
