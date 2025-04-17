package main

import (
	_ "embed"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/kubewarden/k8s-objects-generator/split"
)

//go:embed LICENSE
var LICENSE string

func main() {
	var swaggerFile, kubeVersion, outputDir, gitRepo string
	var swaggerData *SwaggerData
	var err error

	flag.StringVar(&swaggerFile, "f", "", "The swagger file to process")
	flag.StringVar(&kubeVersion, "kube-version", "", "Fetch the swagger file of the specified Kubernetes version")
	flag.StringVar(&outputDir, "o", "./k8s-objects", "The root directory where the files will be generated")
	flag.StringVar(&gitRepo, "repo", "github.com/kubewarden/k8s-objects", "The repository where the generated files are going to be published")

	flag.Parse()

	if swaggerFile != "" && kubeVersion != "" {
		log.Fatal("`-f` and `-kube-version` flags cannot be used at the same time")
	}

	if len(swaggerFile) == 0 && len(kubeVersion) == 0 {
		log.Fatal("one of the `-f` or `-kube-version` flag must be specified")
	}

	if kubeVersion != "" {
		swaggerData, err = DownloadSwagger(kubeVersion)
		if err != nil {
			log.Fatal(err)
		}
	}
	if swaggerFile != "" {
		data, err := os.ReadFile(swaggerFile)
		if err != nil {
			log.Fatalf("cannot read swagger file %s: %v", swaggerFile, err)
		}
		swaggerData = &SwaggerData{
			Data:              data,
			KubernetesVersion: "unknown",
		}
	}

	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		log.Panic(err)
	}

	templatesTmpDir, err := os.MkdirTemp("", "k8s-objects-generator-swagger-templates")
	if err != nil {
		log.Panic(err)
	}

	if err = writeTemplates(templatesTmpDir); err != nil {
		log.Panic(err)
	}
	log.Printf("Created templates at %s", templatesTmpDir)

	defer func() {
		if err := os.RemoveAll(templatesTmpDir); err != nil {
			log.Printf("error removing the temporary directory that holds swagger templates '%s': %v",
				templatesTmpDir, err)
		}
	}()

	project, err := split.NewProject(
		outputDir,
		gitRepo,
		filepath.Join(templatesTmpDir, "swagger_templates"),
	)
	if err != nil {
		log.Panic(err)
	}

	log.Print("Initializing target directory")
	err = project.Init(swaggerData.Data, swaggerData.KubernetesVersion, LICENSE)
	if err != nil {
		log.Panic(err)
	}

	splitter, err := split.NewSplitter(project.SwaggerFile())
	if err != nil {
		log.Panic(err)
	}

	refactoringPlan, err := splitter.ComputeRefactoringPlan()
	if err != nil {
		log.Panic(err)
	}

	if err := splitter.GenerateSwaggerFiles(project, refactoringPlan); err != nil {
		log.Panic(err)
	}

	groupResource := split.NewGroupResource(afero.NewOsFs())
	if err := groupResource.Generate(project, refactoringPlan); err != nil {
		log.Panic(err)
	}

	if err := project.RunGoModTidy(); err != nil {
		log.Panic(errors.Wrap(err, "error running go mod tidy"))
	}
}
