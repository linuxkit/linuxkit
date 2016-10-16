package cli

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	// Version is the build release identifier.
	Version = "Unspecified"

	// Revision is the build source control revision.
	Revision = "Unspecified"
)

// VersionCommand creates a cobra Command that prints build version information.
func VersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print build version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\n", Version)
			fmt.Printf("Revision: %s\n", Revision)
		},
	}
}
