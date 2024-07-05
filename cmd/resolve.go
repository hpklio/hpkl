package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
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
					appConfig.WorkingDir = v
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error starting app: ", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cmd.Flags().BoolVarP(&appConfig.PlainHttp, "plain-http", "p", false, "Use plain http for registry")
	cmd.Flags().StringVar(&appConfig.CacheDir, "cache-dir", path.Join(homeDir, ".pkl/cache"), "The cache directory for storing packages")
	cmd.Flags().StringVarP(&appConfig.WorkingDir, "working-dir", "w", workingDir, "Base path that relative module paths are resolved against.")
	cmd.Flags().StringVar(&appConfig.RootDir, "root-dir", "", "Restricts access to file-based modules and resources to those located under the root directory.")

	return cmd
}

func Resolve(appConfig *app.AppConfig) error {
	resolver, err := app.NewResolver(appConfig)
	sugar := appConfig.Logger.Sugar()
	project := appConfig.Project()

	deps := project.Dependencies()

	sugar.Infow("got", "config", appConfig.Project())

	dependencies := make(map[string]app.Dependency, len(deps.RemoteDependencies))

	for n, d := range appConfig.Project().Dependencies().RemoteDependencies {
		dependencies[n] = app.Dependency{Uri: d.PackageUri}
	}

	resolvedDependencies, err := resolver.Resolve(dependencies)

	if err != nil {
		return err
	}

	err = resolver.Download(resolvedDependencies)

	if err != nil {
		return err
	}

	projectDeps := pklutils.ProjectDeps{
		SchemaVersion:        1,
		ResolvedDependencies: make(map[string]*pklutils.ResolvedDependency, len(resolvedDependencies)),
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

	err = pklutils.PklWriteDeps(appConfig.WorkingDir, &projectDeps)
	if err != nil {
		return err
	}

	return nil
}
