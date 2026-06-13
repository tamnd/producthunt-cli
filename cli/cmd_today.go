package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) todayCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "today",
		Aliases: []string{"feed"},
		Short:   "List today's top products from the Product Hunt feed",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching product hunt feed...")
			products, err := a.client.Today(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(products, len(products))
		},
	}
}
