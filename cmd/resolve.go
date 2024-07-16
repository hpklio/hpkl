package cmd

import (
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/apple/pkl-go/pkl"
	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
)

func NewResolveCmd(appConfig *app.AppConfig) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve all dependencies from pkl project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				for _, v := range args {
					appConfig.Logger.Sugar().Infow("Resolving", "path", v)
					appConfig.WorkingDir = v
					appConfig.Reset()
					err := Resolve(appConfig)
					if err != nil {
						panic(err)
					}
				}
			} else {
				err := Resolve(appConfig)
				if err != nil {
					panic(err)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&appConfig.PlainHttp, "plain-http", "p", false, "Use plain http for registry")

	return cmd
}

func CollectLocalDependencies(dependecies *pkl.ProjectDependencies) map[string]app.Dependency {
	result := make(map[string]app.Dependency)

	for n, dep := range dependecies.LocalDependencies {
		result[dep.PackageUri] = app.Dependency{Uri: dep.PackageUri, Name: n, ProjectFileUri: dep.ProjectFileUri}

		for _, localDep := range dep.Dependencies.LocalDependencies {
			inner := CollectLocalDependencies(localDep.Dependencies)
			maps.Copy(result, inner)
		}
	}

	return result
}

func CollectRemoteDependencies(dependecies *pkl.ProjectDependencies) map[string]app.Dependency {
	result := make(map[string]app.Dependency)

	for _, dep := range dependecies.LocalDependencies {
		remote := dep.Dependencies.RemoteDependencies

		for n, remoteDep := range remote {
			result[remoteDep.PackageUri] = app.Dependency{Uri: remoteDep.PackageUri, Name: n}
		}

		for _, localDep := range dep.Dependencies.LocalDependencies {
			inner := CollectRemoteDependencies(localDep.Dependencies)
			maps.Copy(result, inner)
		}
	}

	for n, dep := range dependecies.RemoteDependencies {
		result[dep.PackageUri] = app.Dependency{Uri: dep.PackageUri, Name: n}
	}

	return result
}

func Resolve(appConfig *app.AppConfig) error {
	resolver, err := app.NewResolver(appConfig)
	// sugar := appConfig.Logger.Sugar()
	project := appConfig.Project()

	remoteDependencies := CollectRemoteDependencies(project.Dependencies())

	resolvedDependencies, err := resolver.Resolve(remoteDependencies)

	if err != nil {
		return err
	}

	err = resolver.Download(resolvedDependencies)

	if err != nil {
		return err
	}

	dependencies := make(map[string]*pklutils.ResolvedDependency, len(resolvedDependencies)+len(project.Dependencies().LocalDependencies))

	projectDeps := pklutils.ProjectDeps{
		SchemaVersion:        1,
		ResolvedDependencies: dependencies,
	}

	for depUri, dep := range resolvedDependencies {

		baseUri, err := url.Parse(depUri)

		if err != nil {
			return err
		}

		mapUri := *baseUri
		mapUri.Path = strings.Replace(mapUri.Path, fmt.Sprintf("@%s", dep.Version), "", 1)

		baseUri.Scheme = "projectpackage"
		versionParsed := semver.MustParse(dep.Version)
		majorVersion := fmt.Sprintf("@%x", versionParsed.Major())
		mapUri.Path += majorVersion

		resolvedDependency := pklutils.ResolvedDependency{
			DependencyType: "remote",
			Uri:            baseUri.String(),
			Checksums:      map[string]string{"sha256": dep.PackageZipChecksums.Sha256},
		}

		projectDeps.ResolvedDependencies[mapUri.String()] = &resolvedDependency
	}

	projectFileUri, err := url.Parse(project.ProjectFileUri)
	projectFilePath := strings.Replace(projectFileUri.Path, "/PklProject", "", 1)

	versionRegex := regexp.MustCompile("^(.*)\\@(\\d+)\\.\\d.\\d")

	localDependencies := CollectLocalDependencies(project.Dependencies())

	for _, dep := range localDependencies {

		projectUri, err := url.Parse(dep.Uri)
		if err != nil {
			return err
		}
		projectUri.Scheme = "projectpackage"

		mapUri := versionRegex.ReplaceAllString(dep.Uri, "$1@$2")

		depProjectFileUri, err := url.Parse(dep.ProjectFileUri)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(projectFilePath, strings.Replace(depProjectFileUri.Path, "/PklProject", "", 1))

		if err != nil {
			return err
		}

		resolvedDependency := pklutils.ResolvedDependency{
			DependencyType: "local",
			Path:           rel,
			Uri:            projectUri.String(),
		}
		projectDeps.ResolvedDependencies[mapUri] = &resolvedDependency
	}

	err = pklutils.PklWriteDeps(appConfig.WorkingDir, &projectDeps)
	if err != nil {
		return err
	}

	return nil
}
