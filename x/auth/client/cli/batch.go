package cli

import (
	"fmt"

	keybase "github.com/cosmos/cosmos-sdk/crypto/keys"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
)

func GetBatchSignCommand(codec *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-batch [in-file] [out-file]",
		Short: "Sign many standard transactions generated offline",
		Long: `Sign a list of transactions created with the --generate-only flag.
It will read StdSignDoc JSONs from [in-file], one transaction per line, and
produce a file of JSON encoded StdSignatures, one per line.

This command is intended to work offline for security purposes.`,
		PreRun: preSignCmd,
		RunE:   makeBatchSignCmd(codec),
		Args:   cobra.ExactArgs(2),
	}

	return flags.PostCommands(cmd)[0]
}

func makeBatchSignCmd(cdc *codec.Codec) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kb, err := keys.NewKeyBaseFromDir(viper.GetString(flags.FlagHome))
		if err != nil {
			return err
		}

		multisigInfo, err := kb.Get(viper.GetString(flags.FlagFrom))
		if err != nil {
			return err
		}

		if multisigInfo.GetType() != keybase.TypeMulti {
			return fmt.Errorf("%q must be of type %s: %s", args[1], keybase.TypeMulti, multisigInfo.GetType())
		}

		txsToSign, err := utils.ReadStdTxsFromFile(cdc, args[0])
		if err != nil {
			return errors.Wrap(err, "error extracting txs from file")
		}

		fmt.Printf("%v\n", txsToSign)

		return nil
	}
}
