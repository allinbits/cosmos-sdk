package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

func GetBatchSignCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-batch [in-file] [out-file]",
		Short: "Sign many standard transactions generated offline",
		Long: `Sign a list of transactions created with the --generate-only flag.
It will read StdSignDoc JSONs from [in-file], one transaction per line, and
produce a file of JSON encoded StdSignatures, one per line.

This command is intended to work offline for security purposes.`,
	}

	return flags.PostCommands(cmd)[0]
}
