package env

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.risoftinc.com/elsa/constants"
	"go.risoftinc.com/elsa/internal/envmanager"
)

const confirmCodeLength = 8

var resetDB string

var ResetCmd = &cobra.Command{
	Use:   constants.EnvResetUsage,
	Short: constants.EnvResetShort,
	Long:  constants.EnvResetLong,
	RunE:  runReset,
}

func init() {
	ResetCmd.Flags().StringVar(&resetDB, constants.EnvFlagDB, "", constants.EnvFlagDBUsage)
}

func runReset(cmd *cobra.Command, args []string) error {
	dbPath := resetDB
	if dbPath == "" {
		var err error
		dbPath, err = envmanager.DefaultDBPath()
		if err != nil {
			return err
		}
	}

	code, err := envmanager.GenerateConfirmCode(confirmCodeLength)
	if err != nil {
		return fmt.Errorf("generate confirmation code: %w", err)
	}

	if err := envmanager.CheckDatabaseNotInUse(dbPath); err != nil {
		if errors.Is(err, envmanager.ErrDatabaseInUse) {
			fmt.Println(constants.MsgEnvResetInUseHint)
		}
		return err
	}

	fmt.Println(constants.MsgEnvResetWarning)
	fmt.Printf(constants.MsgEnvResetDatabase+"\n", dbPath)
	fmt.Printf(constants.MsgEnvResetTypeCode+"\n", code)
	fmt.Print(constants.MsgEnvResetPrompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	input = strings.TrimSpace(input)

	if input != code {
		fmt.Println(constants.MsgEnvResetAborted)
		return fmt.Errorf(constants.ErrEnvResetMismatch)
	}

	if err := envmanager.DeleteDatabase(dbPath); err != nil {
		return err
	}

	fmt.Printf(constants.MsgEnvResetSuccess+"\n", dbPath)
	return nil
}
