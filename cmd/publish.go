package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
	"hpkl.io/hpkl/pkg/registry"
)

func NewPublishCmd(appConfig *app.AppConfig) *cobra.Command {
	var plainHttp bool

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "publish package to oci registry",
		RunE: func(cmd *cobra.Command, args []string) error {

			sugar := appConfig.Logger.Sugar()

			name := appConfig.Project.Package.Name
			version := appConfig.Project.Package.Version
			baseUri := appConfig.Project.Package.BaseUri

			client, err := registry.NewClient(registry.WithPlainHttp(plainHttp))
			if err != nil {
				return err
			}

			archivePath := fmt.Sprintf(".out/%s@%s/%s@%s.zip", name, version, name, version)
			metadataPath := fmt.Sprintf(".out/%s@%s/%s@%s", name, version, name, version)

			ref, err := pklutils.PklBaseUriToRef(baseUri, version)

			if err != nil {
				return err
			}

			sugar.Infof("generated", "ref", ref)

			pushResult, err := client.Push(archivePath, metadataPath, ref, appConfig.Project)

			if err != nil {
				return err
			}

			sugar.Infow("got", "pushResult", pushResult)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&plainHttp, "plain-http", "p", false, "Use plain http for registry")

	return cmd
}
