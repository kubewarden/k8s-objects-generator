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
	"github.com/spf13/afero"

	"github.com/kubewarden/k8s-objects-generator/object_templates"
	"github.com/kubewarden/k8s-objects-generator/swaggerhelpers"
)

const (
	kubernetesGroupVersionKindKey = "x-kubernetes-group-version-kind"
	kubernetesGroupKey            = "group"
	kubernetesVersionKey          = "version"
	kubernetesKindKey             = "kind"
)

type groupVersionResource struct {
	Group   string
	Version string
	Kind    string
}

func (g groupVersionResource) String() string {
	return fmt.Sprintf("%s/%s,Resource=%s", g.Group, g.Version, g.Kind)
}

type groupResource struct {
	fs afero.Fs
}

func NewGroupResource(fs afero.Fs) *groupResource {
	return &groupResource{
		fs: fs,
	}
}

func (g *groupResource) Generate(project Project, plan *RefactoringPlan) error {
	objectKindTemplate, err := template.New("gvk").Parse(object_templates.ObjectKindTemplate)
	if err != nil {
		return err
	}
	groupInfoTemplate, err := template.New("group-info").Parse(object_templates.GroupVersionTemplate)
	if err != nil {
		return err
	}

	var lastGVK *groupVersionResource
	var gvkCount int
	for _, pkg := range plan.Packages {
		log.Println("============================================================================")
		log.Println("Generating GVK files for module", pkg.Name)
		for _, dfn := range pkg.Definitions {
			if gvk := groupKindResource(dfn); gvk != nil {
				gvkCount++
				objectKindFilePath := filepath.Join(project.Root, dfn.PackageName, fmt.Sprintf("%s_gvk.go", strcase.ToSnake(gvk.Kind)))
				if err = g.generateResourceFile(objectKindFilePath, objectKindTemplate, gvk); err != nil {
					return err
				}
				lastGVK = gvk
			}
		}
		if lastGVK != nil {
			// Generates group_info.go file (one per GroupVersion combination)
			groupInfoFilePath := filepath.Join(project.Root, pkg.Name, "group_info.go")
			if err = g.generateResourceFile(groupInfoFilePath, groupInfoTemplate, lastGVK); err != nil {
				return err
			}

			log.Printf("Generated GVK files (visited %d/%d)", gvkCount, len(pkg.Definitions))
			log.Println("Generated group_info.go file")
			lastGVK, gvkCount = nil, 0
		}
	}

	return g.copyStaticFiles(project.Root)
}

func (g *groupResource) generateResourceFile(path string, templ *template.Template, gvk *groupVersionResource) error {
	gvkFile, err := g.fs.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer gvkFile.Close()

	if err := templ.Execute(gvkFile, gvk); err != nil {
		return fmt.Errorf("failed to process template for %s: %v", gvk.String(), err)
	}

	return nil
}

func groupKindResource(definition *swaggerhelpers.Definition) *groupVersionResource {
	extension := definition.SwaggerDefinition.Extensions
	if extension == nil || extension[kubernetesGroupVersionKindKey] == nil {
		return nil
	}

	kubeExtension, isKubeExtension := asKubernetesExtension(extension)
	if !isKubeExtension {
		log.Printf("GVK specific %s key format for %s package definition is not found. Skipping...", kubernetesGroupVersionKindKey, definition.PackageName)
		return nil
	}

	return &groupVersionResource{
		Group:   kubeExtension[kubernetesGroupKey],
		Version: kubeExtension[kubernetesVersionKey],
		Kind:    kubeExtension[kubernetesKindKey],
	}
}

func (g *groupResource) copyStaticFiles(targetRoot string) error {
	log.Println("============================================================================")
	log.Println("Generating static content files")
	err := fs.WalkDir(object_templates.ApimachineryRoot, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skipping any directory
		if d.IsDir() || path == "." {
			return nil
		}
		sourceBuf, err := object_templates.ApimachineryRoot.ReadFile(path)
		if err != nil {
			return err
		}

		if err = g.fs.MkdirAll(filepath.Join(targetRoot, filepath.Dir(path)), os.ModePerm); err != nil {
			return nil
		}
		targetFilePath := filepath.Join(targetRoot, path)
		targetFile, err := g.fs.OpenFile(targetFilePath, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			return err
		}
		log.Println("File", filepath.Base(path), "copied into the", filepath.Dir(targetFilePath))
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
