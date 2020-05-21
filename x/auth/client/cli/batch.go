package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/auth/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
)

const (
	FlagPassPhrase = "passphrase"
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

	cmd.Flags().String(
		FlagMultisig, "",
		"Address of the multisig account on behalf of which the transaction shall be signed",
	)

	cmd.Flags().String(FlagPassPhrase, "", "The passphrase of the key needed to sign the transaction.")

	return flags.PostCommands(cmd)[0]
}

func makeBatchSignCmd(cdc *codec.Codec) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kb, err := keys.NewKeyBaseFromDir(viper.GetString(flags.FlagHome))
		if err != nil {
			return err
		}

		multisigAddrStr := viper.GetString(FlagMultisig)
		if multisigAddrStr == "" {
			return fmt.Errorf("only multisig signature is supported, provide it with %s flag", FlagMultisig)
		}

		_, err = sdk.AccAddressFromBech32(multisigAddrStr)
		if err != nil {
			return err
		}

		from, err := kb.Get(viper.GetString(flags.FlagFrom))
		if err != nil {
			return errors.Wrap(err, "key not found")
		}

		txs, err := utils.ReadStdTxsFromFile(cdc, args[0])
		if err != nil {
			return errors.Wrap(err, "error extracting txs from file")
		}

		passphrase := viper.GetString(FlagPassPhrase)
		for _, tx := range txs {
			_, err = types.MakeSignature(nil, from.GetName(), passphrase, tx)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error signing tx %d", tx.Sequence))
			}
		}

		//fmt.Printf("%v\n", txsToSign)

		return nil
	}
}
