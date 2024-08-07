package cmd

import (
	"errors"
	"os/exec"

	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
)

func NewPackageCmd(appConfig *app.AppConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Package hpkl project",
		RunE: func(cmd *cobra.Command, args []string) error {

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
					return errors.New(string(ee.Stderr))
				}
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&appConfig.PlainHttp, "plain-http", "p", false, "Use plain http for registry")

	return cmd
}
