package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search across LessWrong posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching for %q...", args[0])
			posts, err := a.client.Search(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(posts, len(posts))
		},
	}
}
