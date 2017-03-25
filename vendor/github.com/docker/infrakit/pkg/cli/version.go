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

var info = map[string]map[string]interface{}{}

// RegisterInfo allows any packages that use this register additional information to be displayed by the command.
// For example, a swarm flavor could register the docker api version.  This allows us to selectively incorporate
// only required dependencies based on package registration (in their init()) without explicitly pulling unused
// dependencies.
func RegisterInfo(key string, data map[string]interface{}) {
	info[key] = data
}

// VersionCommand creates a cobra Command that prints build version information.
func VersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print build version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("\n%-24s:  %v", "Version", Version)
			fmt.Printf("\n%-24s:  %v", "Revision", Revision)
			for k, m := range info {
				fmt.Printf("\n\n%s", k)
				for kk, vv := range m {
					fmt.Printf("\n%-24s:  %v", kk, vv)
				}
				fmt.Printf("\n")
			}
			fmt.Println()
		},
	}
}
