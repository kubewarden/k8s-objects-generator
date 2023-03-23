package split

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/go-openapi/spec"
	"github.com/iancoleman/strcase"

	"github.com/kubewarden/k8s-objects-generator/object_templates"
	"github.com/kubewarden/k8s-objects-generator/swagger_helpers"
)

const (
	kubernetesGroupVersionKindKey = "x-kubernetes-group-version-kind"
)

type groupVersionResource struct {
	Group   string
	Version string
	Kind    string
}

func GenerateGroupResources(project Project, plan *RefactoringPlan) error {
	objectKindTemplate, err := template.New("objectKind").Parse(object_templates.ObjectKindTemplate)
	if err != nil {
		return err
	}
	groupVersionTemplate, err := template.New("groupVersion").Parse(object_templates.GroupVersionTemplate)
	if err != nil {
		return err
	}

	var lastGVK *groupVersionResource
	var gvkCount int
	for _, pkg := range plan.Packages {
		log.Println("============================================================================")
		log.Println("Generating GVK resources for module", pkg.Name)
		for _, dfn := range pkg.Definitions {
			gvk, err := groupKindResource(dfn)
			if err != nil {
				return err
			}

			if gvk != nil {
				gvkCount++
				objectKindFilePath := filepath.Join(project.Root, dfn.PackageName, fmt.Sprintf("%s_gvk.go", strcase.ToSnake(gvk.Kind)))
				if err = generateResourceFile(objectKindFilePath, objectKindTemplate, gvk); err != nil {
					return err
				}
				lastGVK = gvk
			}
		}
		if lastGVK != nil {
			groupVersionFilePath := filepath.Join(project.Root, pkg.Name, "group_version.go")
			if err = generateResourceFile(groupVersionFilePath, groupVersionTemplate, lastGVK); err != nil {
				return err
			}

			log.Printf("Generated GVK resources for module %s (visited %d/%d)", pkg.Name, gvkCount, len(pkg.Definitions))
			log.Println("Generated GV resource for module", pkg.Name)
			lastGVK, gvkCount = nil, 0
		}
	}

	return copyStaticFiles(project.Root)
}

func generateResourceFile(path string, templ *template.Template, gvk *groupVersionResource) error {
	gvkFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer gvkFile.Close()

	return templ.Execute(gvkFile, gvk)
}

func groupKindResource(definition *swagger_helpers.Definition) (*groupVersionResource, error) {
	extension := definition.SwaggerDefinition.Extensions
	if extension == nil || extension[kubernetesGroupVersionKindKey] == nil {
		return nil, nil
	}

	kubeExtension, isKubeExtension := asKubernetesExtension(extension)
	if !isKubeExtension {
		log.Printf("GVK specific %s key format for %s package definition is not found. Skipping...", kubernetesGroupVersionKindKey, definition.PackageName)
		return nil, nil
	}

	return &groupVersionResource{
		Group:   kubeExtension["group"],
		Version: kubeExtension["version"],
		Kind:    kubeExtension["kind"],
	}, nil
}

func copyStaticFiles(targetRoot string) error {
	log.Println("============================================================================")
	log.Println("Generating static content files")
	err := fs.WalkDir(object_templates.ApimachineryRoot, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skipping any directory
		if d.IsDir() {
			return nil
		}
		sourceBuf, err := object_templates.ApimachineryRoot.ReadFile(path)
		if err != nil {
			return err
		}

		if err = os.MkdirAll(filepath.Join(targetRoot, filepath.Dir(path)), os.ModePerm); err != nil {
			return nil
		}
		targetFilePath := filepath.Join(targetRoot, path)
		targetFile, err := os.OpenFile(targetFilePath, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		log.Println("File", targetFilePath, "copied")
		defer targetFile.Close()
		if _, err = targetFile.Write(sourceBuf); err != nil {
			return err
		}

		return nil
	})
	log.Println("============================================================================")

	return err
}

func asKubernetesExtension(e spec.Extensions) (map[string]string, bool) {
	if v, ok := e[strings.ToLower(kubernetesGroupVersionKindKey)]; ok {
		slice, isInterfaceSlice := v.([]interface{})
		if !isInterfaceSlice || len(slice) != 1 {
			return nil, false
		}
		if interfaceMap, isMap := slice[0].(map[string]interface{}); isMap {
			extMap := make(map[string]string, len(interfaceMap))
			for key, value := range interfaceMap {
				stringValue, isStringValue := value.(string)
				if !isStringValue {
					return nil, false
				}
				extMap[key] = stringValue
			}
			return extMap, true
		}
	}
	return nil, false
}
