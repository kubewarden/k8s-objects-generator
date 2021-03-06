package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
)

type SwaggerData struct {
	Data              []byte
	KubernetesVersion string
}

// Downloads the swagger file for the Kubernetes version specified by the user
func DownloadSwagger(kubeVersion string) (*SwaggerData, error) {
	version, err := semver.ParseTolerant(kubeVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse kubernetes version %s", kubeVersion)
	}

	downloadUrl := fmt.Sprintf(
		"https://github.com/kubernetes/kubernetes/raw/v%d.%d.%d/api/openapi-spec/swagger.json",
		version.Major, version.Minor, version.Patch)

	log.Printf("Downloading swagger file for Kubernetes %s from %s", version.String(), downloadUrl)

	resp, err := http.Get(downloadUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot fetch swagger file from %s", downloadUrl)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read contents of response from %s", downloadUrl)
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("Response failed with status code: %d and body: %s", resp.StatusCode, string(body))
	}

	return &SwaggerData{
		Data:              body,
		KubernetesVersion: version.String(),
	}, nil
}
