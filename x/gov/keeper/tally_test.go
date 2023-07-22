package keeper_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtestutil "github.com/cosmos/cosmos-sdk/x/gov/testutil"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

	var (
		// handy functions
		delegatorVote = func(s suite, voter sdk.AccAddress, delegations []stakingtypes.Delegation, vote v1.VoteOption) {
			err := s.keeper.AddVote(s.ctx, s.proposal.Id, voter, v1.NewNonSplitVoteOption(vote), "")
			require.NoError(s.t, err)
			s.stakingKeeper.EXPECT().
				IterateDelegations(s.ctx, voter, gomock.Any()).
				DoAndReturn(
					func(ctx context.Context, voter sdk.AccAddress, fn func(index int64, d stakingtypes.DelegationI) bool) error {
						for i, d := range delegations {
							fn(int64(i), d)
						}
						return nil
					})
		}
		validatorVote = func(s suite, voter sdk.ValAddress, vote v1.VoteOption) {
			// validatorVote is like delegatorVote but without delegations
			delegatorVote(s, sdk.AccAddress(voter), nil, vote)
		}
	)
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
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "0",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "one validator votes",
			setup: func(s suite) {
				validatorVote(s, s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
			},
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "0",
				AbstainCount:    "0",
				NoCount:         "1000000",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "one account votes without delegation",
			setup: func(s suite) {
				delegatorVote(s, s.delAddrs[0], nil, v1.VoteOption_VOTE_OPTION_YES)
			},
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "0",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "one delegator votes",
			setup: func(s suite) {
				delegations := []stakingtypes.Delegation{{
					DelegatorAddress: s.delAddrs[0].String(),
					ValidatorAddress: s.valAddrs[0].String(),
					Shares:           sdkmath.LegacyNewDec(42),
				}}
				delegatorVote(s, s.delAddrs[0], delegations, v1.VoteOption_VOTE_OPTION_YES)
			},
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "42",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "one delegator votes yes, validator votes also yes",
			setup: func(s suite) {
				delegations := []stakingtypes.Delegation{{
					DelegatorAddress: s.delAddrs[0].String(),
					ValidatorAddress: s.valAddrs[0].String(),
					Shares:           sdkmath.LegacyNewDec(42),
				}}
				delegatorVote(s, s.delAddrs[0], delegations, v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "1000000",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "one delegator votes yes, validator votes no",
			setup: func(s suite) {
				delegations := []stakingtypes.Delegation{{
					DelegatorAddress: s.delAddrs[0].String(),
					ValidatorAddress: s.valAddrs[0].String(),
					Shares:           sdkmath.LegacyNewDec(42),
				}}
				delegatorVote(s, s.delAddrs[0], delegations, v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
			},
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "42",
				AbstainCount:    "0",
				NoCount:         "999958",
				NoWithVetoCount: "0",
			},
		},
		{
			// one delegator delegates 42 shares to 2 different validators (21 each)
			// delegator votes yes
			// first validator votes yes
			// second validator votes no
			// third validator (no delegation) votes abstain
			name: "delegator with mixed delegations",
			setup: func(s suite) {
				delegations := []stakingtypes.Delegation{
					{
						DelegatorAddress: s.delAddrs[0].String(),
						ValidatorAddress: s.valAddrs[0].String(),
						Shares:           sdkmath.LegacyNewDec(21),
					},
					{
						DelegatorAddress: s.delAddrs[0].String(),
						ValidatorAddress: s.valAddrs[1].String(),
						Shares:           sdkmath.LegacyNewDec(21),
					},
				}
				delegatorVote(s, s.delAddrs[0], delegations, v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				validatorVote(s, s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[2], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "1000021",
				AbstainCount:    "1000000",
				NoCount:         "999979",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "quorum reached with only abstain",
			setup: func(s suite) {
				validatorVote(s, s.valAddrs[0], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				validatorVote(s, s.valAddrs[1], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				validatorVote(s, s.valAddrs[2], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				validatorVote(s, s.valAddrs[3], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:        "0",
				AbstainCount:    "4000000",
				NoCount:         "0",
				NoWithVetoCount: "0",
			},
		},
		{
			name: "quorum reached with >1/3 veto",
			setup: func(s suite) {
				validatorVote(s, s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[2], v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				validatorVote(s, s.valAddrs[4], v1.VoteOption_VOTE_OPTION_NO_WITH_VETO)
				validatorVote(s, s.valAddrs[5], v1.VoteOption_VOTE_OPTION_NO_WITH_VETO)
				validatorVote(s, s.valAddrs[6], v1.VoteOption_VOTE_OPTION_NO_WITH_VETO)
			},
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:        "4000000",
				AbstainCount:    "0",
				NoCount:         "0",
				NoWithVetoCount: "3000000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			govKeeper, accountKeeper, bankKeeper, stakingKeeper, distKeeper, codec, ctx := setupGovKeeper(t)
			params := v1.DefaultParams()
			// Ensure params value are different than false
			params.BurnVoteQuorum = true
			params.BurnVoteVeto = true
			err := govKeeper.Params.Set(ctx, params)
			require.NoError(t, err)
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
			// Submit and activate a proposal
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
			assert.Equal(t, tt.expectedPass, pass, "wrong pass")
			assert.Equal(t, tt.expectedBurn, burn, "wrong burn")
			assert.Equal(t, tt.expectedTally, tally)
		})
	}
}
