package cmd

import (
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
)

func NewDownloadPackageCmd(appConfig *app.AppConfig) *cobra.Command {

	var noTransitive bool

	cmd := &cobra.Command{
		Use:   "download-package",
		Short: "Download package",
		RunE: func(cmd *cobra.Command, args []string) error {

			return nil
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error starting app: ", err)
	}

	cmd.Flags().StringVar(&appConfig.CacheDir, "cache-dir", path.Join(homeDir, ".pkl/cache"), "The cache directory for storing packages")
	cmd.Flags().BoolVar(&noTransitive, "no-transitive", false, "Skip downloading transitive dependencies of a package")

	return cmd
}
