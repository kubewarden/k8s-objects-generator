package split

import (
	_ "embed"
	"path/filepath"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/event_gvk.go.gold
var eventGvkGold string

//go:embed testdata/group_info.go.gold
var groupInfoGold string

func TestKubernetesExtensionParse(t *testing.T) {
	tests := []struct {
		extensionJSON   string
		extensionParsed bool
		expectedGVK     *groupVersionResource
	}{
		{
			extensionJSON: `{"x-kubernetes-group-version-kind": [
        						{
          							"group": "events.k8s.io",
									"kind": "Event",
          							"version": "v1"
        						}
							]}`,
			extensionParsed: true,
			expectedGVK: &groupVersionResource{
				Group:   "events.k8s.io",
				Version: "v1",
				Kind:    "Event",
			},
		},
		{
			extensionJSON: `{"x-kubernetes-group-version-kind": [
        						{
          							"group": "",
          							"kind": "DeleteOptions",
          							"version": "v1"
        						},
        						{
          							"group": "admission.k8s.io",
          							"kind": "DeleteOptions",
          							"version": "v1"
        						}
							]}`,
			extensionParsed: false,
		},
	}

	for _, tt := range tests {
		extension := spec.VendorExtensible{}
		require.NoError(t, extension.UnmarshalJSON([]byte(tt.extensionJSON)))
		kubeExtension, isKubeExtension := asKubernetesExtension(extension.Extensions)
		assert.Equal(t, isKubeExtension, tt.extensionParsed)
		if tt.extensionParsed {
			assert.Equal(t, kubeExtension[kubernetesGroupKey], tt.expectedGVK.Group)
			assert.Equal(t, kubeExtension[kubernetesVersionKey], tt.expectedGVK.Version)
			assert.Equal(t, kubeExtension[kubernetesKindKey], tt.expectedGVK.Kind)
		}
	}
}

func TestGenerateGroupResources(t *testing.T) {
	outputDir := "/testout"
	project, err := NewProject(outputDir, "", "")
	require.NoError(t, err)

	splitter, err := NewSplitter(filepath.Join("testdata", "test-swagger.json"))
	require.NoError(t, err)

	refactoringPlan, err := splitter.ComputeRefactoringPlan()
	require.NoError(t, err)

	fs := afero.NewMemMapFs()
	groupResource := NewGroupResource(fs)
	require.NoError(t, groupResource.Generate(project, refactoringPlan))

	eventGvk, err := afero.ReadFile(fs, filepath.Join(outputDir, "src/api/events/v1/event_gvk.go"))
	require.NoError(t, err)
	groupInfo, err := afero.ReadFile(fs, filepath.Join(outputDir, "src/api/events/v1/group_info.go"))
	require.NoError(t, err)

	assert.Equal(t, eventGvkGold, string(eventGvk))
	assert.Equal(t, groupInfoGold, string(groupInfo))
}
