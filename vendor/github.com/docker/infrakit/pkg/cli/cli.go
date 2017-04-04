package cli

import (
	"github.com/spf13/cobra"
)

const (
	// CliDirEnvVar is the environment variable that points to where the cli config folders are.
	CliDirEnvVar = "INFRAKIT_CLI_DIR"
)

// Modules provides access to CLI module discovery
type Modules interface {

	// List returns a list of preconfigured commands
	List() ([]*cobra.Command, error)
}
