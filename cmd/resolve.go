package cmd

import (
	"fmt"
	"net/url"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
)

func NewResolveCmd(appConfig *app.AppConfig) *cobra.Command {
	var plainHttp bool

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve all dependencies from pkl project",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolver, err := app.NewResolver(plainHttp, appConfig)
			sugar := appConfig.Logger.Sugar()
			project := appConfig.Project()

			deps := project.Dependencies()

			sugar.Infow("got", "config", appConfig.Project())

			dependencies := make(map[string]app.Dependency, len(deps.RemoteDependencies))

			for n, d := range appConfig.Project().Dependencies().RemoteDependencies {
				dependencies[n] = app.Dependency{PackageUri: d.PackageUri}
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

				baseUri.Scheme = "projectpackage"
				versionParsed := semver.MustParse(dep.Version)
				baseUri.Path += fmt.Sprintf("@%s", dep.Version)
				majorVersion := fmt.Sprintf("@%x", versionParsed.Major())
				mapUri.Path += majorVersion

				resolvedDependency := pklutils.ResolvedDependency{
					DependencyType: "remote",
					Uri:            baseUri.String(),
					Checksums:      map[string]string{"sha256": dep.PackageZipChecksums.Sha256},
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
