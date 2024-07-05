package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
)

func NewPackageCmd(appConfig *app.AppConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Package hpkl project",
		RunE: func(cmd *cobra.Command, args []string) error {

			sugar := appConfig.Logger.Sugar()

			pklCmd := exec.Command(
				"pkl",
				"project",
				"package",
				"--skip-publish-check",
				"--working-dir",
				appConfig.WorkingDir,
				"--cache-dir",
				appConfig.CacheDir,
			)
			_, err := pklCmd.Output()

			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					sugar.Error(string(ee.Stderr))
				}
				return err
			}

			return nil
		},
	}

	return cmd
}
