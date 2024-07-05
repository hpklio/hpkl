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
	Short:   "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// SilenceUsage:  true,
	// SilenceErrors: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// TODO: context?
	appConfig, err := app.NewAppConfig(context.Background())

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

	for _, c := range rootCmd.Commands() {
		c.Flags().StringVar(&appConfig.CacheDir, "cache-dir", path.Join(homeDir, ".pkl/cache"), "The cache directory for storing packages")
		c.Flags().StringVarP(&appConfig.WorkingDir, "working-dir", "w", workingDir, "Base path that relative module paths are resolved against.")
		c.Flags().StringVar(&appConfig.RootDir, "root-dir", "", "Restricts access to file-based modules and resources to those located under the root directory.")
	}
}
