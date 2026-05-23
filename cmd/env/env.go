package env

import (
	"github.com/spf13/cobra"
	"go.risoftinc.com/elsa/constants"
)

// EnvCmd is the parent command for environment management
var EnvCmd = &cobra.Command{
	Use:   constants.EnvCommandUsage,
	Short: constants.EnvCommandShort,
	Long:  constants.EnvCommandLong,
}

func init() {
	EnvCmd.AddCommand(ServeCmd)
	EnvCmd.AddCommand(ResetCmd)
}
