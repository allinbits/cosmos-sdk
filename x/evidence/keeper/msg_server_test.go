package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evidence/exported"
	"github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	"github.com/cosmos/cosmos-sdk/x/evidence/types"
)

type HandlerTestSuite struct {
	suite.Suite

	msgServer types.MsgServer
	app       *simapp.SimApp
}

func testMsgSubmitEvidence(r *require.Assertions, e exported.Evidence, s sdk.AccAddress) *types.MsgSubmitEvidence {
	msg, err := types.NewMsgSubmitEvidence(s, e)
	r.NoError(err)
	return msg
}

func (suite *HandlerTestSuite) SetupTest() {
	checkTx := false
	app := simapp.Setup(checkTx)

	// recreate keeper in order to use custom testing types
	evidenceKeeper := keeper.NewKeeper(
		app.AppCodec(), app.GetKey(types.StoreKey), app.StakingKeeper, app.SlashingKeeper,
	)
	router := types.NewRouter()
	router = router.AddRoute(types.RouteEquivocation, testEquivocationHandler(*evidenceKeeper))
	evidenceKeeper.SetRouter(router)

	app.EvidenceKeeper = *evidenceKeeper

	suite.msgServer = keeper.NewMsgServerImpl(*evidenceKeeper)
	suite.app = app
}

func (suite *HandlerTestSuite) TestMsgSubmitEvidence() {
	pk := ed25519.GenPrivKey()
	s := sdk.AccAddress("test________________")

	testCases := []struct {
		msg       *types.MsgSubmitEvidence
		expectErr bool
	}{
		{
			testMsgSubmitEvidence(
				suite.Require(),
				&types.Equivocation{
					Height:           11,
					Time:             time.Now().UTC(),
					Power:            100,
					ConsensusAddress: pk.PubKey().Address().String(),
				},
				s,
			),
			false,
		},
		{
			testMsgSubmitEvidence(
				suite.Require(),
				&types.Equivocation{
					Height:           10,
					Time:             time.Now().UTC(),
					Power:            100,
					ConsensusAddress: pk.PubKey().Address().String(),
				},
				s,
			),
			true,
		},
	}

	for i, tc := range testCases {
		ctx := suite.app.BaseApp.NewContext(false, tmproto.Header{Height: suite.app.LastBlockHeight() + 1})

		res, err := suite.msgServer.SubmitEvidence(sdk.WrapSDKContext(ctx), tc.msg)
		if tc.expectErr {
			suite.Require().Error(err, "expected error; tc #%d", i)
		} else {
			suite.Require().NoError(err, "unexpected error; tc #%d", i)
			suite.Require().NotNil(res, "expected non-nil result; tc #%d", i)

			suite.Require().Equal(tc.msg.GetEvidence().Hash().Bytes(), res.Hash, "invalid hash; tc #%d", i)
		}
	}
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
