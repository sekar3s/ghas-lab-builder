package orgs

import (
	"github.com/spf13/cobra"
)

var OrgsCmd = &cobra.Command{
	Use:   "orgs",
	Short: "Manage organizations within lab environments",
	Long:  "The 'orgs' command lets you create, delete, and manage organizations within GitHub Advanced Security lab environments.",
}

func init() {

	OrgsCmd.AddCommand(CreateCmd)
	OrgsCmd.AddCommand(DeleteCmd)
	OrgsCmd.AddCommand(deleteBatchCmd)
}
