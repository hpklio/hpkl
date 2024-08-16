package cmd

import (
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
	"hpkl.io/hpkl/pkg/registry"
)

func NewPublishCmd(appConfig *app.AppConfig) *cobra.Command {

	logger := appConfig.Logger

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "publish package to oci registry",
		RunE: func(cmd *cobra.Command, args []string) error {

			project := appConfig.Project()
			name := project.Package.Name
			version := project.Package.Version
			baseUri := project.Package.BaseUri

			client, err := registry.NewClient(registry.WithPlainHttp(appConfig.PlainHttp))
			if err != nil {
				return err
			}

			appNameWithVersion := fmt.Sprintf("%s@%s", name, version)                    // app@version
			baseDir := path.Join(appConfig.WorkingDir, ".out", appNameWithVersion)       // working_dir/.out/app@version
			archivePath := path.Join(baseDir, fmt.Sprintf("%s.zip", appNameWithVersion)) // working_dir/.out/app@version/app@version.zip
			metadataPath := path.Join(baseDir, appNameWithVersion)                       // working_dir/.out/app@version/app@version

			ref, err := pklutils.PklBaseUriToRef(baseUri, version)

			if err != nil {
				return err
			}

			pushResult, err := client.Push(archivePath, metadataPath, ref, appConfig.Project())

			if err != nil {
				return err
			}

			logger.Info("Publish result: %+v", pushResult)

			return nil
		},
	}

	return cmd
}
