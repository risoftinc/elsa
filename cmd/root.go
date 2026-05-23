package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	envcmd "go.risoftinc.com/elsa/cmd/env"
	"go.risoftinc.com/elsa/cmd/elsafile"
	"go.risoftinc.com/elsa/cmd/generate"
	"go.risoftinc.com/elsa/cmd/make"
	"go.risoftinc.com/elsa/cmd/migrate"
	"go.risoftinc.com/elsa/cmd/new"
	"go.risoftinc.com/elsa/cmd/watch"
	"go.risoftinc.com/elsa/constants"
	"go.risoftinc.com/elsa/internal/root"
)

var (
	version   string
	goVersion = runtime.Version()
	platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   constants.RootUse,
		Short: constants.RootShort,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := root.NewCommandHandler()
			return handler.HandleRootCommand(cmd, args, version)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add migration command
	rootCmd.AddCommand(migrate.MigrateCmd())

	// Add watch command
	rootCmd.AddCommand(watch.WatchCmd)

	// Add Elsafile commands
	rootCmd.AddCommand(elsafile.InitCmd)
	rootCmd.AddCommand(elsafile.RunCmd)
	rootCmd.AddCommand(elsafile.ListCmd)

	// Add generate commands
	rootCmd.AddCommand(generate.GenerateCmd)
	rootCmd.AddCommand(generate.GenCmd)

	// Add new command
	rootCmd.AddCommand(new.NewCmd)

	// Add make command
	rootCmd.AddCommand(make.MakeCmd)

	// Add env manager command
	rootCmd.AddCommand(envcmd.EnvCmd)
}

// SetVersionInfo sets the version information for the application
func SetVersionInfo(v string) {
	version = v
	// Override the version command to use our version info
	rootCmd.Version = version

	displayHelper := root.NewDisplayHelper()
	rootCmd.SetVersionTemplate(displayHelper.GetVersionTemplate(version, goVersion, platform))
}
