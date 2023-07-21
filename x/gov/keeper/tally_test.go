package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtestutil "github.com/cosmos/cosmos-sdk/x/gov/testutil"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
)

func TestTally(t *testing.T) {
	type suite struct {
		t             *testing.T
		proposal      v1.Proposal
		addrs         []sdk.AccAddress
		keeper        *keeper.Keeper
		ctx           sdk.Context
		accountKeeper *govtestutil.MockAccountKeeper
		bankKeeper    *govtestutil.MockBankKeeper
		stakingKeeper *govtestutil.MockStakingKeeper
		distKeeper    *govtestutil.MockDistributionKeeper
		codec         moduletestutil.TestEncodingConfig
	}
	tests := []struct {
		name          string
		expedited     bool
		setup         func(suite)
		expectedPass  bool
		expectedBurn  bool
		expectedTally v1.TallyResult
		expectedError string
	}{
		{
			name:         "empty",
			setup:        func(s suite) {},
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:        "0",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			govKeeper, accountKeeper, bankKeeper, stakingKeeper, distKeeper, codec, ctx := setupGovKeeper(t)
			addrs := simtestutil.AddTestAddrsIncremental(bankKeeper, stakingKeeper, ctx, 2, sdkmath.NewInt(10000000*v1.DefaultMinExpeditedDepositTokensRatio))
			proposal, err := govKeeper.SubmitProposal(ctx, TestProposal, "", "title", "summary", addrs[0], tt.expedited)
			require.NoError(t, err)
			suite := suite{
				t:             t,
				proposal:      proposal,
				addrs:         addrs,
				ctx:           ctx,
				keeper:        govKeeper,
				accountKeeper: accountKeeper,
				bankKeeper:    bankKeeper,
				stakingKeeper: stakingKeeper,
				distKeeper:    distKeeper,
				codec:         codec,
			}
			tt.setup(suite)

			pass, burn, tally, err := govKeeper.Tally(ctx, proposal)

			require.NoError(t, err)
			require.Equal(t, tt.expectedPass, pass)
			require.Equal(t, tt.expectedBurn, burn)
			require.Equal(t, tt.expectedTally, tally)
		})
	}
}
