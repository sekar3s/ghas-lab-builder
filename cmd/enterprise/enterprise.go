package enterprise

import (
	"github.com/spf13/cobra"
)

var EnterpriseCmd = &cobra.Command{
	Use:   "enterprise",
	Short: "Manage enterprise level operations",
	Long:  "The 'enterprise' command lets you manage operations at the GitHub Enterprise level.",
}

func init() {
	EnterpriseCmd.AddCommand(ListCmd)
}
