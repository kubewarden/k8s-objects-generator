package swagger_helpers

import (
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set"
	openapi_spec "github.com/go-openapi/spec"
	"github.com/pkg/errors"
)

// Wrapper around a Swagger Definition
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
	dependencies mapset.Set
}

func NewDefinition(definition openapi_spec.Schema, id string) (*Definition, error) {
	path := strings.TrimPrefix(id, "io.k8s.")
	chunks := strings.Split(path, ".")
	if len(chunks) < 2 {
		return nil,
			fmt.Errorf("cannot build definition refactoring plan: wrong number of chunks: %v", chunks)
	}

	packageName := strings.Join(chunks[0:len(chunks)-1], "/")
	typeName := chunks[len(chunks)-1]
	plan := Definition{
		SwaggerDefinition: definition,
		PackageName:       packageName,
		TypeName:          typeName,
		dependencies:      mapset.NewSet(),
	}

	if err := plan.computeDependencies(); err != nil {
		return nil, errors.Wrapf(err, "Cannot compute dependencies of package %s", id)
	}

	return &plan, nil
}

func (d *Definition) computeDependencies() error {
	propImports := []PropertyImport{}

	for name, property := range d.SwaggerDefinition.Properties {
		propImport, err := NewPropertyImportFromRef(&property.SchemaProps.Ref)
		if err != nil {
			return errors.Wrapf(err,
				"cannot parse ref pointer for property %s inside of %s/%s",
				name, d.PackageName, d.TypeName)
		}
		if !propImport.IsEmpty() {
			propImports = append(propImports, propImport)
		}

		if property.Items != nil && property.Items.Schema != nil {
			propImport, err := NewPropertyImportFromRef(&property.Items.Schema.SchemaProps.Ref)
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
			propImport, err := NewPropertyImportFromRef(&property.AdditionalProperties.Schema.SchemaProps.Ref)
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
		// a `easyjson.RawMessage`. Interfaces cannot be handled neither by TinyGo,
		// nor by easyjson. We can use instead a `easyjson.RawMessage` which doesn't
		// cause panics at runtime.
		outerObj := make(map[string]interface{})

		importObj := make(map[string]string)
		importObj["package"] = "github.com/mailru/easyjson"

		outerObj["import"] = importObj
		outerObj["type"] = "RawMessage"

		definition.VendorExtensible.AddExtension("x-go-type", outerObj)
		return definition, nil
	}

	for name := range definition.Properties {
		property := definition.Properties[name]

		if err := patchSchemaRef(&property, d.PackageName, interfaces, gitRepo); err != nil {
			return openapi_spec.Schema{}, err
		}

		if property.Items != nil && property.Items.Schema != nil {
			if err := patchSchemaRef(property.Items.Schema, d.PackageName, interfaces, gitRepo); err != nil {
				return openapi_spec.Schema{}, err
			}
		}

		if property.AdditionalProperties != nil {
			if err := patchSchemaRef(property.AdditionalProperties.Schema, d.PackageName, interfaces, gitRepo); err != nil {
				return openapi_spec.Schema{}, err
			}
		}

		definition.Properties[name] = property
	}

	return definition, nil
}

// Changes the Ref value of the provided schema object to replace all
// references with x-go-import statements
func patchSchemaRef(schema *openapi_spec.Schema,
	definitionPackage string,
	interfaces *InterfaceRegistry,
	gitRepo string,
) error {
	propImport, err := NewPropertyImportFromRef(&schema.SchemaProps.Ref)
	if err != nil {
		return err
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
		schema.SchemaProps.Ref = newRef
	} else {
		// A definition from another namespace is being referenced, we have to rewrite that
		// as a Go import

		schema.SchemaProps.Ref = openapi_spec.Ref{}

		// This is an interface, we have to set the `x-nullable` extension to not
		// have invalid Go code being generated by Swagger
		if interfaces.IsInterface(gitRepo, propImport.PackageName, propImport.TypeName) {
			schema.VendorExtensible.AddExtension("x-nullable", false)

			// The type being refereced is an interface. The type is not generated by
			// swagger becase we are replacing it with `easyjson.RawMessage`. This type
			// is defined inside of another package, hence we cannot rely on swagger
			// to automatically change the object type to be `easyjson.RawMessage`,
			// we have to handle that on our own.

			outerObj := make(map[string]interface{})

			importObj := make(map[string]string)
			importObj["package"] = "github.com/mailru/easyjson"

			outerObj["import"] = importObj
			outerObj["type"] = "RawMessage"

			schema.VendorExtensible.AddExtension("x-go-type", outerObj)
		} else {
			schema.VendorExtensible.AddExtension("x-go-type", propImport.ToMap(gitRepo))
		}
	}
	return nil
}
