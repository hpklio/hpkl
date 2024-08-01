package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"hpkl.io/hpkl/pkg/pklutils"
	"hpkl.io/hpkl/pkg/registry"
)

type (
	ResolverType int

	Checksums struct {
		Sha256 string `json:"sha256"`
	}

	Dependency struct {
		Uri            string     `json:"uri"`
		Checksums      *Checksums `json:"checksums"`
		Name           string     `json:"name"`
		ProjectFileUri string     `json:"project_file_uri"`
	}

	Metadata struct {
		Name                string                `json:"name"`
		PackageUri          string                `json:"packageUri"`
		Version             string                `json:"version"`
		PackageZipUrl       string                `json:"packageZipUrl"`
		PackageZipChecksums Checksums             `json:"packageZipChecksums"`
		Authors             []string              `json:"authors"`
		Dependencies        map[string]Dependency `json:"dependencies"`
		ResolverType        ResolverType          `json:"-"`
		PlainHttp           bool                  `json:"-"`
	}

	Resolver struct {
		ociResolver  *OciResolver
		httpResolver *HttpResolver
		basePath     string
		cache        map[string]*Metadata
		config       *AppConfig
	}

	DependencyResolver interface {
		ResolveMetadata(uri string, plainHttp bool) (*Metadata, error)
		ResolveArchive(metadata *Metadata) ([]byte, error)
	}

	OciResolver struct {
		client      *registry.Client
		plainClient *registry.Client
		config      *AppConfig
	}

	HttpResolver struct {
		config    *AppConfig
		plainHttp bool
	}

	ResolvedDependency struct {
		DependencyType string            `json:"type"`
		Uri            string            `json:"uri"`
		Checksums      map[string]string `json:"checksums"`
	}

	ProjectDependencies struct {
		SchemaVersion        int                            `json:"schemaVersion"`
		ResolvedDependencies map[string]*ResolvedDependency `json:"resolvedDependencies"`
	}
)

const (
	OCI ResolverType = iota
	HTTP
)

func NewResolver(appConfig *AppConfig) (*Resolver, error) {
	oci, err := NewOciResolver(appConfig)

	if err != nil {
		return nil, err
	}

	http := NewHttpResolver(appConfig)

	if err != nil {
		return nil, err
	}

	return &Resolver{
		ociResolver:  oci,
		httpResolver: http,
		basePath:     path.Join(appConfig.CacheDir, "package-2"),
		config:       appConfig,
		cache:        make(map[string]*Metadata),
	}, nil
}

func (r *Resolver) Resolve(dependencies map[string]Dependency) (map[string]*Metadata, error) {
	logger := r.config.Logger
	result := make(map[string]*Metadata)

	for _, dependency := range dependencies {
		metadata, ok := r.cache[dependency.Uri]
		dependencyName := dependency.Name
		if !ok {
			var resolver DependencyResolver

			if strings.HasSuffix(dependencyName, ".oci") {
				logger.Info("Resolving: %s as %+v proto: oci", dependencyName, dependency)
				resolver = r.ociResolver
			} else {
				logger.Info("Resolving: %s as %+v proto: http", dependencyName, dependency)
				resolver = r.httpResolver
			}

			plain := strings.Contains(dependencyName, ".plain")

			metadata, err := resolver.ResolveMetadata(dependency.Uri, plain)

			if err != nil {
				logger.Error("Metadata resolving error: %s - %+v", dependencyName, dependency)
				return nil, err
			}

			for metadataName, metadataDep := range metadata.Dependencies {
				metadataDep.Name = metadataName
				metadata.Dependencies[metadataName] = metadataDep
			}

			for metadataName, metadataDep := range metadata.Dependencies {
				metadataDep.Name = metadataName
				metadata.Dependencies[metadataName] = metadataDep
			}

			r.cache[dependency.Uri] = metadata
			result[dependency.Uri] = metadata

			if len(metadata.Dependencies) > 0 {
				subs, err := r.Resolve(metadata.Dependencies)

				if err != nil {
					return nil, err
				}

				for u, d := range subs {
					result[u] = d
				}
			}
		} else {
			result[dependency.Uri] = metadata
		}
	}
	return result, nil
}

func (r *Resolver) Exists(metadata *Metadata) (bool, error) {
	baseUri, err := url.Parse(metadata.PackageUri)

	if err != nil {
		return false, err
	}

	basePath := pklutils.PklGetRelativePath(r.basePath, baseUri)

	if _, err := os.Stat(basePath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return true, nil
	}

}

func (r *Resolver) Download(dependencies map[string]*Metadata) error {

	logger := r.config.Logger

	for u, m := range dependencies {
		e, err := r.Exists(m)

		if err != nil {
			return err
		}

		if !e {
			var resolver DependencyResolver

			if m.ResolverType == OCI {
				logger.Info("Downloading %s as %+v proto: oci", u, m)
				resolver = r.ociResolver
			} else {
				logger.Info("Downloading %s as %+v proto: http", u, m)
				resolver = r.httpResolver
			}

			bytes, err := resolver.ResolveArchive(m)

			if err != nil {
				return err
			}

			baseUri, err := url.Parse(u)

			if err != nil {
				return err
			}

			basePath := pklutils.PklGetRelativePath(r.basePath, baseUri)
			err = os.MkdirAll(basePath, os.ModePerm)

			if err != nil {
				return err
			}

			metaPath := path.Join(basePath, fmt.Sprintf("%s@%s.json", m.Name, m.Version))
			archivePath := path.Join(basePath, fmt.Sprintf("%s@%s.zip", m.Name, m.Version))

			metadataBytes, err := json.Marshal(m)

			if err != nil {
				return err
			}

			err = os.WriteFile(metaPath, metadataBytes, os.ModePerm)

			if err != nil {
				return err
			}

			err = os.WriteFile(archivePath, bytes, os.ModePerm)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewOciResolver(appConfig *AppConfig) (*OciResolver, error) {
	client, err := registry.NewClient(registry.WithPlainHttp(appConfig.PlainHttp))
	plainClient, err := registry.NewClient(registry.WithPlainHttp(true))

	if err != nil {
		return nil, err
	}

	return &OciResolver{client: client, plainClient: plainClient, config: appConfig}, nil
}

func (r *OciResolver) ResolveMetadata(uri string, plainHttp bool) (*Metadata, error) {
	ref, err := pklutils.PklUriToRef(uri)

	if err != nil {
		return nil, err
	}

	client := r.client
	if plainHttp {
		client = r.plainClient
	}

	result, err := client.Pull(ref, registry.PullOptWithPackage(false))

	if err != nil {
		return nil, err
	}

	var metadata *Metadata
	if err := json.Unmarshal(result.Metadata.Data, &metadata); err != nil {
		return nil, err
	}

	metadata.ResolverType = OCI

	return metadata, nil
}

func (r *OciResolver) ResolveArchive(metadata *Metadata) ([]byte, error) {
	ref, err := pklutils.PklUriToRef(metadata.PackageUri)

	if err != nil {
		return nil, err
	}

	client := r.client
	if metadata.PlainHttp {
		client = r.plainClient
	}

	result, err := client.Pull(ref, registry.PullOptWithPackage(true))

	if err != nil {
		return nil, err
	}

	return result.Archive.Data, nil
}

func NewHttpResolver(appConfig *AppConfig) *HttpResolver {
	return &HttpResolver{plainHttp: appConfig.PlainHttp, config: appConfig}
}

func (r *HttpResolver) ResolveMetadata(uri string, plainHttp bool) (*Metadata, error) {

	u, err := url.Parse(uri)
	logger := r.config.Logger

	if err != nil {
		logger.Error("Parsing error %s", uri)
		return nil, err
	}

	if r.plainHttp || plainHttp {
		u.Scheme = "http"
	} else {
		u.Scheme = "https"
	}

	resp, err := http.Get(u.String())

	if err != nil {
		logger.Error("Http get error %s", u.String())
		return nil, err
	}

	if resp.StatusCode > 300 {
		return nil, fmt.Errorf("Http get Error status: %s", resp.Status)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	var metadata *Metadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		logger.Error("Json unmarshal error: %s", body)
		return nil, err
	}

	metadata.ResolverType = HTTP
	metadata.PlainHttp = plainHttp

	return metadata, nil
}

func (r *HttpResolver) ResolveArchive(metadata *Metadata) ([]byte, error) {
	resp, err := http.Get(metadata.PackageZipUrl)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	return body, nil
}
