/*
Copyright © 2024 German Osin
*/
package cmd

import (
	"context"
	"os"
	"path"

	"log"

	"github.com/spf13/cobra"
	"go.szostok.io/version/extension"
	"hpkl.io/hpkl/pkg/app"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "hpkl",
	Version: app.Version(),
	Short:   "The tool extends PKL-lang with OCI and VALS capabilities, providing end-to-end configuration type safety.",
	Long: `The tool extends the PKL-lang with OCI and VALS capabilities, enhancing it to provide end-to-end configuration type safety. 
	This integration ensures robust, consistent, and error-free configurations by leveraging the OCI 
	(Open Container Initiative) standards for container specifications and the vals for secrets management. 
	The tool simplifies the development process, reduces configuration errors, and boosts overall system reliability 
	by enforcing strict type safety across all configurations.`,
	SilenceUsage: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	appConfig, err := app.NewAppConfig(
		context.Background(),
		rootCmd.OutOrStdout(),
		rootCmd.OutOrStderr(),
	)

	if err != nil {
		log.Fatal("Error starting app: ", err)
	}

	rootCmd.AddCommand(NewLoginCmd(appConfig))
	rootCmd.AddCommand(NewResolveCmd(appConfig))
	rootCmd.AddCommand(NewPublishCmd(appConfig))
	rootCmd.AddCommand(NewPackageCmd(appConfig))
	rootCmd.AddCommand(NewEvalCmd(appConfig))
	rootCmd.AddCommand(NewProjectCmd(appConfig))
	rootCmd.AddCommand(NewDownloadPackageCmd(appConfig))
	rootCmd.AddCommand(extension.NewVersionCobraCmd())

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error starting app: ", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	appConfig.DefaultCacheDir = path.Join(homeDir, ".pkl/cache")

	rootCmd.PersistentFlags().StringVar(&appConfig.CacheDir, "cache-dir", path.Join(homeDir, ".pkl/cache"), "The cache directory for storing packages")
	rootCmd.PersistentFlags().StringVarP(&appConfig.WorkingDir, "working-dir", "w", workingDir, "Base path that relative module paths are resolved against.")
	rootCmd.PersistentFlags().StringVar(&appConfig.RootDir, "root-dir", "", "Restricts access to file-based modules and resources to those located under the root directory.")
}
