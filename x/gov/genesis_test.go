package gov_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

func TestImportExportQueues_ErrorUnconsistentState(t *testing.T) {
	suite := createTestSuite(t)
	app := suite.App
	ctx := app.BaseApp.NewContext(false)
	require.Panics(t, func() {
		gov.InitGenesis(ctx, suite.AccountKeeper, suite.BankKeeper, suite.GovKeeper, &v1.GenesisState{
			Deposits: v1.Deposits{
				{
					ProposalId: 1234,
					Depositor:  "me",
					Amount: sdk.Coins{
						sdk.NewCoin(
							"stake",
							sdkmath.NewInt(1234),
						),
					},
				},
			},
		})
	})
	gov.InitGenesis(ctx, suite.AccountKeeper, suite.BankKeeper, suite.GovKeeper, v1.DefaultGenesisState())
	genState, err := gov.ExportGenesis(ctx, suite.GovKeeper)
	require.NoError(t, err)
	require.Equal(t, genState, v1.DefaultGenesisState())
}

func TestInitGenesis(t *testing.T) {
	var (
		testAddrs = simtestutil.CreateRandomAccounts(2)
		params    = &v1.Params{
			MinDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(42))),
		}
		quorumTimeout                = time.Hour * 20
		paramsWithQuorumCheckEnabled = &v1.Params{
			MinDeposit:       sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(42))),
			QuorumCheckCount: 10,
			QuorumTimeout:    &quorumTimeout,
		}

		depositAmount = sdk.Coins{
			sdk.NewCoin(
				"stake",
				sdkmath.NewInt(1234),
			),
		}
		deposits = v1.Deposits{
			{
				ProposalId: 1234,
				Depositor:  testAddrs[0].String(),
				Amount:     depositAmount,
			},
			{
				ProposalId: 1234,
				Depositor:  testAddrs[1].String(),
				Amount:     depositAmount,
			},
		}
		votes = []*v1.Vote{
			{
				ProposalId: 1234,
				Voter:      testAddrs[0].String(),
				Options:    v1.NewNonSplitVoteOption(v1.OptionYes),
			},
			{
				ProposalId: 1234,
				Voter:      testAddrs[1].String(),
				Options:    v1.NewNonSplitVoteOption(v1.OptionNo),
			},
		}
		depositEndTime  = time.Now().Add(time.Hour * 8)
		votingStartTime = time.Now()
		votingEndTime   = time.Now().Add(time.Hour * 24)
		proposals       = []*v1.Proposal{
			{
				Id:              1234,
				Status:          v1.StatusVotingPeriod,
				DepositEndTime:  &depositEndTime,
				VotingStartTime: &votingStartTime,
				VotingEndTime:   &votingEndTime,
			},
			{
				Id:              12345,
				Status:          v1.StatusDepositPeriod,
				DepositEndTime:  &depositEndTime,
				VotingStartTime: &votingStartTime,
				VotingEndTime:   &votingEndTime,
			},
			{
				Id:              123456,
				Status:          v1.StatusVotingPeriod,
				Expedited:       true,
				DepositEndTime:  &depositEndTime,
				VotingStartTime: &votingStartTime,
				VotingEndTime:   &votingEndTime,
			},
		}
		assertProposals = func(t *testing.T, ctx sdk.Context, s suite, expectedProposals []*v1.Proposal) {
			t.Helper()
			assert := assert.New(t)
			require := require.New(t)
			params, err := s.GovKeeper.Params.Get(ctx)
			require.NoError(err)
			it, err := s.GovKeeper.Proposals.Iterate(ctx, nil)
			require.NoError(err)
			proposals, err := it.Values()
			require.NoError(err)
			cdc := codec.NewLegacyAmino()
			expPropJSON := cdc.MustMarshalJSON(expectedProposals)
			propJSON := cdc.MustMarshalJSON(proposals)
			assert.JSONEq(string(expPropJSON), string(propJSON))
			// Check gov queues
			mustBool := func(b bool, err error) bool {
				require.NoError(err)
				return b
			}
			for _, p := range proposals {
				switch p.Status {
				case v1.StatusVotingPeriod:
					assert.True(mustBool(s.GovKeeper.ActiveProposalsQueue.Has(ctx, collections.Join(*p.VotingEndTime, p.Id))))
					assert.False(mustBool((s.GovKeeper.InactiveProposalsQueue.Has(ctx, collections.Join(*p.DepositEndTime, p.Id)))))
					assert.True(mustBool((s.GovKeeper.VotingPeriodProposals.Has(ctx, p.Id))))
					if params.QuorumCheckCount > 0 {
						if p.Expedited {
							assert.False(mustBool(s.GovKeeper.QuorumCheckQueue.Has(ctx, collections.Join(p.VotingStartTime.Add(*params.QuorumTimeout), p.Id))))
						} else {
							assert.True(mustBool(s.GovKeeper.QuorumCheckQueue.Has(ctx, collections.Join(p.VotingStartTime.Add(*params.QuorumTimeout), p.Id))))
						}
					}
				case v1.StatusDepositPeriod:
					assert.False(mustBool((s.GovKeeper.ActiveProposalsQueue.Has(ctx, collections.Join(*p.VotingEndTime, p.Id)))))
					assert.True(mustBool((s.GovKeeper.InactiveProposalsQueue.Has(ctx, collections.Join(*p.DepositEndTime, p.Id)))))
					assert.False(mustBool((s.GovKeeper.VotingPeriodProposals.Has(ctx, p.Id))))
				}
			}
		}
	)

	tests := []struct {
		name          string
		genesis       v1.GenesisState
		moduleBalance sdk.Coins
		requirePanic  bool
		assert        func(*testing.T, sdk.Context, suite)
	}{
		{
			name:         "fail: genesis without params",
			requirePanic: true,
		},
		{
			name: "ok: genesis with only params",
			genesis: v1.GenesisState{
				Params: params,
			},
			assert: func(t *testing.T, ctx sdk.Context, s suite) {
				t.Helper()
				p, err := s.GovKeeper.Params.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, *params, p)
			},
		},
		{
			name: "ok: genesis with constitution",
			genesis: v1.GenesisState{
				Params:       params,
				Constitution: "my constitution",
			},
			assert: func(t *testing.T, ctx sdk.Context, s suite) {
				t.Helper()
				p, err := s.GovKeeper.Params.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, *params, p)
				c, err := s.GovKeeper.Constitution.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, "my constitution", c)
			},
		},
		{
			name:          "fail: genesis with deposits but module balance is not equal to total deposits",
			moduleBalance: depositAmount,
			genesis: v1.GenesisState{
				Params:   params,
				Deposits: deposits,
			},
			requirePanic: true,
		},
		{
			name:          "ok: genesis with deposits and module balance is equal to total deposits",
			moduleBalance: depositAmount.MulInt(sdkmath.NewInt(2)), // *2 because there's 2 deposits
			genesis: v1.GenesisState{
				Params:   params,
				Deposits: deposits,
			},
			assert: func(t *testing.T, ctx sdk.Context, s suite) {
				t.Helper()
				p, err := s.GovKeeper.Params.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, *params, p)
				ds, err := s.GovKeeper.GetDeposits(ctx, deposits[0].ProposalId)
				require.NoError(t, err)
				assert.ElementsMatch(t, deposits, ds)
			},
		},
		{
			name: "ok: genesis with votes",
			genesis: v1.GenesisState{
				Params: params,
				Votes:  votes,
			},
			assert: func(t *testing.T, ctx sdk.Context, s suite) {
				t.Helper()
				p, err := s.GovKeeper.Params.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, *params, p)
				rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](1234)
				it, err := s.GovKeeper.Votes.Iterate(ctx, rng)
				require.NoError(t, err)
				vs, err := it.Values()
				require.NoError(t, err)
				var expectedVotes []v1.Vote // turn []*v1.Vote to []v1.Vote for assertion
				for _, v := range votes {
					expectedVotes = append(expectedVotes, *v)
				}
				assert.ElementsMatch(t, expectedVotes, vs)
			},
		},
		{
			name: "ok: genesis with proposals",
			genesis: v1.GenesisState{
				Params:    params,
				Proposals: proposals,
			},
			assert: func(t *testing.T, ctx sdk.Context, s suite) {
				t.Helper()
				p, err := s.GovKeeper.Params.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, *params, p)
				assertProposals(t, ctx, s, proposals)
			},
		},
		{
			name: "ok: genesis with proposals and quorum check enabled",
			genesis: v1.GenesisState{
				Params:    paramsWithQuorumCheckEnabled,
				Proposals: proposals,
			},
			assert: func(t *testing.T, ctx sdk.Context, s suite) {
				t.Helper()
				p, err := s.GovKeeper.Params.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, *paramsWithQuorumCheckEnabled, p)
				assertProposals(t, ctx, s, proposals)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := createTestSuite(t)
			app := suite.App
			ctx := app.BaseApp.NewContext(false)
			if tt.moduleBalance.IsAllPositive() {
				err := suite.BankKeeper.MintCoins(ctx, minttypes.ModuleName, tt.moduleBalance)
				require.NoError(t, err)
				err = suite.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, govtypes.ModuleName, tt.moduleBalance)
				require.NoError(t, err)
			}
			if tt.requirePanic {
				defer func() {
					require.NotNil(t, recover())
				}()
			}

			gov.InitGenesis(ctx, suite.AccountKeeper, suite.BankKeeper, suite.GovKeeper, &tt.genesis)

			if tt.requirePanic {
				require.Fail(t, "should have panic")
				return
			}
			tt.assert(t, ctx, suite)
		})
	}
}
