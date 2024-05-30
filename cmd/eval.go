package cmd

import (
	"github.com/apple/pkl-go/pkl"
	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
)

func NewEvalCmd(appConfig *app.AppConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Eval pkl file",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {

			// sugar := appConfig.Logger.Sugar()
			evaluator, err := pkl.NewEvaluator(
				cmd.Context(),
				pkl.PreconfiguredOptions,
				pklutils.WithVals(appConfig.Logger),
			)

			if err != nil {
				return err
			}

			text, err := evaluator.EvaluateOutputText(cmd.Context(), pkl.FileSource(args[0]))

			if err != nil {
				return err
			}

			cmd.OutOrStdout().Write([]byte(text))

			return nil
		},
	}

	return cmd
}
