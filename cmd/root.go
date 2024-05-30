/*
Copyright © 2024 German Osin
*/
package cmd

import (
	"context"
	"os"

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
	rootCmd.AddCommand(NewPullCmd(appConfig))
	rootCmd.AddCommand(NewPublishCmd(appConfig))
	rootCmd.AddCommand(NewBuildCmd(appConfig))
	rootCmd.AddCommand(NewEvalCmd(appConfig))
	rootCmd.AddCommand(extension.NewVersionCobraCmd())
}
