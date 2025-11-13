package enterprise

import (
	"log/slog"

	"github.com/s-samadi/ghas-lab-builder/internal/config"
	api "github.com/s-samadi/ghas-lab-builder/internal/github"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all organizations in the specified enterprise",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Traverse up to find and call the root command's PersistentPreRunE
		root := cmd
		for root.Parent() != nil {
			root = root.Parent()
		}

		// Call root's PersistentPreRunE if it exists
		if root.PersistentPreRunE != nil {
			if err := root.PersistentPreRunE(cmd, args); err != nil {
				return err
			}
		}

		ctx := cmd.Context()
		cmd.SetContext(ctx)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Get logger from context
		logger, ok := ctx.Value(config.LoggerKey).(*slog.Logger)
		if !ok {
			logger = slog.Default()
		}

		// Get enterprise slug from context using the proper key
		enterpriseSlug := ctx.Value(config.EnterpriseSlugKey).(string)

		organizations, err := api.GetEnterpriseOrganizations(ctx, logger, enterpriseSlug)
		if err != nil {
			return err
		}

		for _, org := range organizations {
			logger.Info("Organization found",
				slog.String("id", org.ID),
				slog.String("login", org.Login),
				slog.String("name", org.Name))
		}

		return nil

	},
}
