package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
)

type SwaggerData struct {
	Data              []byte
	KubernetesVersion string
}

// DownloadSwagger downloads the swagger file for the Kubernetes version
// specified by the user.
func DownloadSwagger(kubeVersion string) (*SwaggerData, error) {
	version, err := semver.ParseTolerant(kubeVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse kubernetes version %s", kubeVersion)
	}

	downloadURL := fmt.Sprintf(
		"https://github.com/kubernetes/kubernetes/raw/v%d.%d.%d/api/openapi-spec/swagger.json",
		version.Major, version.Minor, version.Patch)

	slog.Info("Downloading swagger file for Kubernetes", "version", version.String(), "downloadURL", downloadURL)

	resp, err := http.Get(downloadURL) //nolint:gosec,noctx // let's keep the code simple, we just do 1 request..
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot fetch swagger file from %s", downloadURL)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Info("failed to close response body", "error", cerr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read contents of response from %s", downloadURL)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("response failed with status code: %d and body: %s", resp.StatusCode, string(body))
	}

	return &SwaggerData{
		Data:              body,
		KubernetesVersion: version.String(),
	}, nil
}
