package dbdesign

import (
	"github.com/spf13/cobra"
	"go.risoftinc.com/elsa/constants"
)

// DBDesignCmd is the parent command for database designer
var DBDesignCmd = &cobra.Command{
	Use:   constants.DBDesignCommandUsage,
	Short: constants.DBDesignCommandShort,
	Long:  constants.DBDesignCommandLong,
}

func init() {
	DBDesignCmd.AddCommand(ServeCmd)
}
