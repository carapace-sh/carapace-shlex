package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-bridge/pkg/actions/bridge"
	shlex "github.com/carapace-sh/carapace-shlex"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  "carapace-shlex",
	Long: "simple shell lexer",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tokens, err := shlex.Split(args[0])
		if err != nil {
			return err
		}

		if cmd.Flag("current-pipeline").Changed {
			tokens = tokens.CurrentPipeline()
		}
		if cmd.Flag("filter-redirects").Changed {
			tokens = tokens.FilterRedirects()
		}
		if cmd.Flag("words").Changed {
			tokens = tokens.Words()
		}

		switch {
		case cmd.Flag("wordbreak-prefix").Changed:
			fmt.Fprintln(cmd.OutOrStdout(), tokens.WordbreakPrefix())
			return nil
		case cmd.Flag("join").Changed:
			words := make([]string, 0)
			for _, word := range tokens.Words() {
				words = append(words, word.Value)
			}
			fmt.Fprintln(cmd.OutOrStdout(), shlex.Join(words))
			return nil
		default:
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", "  ")
			return encoder.Encode(tokens)
		}
	},
}

func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().Bool("filter-redirects", false, "filter redirects")
	rootCmd.Flags().Bool("current-pipeline", false, "show current pipeline")
	rootCmd.Flags().Bool("wordbreak-prefix", false, "show wordbreak prefix")
	rootCmd.Flags().Bool("words", false, "combine adjoining tokens")
	rootCmd.Flags().Bool("join", false, "re-join words")

	rootCmd.MarkFlagsMutuallyExclusive(
		"join",
		"wordbreak-prefix",
	)

	carapace.Gen(rootCmd).PositionalCompletion(
		bridge.ActionCarapaceBin().SplitP(),
	)
}
