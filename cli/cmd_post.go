package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/lesswrong-cli/lesswrong"
)

func (a *App) postCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "post <id>",
		Short: "Show a single LessWrong post by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			a.progressf("fetching post %s...", id)
			p, err := a.client.Post(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]lesswrong.Post{p})
		},
	}
}
