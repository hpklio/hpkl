package cmd

import (
	"github.com/apple/pkl-go/pkl"
	"github.com/spf13/cobra"
	"hpkl.io/hpkl/pkg/app"
	"hpkl.io/hpkl/pkg/pklutils"
)

func NewEvalCmd(appConfig *app.AppConfig) *cobra.Command {
	var expression string
	var moduleOutputSeparator string
	var format string

	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Eval pkl file",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {

			for i, module := range args {

				project, err := appConfig.ProjectOrErr()
				var projectFunc func(opts *pkl.EvaluatorOptions)

				if project != nil && err == nil {
					projectFunc = pkl.WithProject(project)
				} else {
					projectFunc = func(opts *pkl.EvaluatorOptions) {}
				}

				evaluator, err := pkl.NewEvaluator(
					cmd.Context(),
					projectFunc,
					pkl.PreconfiguredOptions,
					pklutils.WithVals(),
					func(opts *pkl.EvaluatorOptions) {
						opts.CacheDir = appConfig.CacheDir
						if appConfig.RootDir != "" {
							opts.RootDir = appConfig.RootDir
						}
						if format != "" {
							opts.OutputFormat = format
						}
					},
				)

				if err != nil {
					return err
				}

				var text string

				if expression == "" {
					text, err = evaluator.EvaluateOutputText(cmd.Context(), pkl.FileSource(module))
				} else {
					bytes, err := evaluator.EvaluateExpressionRaw(cmd.Context(), pkl.FileSource(module), expression)
					if err == nil {
						text = string(bytes[3:])
					}
				}

				if err != nil {
					return err
				}

				cmd.OutOrStdout().Write(([]byte)(text))

				if len(args) > i+1 {
					cmd.OutOrStdout().Write([]byte(moduleOutputSeparator))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&moduleOutputSeparator, "module-output-separator", "---", "Separator to use when multiple module outputs are written to the same file.")
	cmd.Flags().StringVarP(&expression, "expression", "x", "", "Expression to be evaluated within the module.")
	cmd.Flags().StringVarP(&format, "format", "f", "", "Output format to generate. <json, jsonnet,pcf, properties, plist, textproto, xml, yaml>")

	return cmd
}
