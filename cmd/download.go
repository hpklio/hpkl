package cmd

import (
	"errors"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
)

func NewDownloadPackageCmd(appConfig *app.AppConfig) *cobra.Command {

	var noTransitive bool

	cmd := &cobra.Command{
		Use:   "download-package",
		Short: "Download package",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appConfig.CacheDir != appConfig.DefaultCacheDir {
				basePath := path.Join(appConfig.DefaultCacheDir, "package-2")
				baseTargetPath := path.Join(appConfig.CacheDir, "package-2")

				for _, v := range args {

					parts := strings.Split(v, "::")

					u, err := url.Parse(parts[0])
					if err != nil {
						panic(err)
					}

					relativePath := pklutils.PklGetRelativePath(basePath, u)
					targetPath := pklutils.PklGetRelativePath(baseTargetPath, u)
					parentDir := path.Join(targetPath, "..")

					if _, err := os.Stat(parentDir); errors.Is(err, os.ErrNotExist) {
						os.MkdirAll(parentDir, os.ModePerm)
					}

					err = os.Symlink(relativePath, targetPath)
					if err != nil {
						panic(err)
					}

				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&noTransitive, "no-transitive", false, "Skip downloading transitive dependencies of a package")

	return cmd
}
