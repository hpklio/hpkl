package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
	"hpkl.io/hpkl/pkg/registry"
)

func NewPullCmd(appConfig *app.AppConfig) *cobra.Command {
	var plainHttp bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull all dependencies from hpkl file",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := registry.NewClient(registry.WithPlainHttp(plainHttp))
			sugar := appConfig.Logger.Sugar()
			sugar.Infow("got", "config", appConfig.Project)

			dependencies := appConfig.Project.Dependencies().RemoteDependencies

			projectDeps := pklutils.ProjectDeps{
				SchemaVersion:        1,
				ResolvedDependencies: make(map[string]*pklutils.ResolvedDependency, len(dependencies)),
			}

			for _, dep := range dependencies {
				if err != nil {
					return err
				}
				res, err := doPull(client, dep.PackageUri, appConfig)

				if err != nil {
					return err
				}

				digestParts := strings.Split(res.Archive.Digest, ":")

				baseUri, err := url.Parse(res.Archive.Project.Package.BaseUri)

				if err != nil {
					return err
				}

				mapUri := *baseUri

				baseUri.Scheme = "projectpackage"
				versionParsed := semver.MustParse(res.Archive.Project.Package.Version)
				baseUri.Path += fmt.Sprintf("@%s", res.Archive.Project.Package.Version)
				majorVersion := fmt.Sprintf("@%x", versionParsed.Major())
				mapUri.Path += majorVersion

				resolvedDependency := pklutils.ResolvedDependency{
					DependencyType: "remote",
					Uri:            baseUri.String(),
					Checksums:      map[string]string{digestParts[0]: digestParts[1]},
				}

				projectDeps.ResolvedDependencies[mapUri.String()] = &resolvedDependency
			}

			err = pklutils.PklWriteDeps(&projectDeps)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&plainHttp, "plain-http", "p", false, "Use plain http for registry")
	return cmd
}

func doPull(client *registry.Client, uri string, appConfig *app.AppConfig) (*registry.PullResult, error) {
	logger := appConfig.Logger.Sugar()

	ref, err := pklutils.PklUriToRef(uri)

	if err != nil {
		return nil, err
	}
	result, err := client.Pull(ref)
	if err != nil {
		return nil, err
	}
	logger.Debugf("got package", "pkg", result)

	project := result.Archive.Project
	name := project.Package.Name
	version := project.Package.Version
	baseUri, err := url.Parse(project.Package.BaseUri)

	if err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()

	if err != nil {
		return nil, err
	}

	basePath := pklutils.PklGetRelativePath(path.Join(homeDir, ".pkl/cache/package-1"), baseUri, version)
	err = os.MkdirAll(basePath, os.ModePerm)

	if err != nil {
		return nil, err
	}

	metaPath := path.Join(basePath, fmt.Sprintf("%s@%s", name, version))
	archivePath := path.Join(basePath, fmt.Sprintf("%s@%s.zip", name, version))

	err = os.WriteFile(metaPath, result.Metadata.Data, os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(archivePath, result.Archive.Data, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return result, nil

}
