package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/kubewarden/k8s-objects-generator/split"
)

func main() {
	var swaggerFile, outputDir, gitRepo string

	flag.StringVar(&swaggerFile, "f", "swagger.json", "The swagger file to process")
	flag.StringVar(&outputDir, "o", "./k8s-objects", "The root directory where the files will be generated")
	flag.StringVar(&gitRepo, "repo", "github.com/kubewarden/k8s-objects", "The repository where the generated files are going to be published")

	flag.Parse()

	outputDir, err := filepath.Abs(outputDir)
	if err != nil {
		log.Panic(err)
	}

	templatesTmpDir, err := os.MkdirTemp("", "k8s-objects-generator-swagger-templates")
	if err != nil {
		log.Fatal(err)
	}

	if err = writeTemplates(templatesTmpDir); err != nil {
		log.Fatal(err)
	}
	log.Printf("crated templates at %s", templatesTmpDir)

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
		log.Fatal(err)
	}

	log.Print("Initializing target directory")
	err = project.Init()
	if err != nil {
		log.Fatal(err)
	}

	splitter, err := split.NewSplitter(swaggerFile)
	if err != nil {
		log.Panic(err)
	}

	refactoringPlan, err := splitter.ComputeRefactoringPlan()
	if err != nil {
		log.Panic(err)
	}

	if err := splitter.GenerateSwaggerFiles(project, refactoringPlan); err != nil {
		log.Fatal(err)
	}

	if err := split.GenerateEasyjsonFiles(project, refactoringPlan); err != nil {
		log.Fatal(err)
	}
}
