package cmd

import (
	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/registry"
)

func NewLoginCmd(appConfig *app.AppConfig) *cobra.Command {
	var Login string
	var Password string
	var Insecure bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the registry",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {

			client, err := registry.NewClient()
			if err != nil {
				return err
			}

			err = client.Login(
				args[0],
				registry.LoginOptBasicAuth(Login, Password),
				registry.LoginOptInsecure(Insecure),
			)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&Login, "login", "l", "", "Registry login")
	cmd.Flags().StringVarP(&Password, "password", "p", "", "Registry password")
	cmd.Flags().BoolVarP(&Insecure, "insecure", "i", false, "Use insecure connection")
	cmd.MarkFlagRequired("login")
	cmd.MarkFlagRequired("password")

	return cmd
}
