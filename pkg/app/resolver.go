package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"go.uber.org/zap"
	"hpkl.io/hpkl/pkg/pklutils"
	"hpkl.io/hpkl/pkg/registry"
)

type (
	ResolverType int

	Checksums struct {
		Sha256 string `json:"sha256"`
	}

	Dependency struct {
		PackageUri string     `json:"packageUri"`
		Checksums  *Checksums `json:"checksums"`
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
	}

	Resolver struct {
		ociResolver  *OciResolver
		httpResolver *HttpResolver
		basePath     string
		cache        map[string]*Metadata
		appConfig    *AppConfig
	}

	DependencyResolver interface {
		ResolveMetadata(uri string) (*Metadata, error)
		ResolveArchive(metadata *Metadata) ([]byte, error)
	}

	OciResolver struct {
		client *registry.Client
		logger *zap.Logger
	}

	HttpResolver struct {
		plainHttp bool
		logger    *zap.Logger
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

func NewResolver(plainHttp bool, appConfig *AppConfig) (*Resolver, error) {
	oci, err := NewOciResolver(plainHttp, appConfig)

	if err != nil {
		return nil, err
	}

	http := NewHttpResolver(plainHttp, appConfig)

	if err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()

	if err != nil {
		return nil, err
	}

	return &Resolver{
		ociResolver:  oci,
		httpResolver: http,
		basePath:     path.Join(homeDir, ".pkl/cache/package-1"),
		appConfig:    appConfig,
		cache:        make(map[string]*Metadata),
	}, nil

}

func (r *Resolver) Resolve(dependencies map[string]Dependency) (map[string]*Metadata, error) {
	result := make(map[string]*Metadata)

	logger := r.appConfig.Logger.Sugar()

	for dependencyName, dependency := range dependencies {
		metadata, ok := r.cache[dependency.PackageUri]
		if !ok {
			var resolver DependencyResolver

			if strings.HasSuffix(dependencyName, ".oci") {
				logger.Infow("Resolving", "name", dependencyName, "as", dependency, "proto", "oci")
				resolver = r.ociResolver
			} else {
				logger.Infow("Resolving", "name", dependencyName, "as", dependency, "proto", "http")
				resolver = r.httpResolver
			}

			metadata, err := resolver.ResolveMetadata(dependency.PackageUri)

			if err != nil {
				logger.Errorw("Metadata resolving error", "name", dependencyName, "value", dependency)
				return nil, err
			}

			r.cache[dependency.PackageUri] = metadata
			result[dependency.PackageUri] = metadata

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
			result[dependency.PackageUri] = metadata
		}
	}
	return result, nil
}

func (r *Resolver) Download(dependencies map[string]*Metadata) error {
	logger := r.appConfig.Logger.Sugar()
	for u, m := range dependencies {
		var resolver DependencyResolver

		if m.ResolverType == OCI {
			logger.Infow("Downloading", "name", u, "as", m, "proto", "oci")
			resolver = r.ociResolver
		} else {
			logger.Infow("Resolving", "name", u, "as", m, "proto", "http")
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

		basePath := pklutils.PklGetRelativePath(r.basePath, baseUri, m.Version)
		err = os.MkdirAll(basePath, os.ModePerm)

		if err != nil {
			return err
		}

		metaPath := path.Join(basePath, fmt.Sprintf("%s@%s", m.Name, m.Version))
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

	return nil
}

func NewOciResolver(plainHttp bool, appConfig *AppConfig) (*OciResolver, error) {
	client, err := registry.NewClient(registry.WithPlainHttp(plainHttp))

	if err != nil {
		return nil, err
	}

	return &OciResolver{client: client, logger: appConfig.Logger}, nil
}

func (r *OciResolver) ResolveMetadata(uri string) (*Metadata, error) {
	ref, err := pklutils.PklUriToRef(uri)

	if err != nil {
		return nil, err
	}

	result, err := r.client.Pull(ref, registry.PullOptWithPackage(false))

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

	result, err := r.client.Pull(ref, registry.PullOptWithPackage(true))

	if err != nil {
		return nil, err
	}

	return result.Archive.Data, nil
}

func NewHttpResolver(plainHttp bool, appConfig *AppConfig) *HttpResolver {
	return &HttpResolver{plainHttp: plainHttp, logger: appConfig.Logger}
}

func (r *HttpResolver) ResolveMetadata(uri string) (*Metadata, error) {
	logger := r.logger.Sugar()
	u, err := url.Parse(uri)

	if err != nil {
		logger.Errorw("Parsing error", "uri", uri)
		return nil, err
	}

	if r.plainHttp {
		u.Scheme = "http"
	} else {
		u.Scheme = "https"
	}

	resp, err := http.Get(u.String())

	if err != nil {
		logger.Errorw("Http get error", "uri", u.String())
		return nil, err
	}

	if resp.StatusCode > 300 {
		logger.Errorw("Http error", "resp", resp.Status)
		return nil, fmt.Errorf("Http get Error")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	var metadata *Metadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		logger.Errorw("Json unmarshal error", "data", body)
		return nil, err
	}

	metadata.ResolverType = HTTP

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