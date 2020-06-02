package cli

import (
	"fmt"
	"io"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	FlagPassPhrase = "passphrase"
)

func GetBatchSignCommand(codec *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-batch [in-file]",
		Short: "Sign many standard transactions generated offline",
		Long: `Sign a list of transactions created with the --generate-only flag.
It will read StdSignDoc JSONs from [in-file], one transaction per line, and
produce a file of JSON encoded StdSignatures, one per line.

This command is intended to work offline for security purposes.`,
		PreRun: preSignCmd,
		RunE:   makeBatchSignCmd(codec),
		Args:   cobra.ExactArgs(1),
	}

	cmd.Flags().String(client.FlagOutputDocument, "",
		"write the result to the given file instead of the default location")

	cmd.Flags().String(
		FlagMultisig, "",
		"Address of the multisig account on behalf of which the transaction shall be signed",
	)

	return flags.PostCommands(cmd)[0]
}

func makeBatchSignCmd(cdc *codec.Codec) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cliCtx := context.NewCLIContext().WithCodec(cdc)
		txBldr := types.NewTxBuilderFromCLI()

		out, err := setOutput()
		if err != nil {
			return errors.Wrap(err, "error with output")
		}

		txs, err := utils.ReadStdTxsFromFile(cdc, args[0])
		if err != nil {
			return errors.Wrap(err, "error extracting txs from file")
		}

		multisigAddrStr := viper.GetString(FlagMultisig)

		sequence := txBldr.Sequence()
		for _, tx := range txs {
			txBldr = txBldr.WithSequence(sequence)

			var stdTx types.StdTx
			if multisigAddrStr != "" {
				var multisigAddr sdk.AccAddress

				multisigAddr, err = sdk.AccAddressFromBech32(multisigAddrStr)
				if err != nil {
					return err
				}

				stdTx, err = utils.SignStdTxWithSignerAddress(
					txBldr, cliCtx, multisigAddr, cliCtx.GetFromName(), tx, true,
				)
				if err != nil {
					return errors.Wrap(err, "error signing stdTx")
				}
			} else {
				stdTx, err = utils.SignStdTx(txBldr, cliCtx, viper.GetString(flags.FlagFrom), tx, false, true)
				if err != nil {
					return errors.Wrap(err, "error signing stdTx")
				}
			}

			json, err := cdc.MarshalJSON(stdTx.GetSignatures()[0])
			_, err = fmt.Fprintf(out, "%s\n", json)
			if err != nil {
				return errors.Wrap(err, "error writing to output")
			}

			sequence++
		}

		return nil
	}
}

func setOutput() (io.Writer, error) {
	outputFlag := viper.GetString(client.FlagOutputDocument)
	if outputFlag == "" {
		return os.Stdout, nil
	}

	out, err := os.Create(outputFlag)
	if err != nil {
		return nil, err
	}

	return out, nil
}
