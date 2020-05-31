package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client"
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

	cmd.Flags().String(FlagPassPhrase, "", "The passphrase of the key needed to sign the transaction.")
	cmd.Flags().String(client.FlagOutputDocument, "",
		"write the resulto to the given file instead of the default location")

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

		passphrase := viper.GetString(FlagPassPhrase)
		if passphrase == "" {
			return fmt.Errorf("flag '--%s' is required", FlagPassPhrase)
		}

		accountNum := viper.GetUint64(client.FlagAccountNumber)
		if accountNum == 0 {
			return fmt.Errorf("flag '--%s' is required", client.FlagAccountNumber)
		}

		sequence := viper.GetUint64(client.FlagSequence)
		if sequence == 0 {
			return fmt.Errorf("flag '--%s' is required", client.FlagSequence)
		}

		chainId := viper.GetString(client.FlagChainID)
		if chainId == "" {
			return fmt.Errorf("flag '--%s' is required", client.FlagChainID)
		}

		txs, err := utils.ReadStdTxsFromFile(cdc, args[0])
		if err != nil {
			return errors.Wrap(err, "error extracting txs from file")
		}

		for _, tx := range txs {
			stdTx, err := utils.SignStdTx(txBldr, cliCtx, viper.GetString(flags.FlagFrom), tx, false, true)
			if err != nil {
				return errors.Wrap(err, "error signing stdTx")
			}

			json, err := cdc.MarshalJSON(stdTx.GetSignatures()[0])
			_, err = fmt.Fprintf(out, "%s\n", json)
			if err != nil {
				return errors.Wrap(err, "error writing to output")
			}
		}

		return nil
	}
}

func setOutput() (io.Writer, error) {
	outputFlag := viper.GetString(client.FlagOutputDocument)
	if outputFlag == "" {
		return os.Stdout, nil
	}

	out, err := os.OpenFile(outputFlag, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	return out, nil
}
