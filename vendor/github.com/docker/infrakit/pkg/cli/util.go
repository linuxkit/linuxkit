package cli

import (
	"os"

	logutil "github.com/docker/infrakit/pkg/log"
	"github.com/spf13/cobra"
)

var log = logutil.New("module", "cli/core")

// UpTree traverses up the command tree and starts executing the do function in the order from top
// of the command tree to the bottom.  Cobra commands executes only one level of PersistentPreRunE
// in reverse order.  This breaks our model of setting log levels at the very top and have the log level
// set throughout the entire hierarchy of command execution.
func UpTree(c *cobra.Command, do func(*cobra.Command, []string) error) error {
	if p := c.Parent(); p != nil {
		return UpTree(p, do)
	}
	return do(c, c.Flags().Args())
}

// EnsurePersistentPreRunE works around a limit of COBRA where only the persistent runE is executed at the
// parent of the leaf node.
func EnsurePersistentPreRunE(c *cobra.Command) error {
	return UpTree(c, func(x *cobra.Command, argv []string) error {
		if x.PersistentPreRunE != nil {
			return x.PersistentPreRunE(x, argv)
		}
		return nil
	})
}

// MustNotNil checks the object, if nil , exits and logs message
func MustNotNil(object interface{}, message string, ctx ...string) {
	if object == nil {
		log.Crit(message, ctx)
		os.Exit(-1)
	}
}
