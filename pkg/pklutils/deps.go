package pklutils

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
)

type ResolvedDependency struct {
	DependencyType string            `json:"type"`
	Uri            string            `json:"uri"`
	Checksums      map[string]string `json:"checksums"`
}

type ProjectDeps struct {
	SchemaVersion        int                            `json:"schemaVersion"`
	ResolvedDependencies map[string]*ResolvedDependency `json:"resolvedDependencies"`
}

func PklWriteDeps(workingDir string, deps *ProjectDeps) error {
	depsData, err := json.MarshalIndent(deps, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(workingDir, "PklProject.deps.json"), depsData, os.ModePerm)
	if err != nil {
		return err
	}

	return err
}

func PklGetRelativePath(cacheDir string, baseUri *url.URL, version string) string {
	return path.Join(
		cacheDir,
		baseUri.Host,
		baseUri.Path,
	)
}

func PklBaseUriToRef(uri string, version string) (string, error) {
	baseUri, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%s:%s", baseUri.Host, baseUri.Path, version), nil
}

func PklUriToRef(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	s := strings.Split(u.Path, "@")
	return fmt.Sprintf("%s%s:%s", u.Host, s[0], s[1]), nil
}
