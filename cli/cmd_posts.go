package cli

import (
	"time"

	"github.com/spf13/cobra"
)

// topCmd returns the top command with its --days flag.
func (a *App) topCmd() *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Top-scored LessWrong posts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			after := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
			a.progressf("fetching %d top posts (past %d days)...", n, days)
			posts, err := a.client.Posts(cmd.Context(), "top", n, after)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(posts, len(posts))
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "look-back window in days")
	return cmd
}

// postsCmd returns a simple post-list command for views that need no extra flags.
func (a *App) postsCmd(name, short, view string, defaultLimit int) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(defaultLimit)
			a.progressf("fetching %d %s posts...", n, name)
			posts, err := a.client.Posts(cmd.Context(), view, n, "")
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(posts, len(posts))
		},
	}
}
