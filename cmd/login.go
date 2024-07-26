package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/registry"
)

func NewLoginCmd(appConfig *app.AppConfig) *cobra.Command {
	var Login string
	var Password string
	var Insecure bool
	var PasswordStdin bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the registry",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {

			client, err := registry.NewClient()

			if err != nil {
				return err
			}

			if PasswordStdin {
				secret, err := io.ReadAll(os.Stdin)

				if err != nil {
					return err
				}

				Password = strings.TrimSuffix(strings.TrimSuffix(string(secret), "\n"), "\r")
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
	cmd.Flags().BoolVar(&PasswordStdin, "password-stdin", false, "read password from stdin")
	cmd.Flags().BoolVarP(&Insecure, "insecure", "i", false, "Use insecure connection")
	cmd.MarkFlagRequired("login")
	cmd.MarkFlagRequired("password")

	return cmd
}
