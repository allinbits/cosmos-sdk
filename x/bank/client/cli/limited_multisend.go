package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	bankfork "github.com/cosmos/cosmos-sdk/cmd/gaia/app/x/bank"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

var (
	uatomDenom    = "uatom"
	atomsToUatoms = int64(1000000)
)

// MultiSendTxCmd creates a (temporary) command to construt a limited MsgMultiSend
// transaction.
func MultiSendTxCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisend [recipient-address]",
		Short: "Create a (limited) unsigned multi-send tx",
		Long: `Create a (limited) unsigned multi-send tx. This command is only
temporary and should be used with caution in that it only allows for a limted
MsgMultiSend message to be generated.

This command will construct an unsigned MsgMultiSend message that has a single
input (the sender account) and two outputs, the burn address and the recipient.
The total of the single input will be 10atom, where 9atom gets sent to the
burn address (cosmos1x4p90uuy63fqzsheamn48vq88q3eusykf0a69v) and 1atom gets sent
to the recipient.

Example:
	gaiacli tx multisend cosmos1ukumqufn0x8ny7xk6quasus2rak5furuwazyml --from=<from_key_or_address> > unsigned_limited_multisend.json
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(cdc)

			if err := cliCtx.EnsureAccountExists(); err != nil {
				return err
			}

			to, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			coins := sdk.Coins{sdk.NewInt64Coin(uatomDenom, 10*atomsToUatoms)}

			from := cliCtx.GetFromAddress()
			account, err := cliCtx.GetAccount(from)
			if err != nil {
				return err
			}

			// ensure account has enough coins
			if !account.GetCoins().IsAllGTE(coins) {
				return fmt.Errorf("address %s doesn't have enough coins to pay for this transaction", from)
			}

			input1 := bank.NewInput(from, coins)
			output1 := bank.NewOutput(bankfork.BurnedCoinsAccAddr, sdk.Coins{sdk.NewInt64Coin(uatomDenom, 9*atomsToUatoms)})
			output2 := bank.NewOutput(to, sdk.Coins{sdk.NewInt64Coin(uatomDenom, 1*atomsToUatoms)})
			msg := bank.NewMsgMultiSend([]bank.Input{input1}, []bank.Output{output1, output2})

			return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg}, false)
		},
	}

	return client.PostCommands(cmd)[0]
}
