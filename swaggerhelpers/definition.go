package swaggerhelpers

import (
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	openapi_spec "github.com/go-openapi/spec"
	"github.com/pkg/errors"

	"github.com/kubewarden/k8s-objects-generator/common"
)

// Definition is wrapper around a Swagger Definition.
type Definition struct {
	// Original definition
	SwaggerDefinition openapi_spec.Schema
	// Name of the package where the object declared by this Definition is going
	// to be located **after** the split is done
	PackageName string

	// Name of the type that will identify the object declared by this Definition
	// **after** the split is done
	TypeName string

	// list of package names this definition depends on
	// For example, if a definition has a `meta` property of type
	// `apimachinery/pkg/apis/meta/v1/ObjectMeta`, then this definition depends
	// on `apimachinery/pkg/apis/meta/v1/`
	dependencies mapset.Set[string]
}

func NewDefinition(definition openapi_spec.Schema, id string) (*Definition, error) {
	// Some definitions need special tuning to work properly
	patchDefinition(&definition, id)

	path := strings.TrimPrefix(id, "io.k8s.")
	chunks := strings.Split(path, ".")
	if len(chunks) < common.ChunkNumber {
		return nil,
			fmt.Errorf("cannot build definition refactoring plan: wrong number of chunks: %v", chunks)
	}

	packageName := strings.Join(chunks[0:len(chunks)-1], "/")
	typeName := chunks[len(chunks)-1]
	plan := Definition{
		SwaggerDefinition: definition,
		PackageName:       packageName,
		TypeName:          typeName,
		dependencies:      mapset.NewSet[string](),
	}

	if err := plan.computeDependencies(); err != nil {
		return nil, errors.Wrapf(err, "Cannot compute dependencies of package %s", id)
	}

	return &plan, nil
}

func patchDefinition(definition *openapi_spec.Schema, id string) {
	// Ensure `Time` objects consumed by structs are accessed via pointers. This ensures they can be
	// completely omitted when they are not set.
	// This fixes https://github.com/kubewarden/kubewarden-controller/issues/570
	if id == "io.k8s.apimachinery.pkg.apis.meta.v1.Time" {
		definition.AddExtension("x-nullable", true)
	}
}

//nolint:gocognit // keep cognitive complexity as it is, this function is quite contained
func (d *Definition) computeDependencies() error {
	var propImports []PropertyImport

	for name, property := range d.SwaggerDefinition.Properties {
		propImport, err := NewPropertyImportFromRef(&property.Ref)
		if err != nil {
			return errors.Wrapf(err,
				"cannot parse ref pointer for property %s inside of %s/%s",
				name, d.PackageName, d.TypeName)
		}
		if !propImport.IsEmpty() {
			propImports = append(propImports, propImport)
		}

		if property.Items != nil && property.Items.Schema != nil {
			propImport, err := NewPropertyImportFromRef(&property.Items.Schema.Ref)
			if err != nil {
				return errors.Wrapf(err,
					"cannot parse ref pointer for property item %s inside of %s/%s",
					name, d.PackageName, d.TypeName)
			}
			if !propImport.IsEmpty() {
				propImports = append(propImports, propImport)
			}
		}

		if property.AdditionalProperties != nil {
			propImport, err := NewPropertyImportFromRef(&property.AdditionalProperties.Schema.Ref)
			if err != nil {
				return errors.Wrapf(err,
					"cannot parse ref pointer for additional property %s inside of %s/%s",
					name, d.PackageName, d.TypeName)
			}
			if !propImport.IsEmpty() {
				propImports = append(propImports, propImport)
			}
		}
	}

	for _, propImport := range propImports {
		if propImport.PackageName != d.PackageName {
			d.dependencies.Add(propImport.PackageName)
		}
	}

	return nil
}

func (d *Definition) GeneratePatchedOpenAPIDef(gitRepo string, interfaces *InterfaceRegistry) (openapi_spec.Schema, error) {
	definition := d.SwaggerDefinition

	if interfaces.IsInterface(gitRepo, d.PackageName, d.TypeName) {
		// This is an interface, we have to generate not an `{}interface` but
		// a `json.RawMessage`. Interfaces cannot be handled neither by TinyGo,
		// nor by json. We can use instead a `json.RawMessage` which doesn't
		// cause panics at runtime.
		outerObj := make(map[string]interface{})

		importObj := make(map[string]string)
		importObj["package"] = "encoding/json"

		outerObj["import"] = importObj
		outerObj["type"] = "RawMessage"

		definition.AddExtension("x-go-type", outerObj)
		return definition, nil
	}

	required := mapset.NewSet[string]()
	for _, r := range d.SwaggerDefinition.Required {
		required.Add(r)
	}

	for name := range definition.Properties {
		property := definition.Properties[name]
		isRequired := required.Contains(name)

		if err := patchSchemaRef(&property, d.PackageName, interfaces, isRequired, gitRepo); err != nil {
			return openapi_spec.Schema{}, err
		}

		if property.Items != nil && property.Items.Schema != nil {
			if err := patchSchemaRef(property.Items.Schema, d.PackageName, interfaces, isRequired, gitRepo); err != nil {
				return openapi_spec.Schema{}, err
			}
		}

		if property.AdditionalProperties != nil {
			if err := patchSchemaRef(property.AdditionalProperties.Schema, d.PackageName, interfaces, isRequired, gitRepo); err != nil {
				return openapi_spec.Schema{}, err
			}
		}

		definition.Properties[name] = property
	}

	return definition, nil
}

// patchSchemaRef changes the Ref value of the provided schema object to replace all
// references with x-go-import statements.
func patchSchemaRef(schema *openapi_spec.Schema,
	definitionPackage string,
	interfaces *InterfaceRegistry,
	isRequired bool,
	gitRepo string,
) error {
	propImport, err := NewPropertyImportFromRef(&schema.Ref)
	if err != nil {
		return err
	}

	// handle non-required attributes
	isInterface := false
	if !propImport.IsEmpty() {
		isInterface = interfaces.IsInterface(gitRepo, propImport.PackageName, propImport.TypeName)
	}

	if !isRequired {
		// non-required properties have `json:"omitempty"` set
		schema.AddExtension("x-omitempty", true)

		isBasicType := true
		if schema.Ref.GetPointer() != nil && !schema.SchemaProps.Ref.GetPointer().IsEmpty() {
			isBasicType = false
		}

		if !isBasicType && !isInterface {
			// in addition to that, non-required objects must be set to nullable
			// so that the generated code will reference them by pointer
			schema.AddExtension("x-nullable", true)
		}
	}

	if propImport.IsEmpty() {
		return nil
	}

	if propImport.PackageName == definitionPackage {
		// A definition from the same namespace is being referenced, we have to update
		// the ref to link to the new ID of the resource
		newRef, err := openapi_spec.NewRef(fmt.Sprintf("#/definitions/%s", propImport.TypeName))
		if err != nil {
			return err
		}
		schema.Ref = newRef
	} else {
		// A definition from another namespace is being referenced, we have to rewrite that
		// as a Go import

		schema.Ref = openapi_spec.Ref{}

		// This is an interface, we have to set the `x-nullable` extension to not
		// have invalid Go code being generated by Swagger
		if isInterface {
			schema.AddExtension("x-nullable", false)

			// The type being refereced is an interface. The type is not generated by
			// swagger because we are replacing it with `json.RawMessage`. This type
			// is defined inside of another package, hence we cannot rely on swagger
			// to automatically change the object type to be `json.RawMessage`,
			// we have to handle that on our own.

			outerObj := make(map[string]interface{})

			importObj := make(map[string]string)
			importObj["package"] = "encoding/json"

			outerObj["import"] = importObj
			outerObj["type"] = "RawMessage"

			schema.AddExtension("x-go-type", outerObj)
		} else {
			schema.AddExtension("x-go-type", propImport.ToMap(gitRepo))
		}
	}

	return nil
}
