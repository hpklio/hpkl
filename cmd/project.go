package cmd

import (
	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
)

func NewProjectCmd(appConfig *app.AppConfig) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Project commands",
	}

	cmd.AddCommand(NewResolveCmd(appConfig))
	cmd.AddCommand(NewPackageCmd(appConfig))

	return cmd
}
