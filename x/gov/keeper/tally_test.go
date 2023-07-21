package keeper_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtestutil "github.com/cosmos/cosmos-sdk/x/gov/testutil"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestTally(t *testing.T) {
	type suite struct {
		t             *testing.T
		proposal      v1.Proposal
		valAddrs      []sdk.ValAddress
		delAddrs      []sdk.AccAddress
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
			name:         "no votes",
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
		{
			name: "vote without delegation",
			setup: func(s suite) {
				voter := s.delAddrs[0]
				err := s.keeper.AddVote(s.ctx, s.proposal.Id, s.delAddrs[0], v1.NewNonSplitVoteOption(v1.VoteOption_VOTE_OPTION_YES), "")
				require.NoError(s.t, err)
				s.stakingKeeper.EXPECT().IterateDelegations(s.ctx, voter, gomock.Any())
			},
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:        "0",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "vote with delegation",
			setup: func(s suite) {
				voter := s.delAddrs[0]
				err := s.keeper.AddVote(s.ctx, s.proposal.Id, voter, v1.NewNonSplitVoteOption(v1.VoteOption_VOTE_OPTION_YES), "")
				require.NoError(s.t, err)
				s.stakingKeeper.EXPECT().
					IterateDelegations(s.ctx, voter, gomock.Any()).
					DoAndReturn(
						func(ctx context.Context, voter sdk.AccAddress, fn func(index int64, d stakingtypes.DelegationI) bool) error {
							fn(0, stakingtypes.Delegation{
								DelegatorAddress: voter.String(),
								ValidatorAddress: s.valAddrs[0].String(),
								Shares:           sdkmath.LegacyNewDec(42),
							})
							return nil
						})
			},
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:        "42",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			govKeeper, accountKeeper, bankKeeper, stakingKeeper, distKeeper, codec, ctx := setupGovKeeper(t)
			var (
				numVals       = 10
				numDelegators = 5
				addrs         = simtestutil.AddTestAddrsIncremental(
					bankKeeper, stakingKeeper, ctx, numVals+numDelegators,
					sdkmath.NewInt(10000000*v1.DefaultMinExpeditedDepositTokensRatio),
				)
				valAddrs = simtestutil.ConvertAddrsToValAddrs(addrs[:numVals])
				delAddrs = addrs[numVals:]
			)
			// Mocks a bunch of validators
			stakingKeeper.EXPECT().
				IterateBondedValidatorsByPower(ctx, gomock.Any()).
				DoAndReturn(
					func(ctx context.Context, fn func(index int64, validator stakingtypes.ValidatorI) bool) error {
						for i := int64(0); i < int64(numVals); i++ {
							fn(i, stakingtypes.Validator{
								OperatorAddress: valAddrs[i].String(),
								Status:          stakingtypes.Bonded,
								Tokens:          sdkmath.NewInt(1000000),
								DelegatorShares: sdkmath.LegacyNewDec(1000000),
							})
						}
						return nil
					})
			proposal, err := govKeeper.SubmitProposal(ctx, TestProposal, "", "title", "summary", delAddrs[0], tt.expedited)
			require.NoError(t, err)
			err = govKeeper.ActivateVotingPeriod(ctx, proposal)
			require.NoError(t, err)
			suite := suite{
				t:             t,
				proposal:      proposal,
				valAddrs:      valAddrs,
				delAddrs:      delAddrs,
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
			require.Equal(t, tt.expectedPass, pass, "wrong pass")
			require.Equal(t, tt.expectedBurn, burn, "wrong burn")
			require.Equal(t, tt.expectedTally, tally)
		})
	}
}
