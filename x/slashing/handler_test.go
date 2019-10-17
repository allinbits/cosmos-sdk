package slashing

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/internal/keeper"
	"github.com/cosmos/cosmos-sdk/x/slashing/internal/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

func TestCannotUnjailUnlessJailed(t *testing.T) {
	// initial setup
	ctx, ck, sk, _, keeper := slashingkeeper.CreateTestInput(t, DefaultParams())
	slh := NewHandler(keeper)
	amt := sdk.TokensFromConsensusPower(100)
	addr, val := slashingkeeper.Addrs[0], slashingkeeper.Pks[0]
	msg := slashingkeeper.NewTestMsgCreateValidator(addr, val, amt)
	got := staking.NewHandler(sk)(ctx, msg)
	require.True(t, got.IsOK(), "%v", got)
	staking.EndBlocker(ctx, sk)

	require.Equal(
		t, ck.GetCoins(ctx, sdk.AccAddress(addr)),
		sdk.Coins{sdk.NewCoin(sk.GetParams(ctx).BondDenom, slashingkeeper.InitTokens.Sub(amt))},
	)
	require.Equal(t, amt, sk.Validator(ctx, addr).GetBondedTokens())

	// assert non-jailed validator can't be unjailed
	got = slh(ctx, NewMsgUnjail(addr))
	require.False(t, got.IsOK(), "allowed unjail of non-jailed validator")
	require.EqualValues(t, CodeValidatorNotJailed, got.Code)
	require.EqualValues(t, DefaultCodespace, got.Codespace)
}

func TestCannotUnjailUnlessMeetMinSelfDelegation(t *testing.T) {
	// initial setup
	ctx, ck, sk, _, keeper := slashingkeeper.CreateTestInput(t, DefaultParams())
	slh := NewHandler(keeper)
	amtInt := int64(100)
	addr, val, amt := slashingkeeper.Addrs[0], slashingkeeper.Pks[0], sdk.TokensFromConsensusPower(amtInt)
	msg := slashingkeeper.NewTestMsgCreateValidator(addr, val, amt)
	msg.MinSelfDelegation = amt
	got := staking.NewHandler(sk)(ctx, msg)
	require.True(t, got.IsOK())
	staking.EndBlocker(ctx, sk)

	require.Equal(
		t, ck.GetCoins(ctx, sdk.AccAddress(addr)),
		sdk.Coins{sdk.NewCoin(sk.GetParams(ctx).BondDenom, slashingkeeper.InitTokens.Sub(amt))},
	)

	unbondAmt := sdk.NewCoin(sk.GetParams(ctx).BondDenom, sdk.OneInt())
	undelegateMsg := staking.NewMsgUndelegate(sdk.AccAddress(addr), addr, unbondAmt)
	got = staking.NewHandler(sk)(ctx, undelegateMsg)

	require.True(t, sk.Validator(ctx, addr).IsJailed())

	// assert non-jailed validator can't be unjailed
	got = slh(ctx, NewMsgUnjail(addr))
	require.False(t, got.IsOK(), "allowed unjail of validator with less than MinSelfDelegation")
	require.EqualValues(t, CodeValidatorNotJailed, got.Code)
	require.EqualValues(t, DefaultCodespace, got.Codespace)
}

func TestJailedValidatorDelegations(t *testing.T) {
	ctx, _, stakingKeeper, _, slashingKeeper := slashingkeeper.CreateTestInput(t, DefaultParams())

	stakingParams := stakingKeeper.GetParams(ctx)
	stakingParams.UnbondingTime = 0
	stakingKeeper.SetParams(ctx, stakingParams)

	// create a validator
	bondAmount := sdk.TokensFromConsensusPower(10)
	valPubKey := slashingkeeper.Pks[0]
	valAddr, consAddr := slashingkeeper.Addrs[1], sdk.ConsAddress(slashingkeeper.Addrs[0])

	msgCreateVal := slashingkeeper.NewTestMsgCreateValidator(valAddr, valPubKey, bondAmount)
	got := staking.NewHandler(stakingKeeper)(ctx, msgCreateVal)
	require.True(t, got.IsOK(), "expected create validator msg to be ok, got: %v", got)

	// end block
	staking.EndBlocker(ctx, stakingKeeper)

	// set dummy signing info
	newInfo := NewValidatorSigningInfo(consAddr, 0, 0, time.Unix(0, 0), false, 0)
	slashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)

	// delegate tokens to the validator
	delAddr := sdk.AccAddress(slashingkeeper.Addrs[2])
	msgDelegate := slashingkeeper.NewTestMsgDelegate(delAddr, valAddr, bondAmount)
	got = staking.NewHandler(stakingKeeper)(ctx, msgDelegate)
	require.True(t, got.IsOK(), "expected delegation to be ok, got %v", got)

	unbondAmt := sdk.NewCoin(stakingKeeper.GetParams(ctx).BondDenom, bondAmount)

	// unbond validator total self-delegations (which should jail the validator)
	msgUndelegate := staking.NewMsgUndelegate(sdk.AccAddress(valAddr), valAddr, unbondAmt)
	got = staking.NewHandler(stakingKeeper)(ctx, msgUndelegate)
	require.True(t, got.IsOK(), "expected begin unbonding validator msg to be ok, got: %v", got)

	err := stakingKeeper.CompleteUnbonding(ctx, sdk.AccAddress(valAddr), valAddr)
	require.Nil(t, err, "expected complete unbonding validator to be ok, got: %v", err)

	// verify validator still exists and is jailed
	validator, found := stakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)
	require.True(t, validator.IsJailed())

	// verify the validator cannot unjail itself
	got = NewHandler(slashingKeeper)(ctx, NewMsgUnjail(valAddr))
	require.False(t, got.IsOK(), "expected jailed validator to not be able to unjail, got: %v", got)

	// self-delegate to validator
	msgSelfDelegate := slashingkeeper.NewTestMsgDelegate(sdk.AccAddress(valAddr), valAddr, bondAmount)
	got = staking.NewHandler(stakingKeeper)(ctx, msgSelfDelegate)
	require.True(t, got.IsOK(), "expected delegation to not be ok, got %v", got)

	// verify the validator can now unjail itself
	got = NewHandler(slashingKeeper)(ctx, NewMsgUnjail(valAddr))
	require.True(t, got.IsOK(), "expected jailed validator to be able to unjail, got: %v", got)
}

func TestInvalidMsg(t *testing.T) {
	k := Keeper{}
	h := NewHandler(k)

	res := h(sdk.NewContext(nil, abci.Header{}, false, nil), sdk.NewTestMsg())
	require.False(t, res.IsOK())
	require.True(t, strings.Contains(res.Log, "unrecognized slashing message type"))
}

// Test a validator through uptime, downtime, revocation,
// unrevocation, starting height reset, and revocation again
func TestHandleAbsentValidator(t *testing.T) {

	// initial setup
	ctx, ck, sk, _, keeper := slashingkeeper.CreateTestInput(t, slashingkeeper.TestParams())
	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power)
	addr, val := slashingkeeper.Addrs[0], slashingkeeper.Pks[0]
	sh := staking.NewHandler(sk)
	slh := NewHandler(keeper)
	got := sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addr, val, amt))
	require.True(t, got.IsOK())
	staking.EndBlocker(ctx, sk)

	require.Equal(
		t, ck.GetCoins(ctx, sdk.AccAddress(addr)),
		sdk.NewCoins(sdk.NewCoin(sk.GetParams(ctx).BondDenom, slashingkeeper.InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, sk.Validator(ctx, addr).GetBondedTokens())

	// will exist since the validator has been bonded
	info, found := keeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(val.Address()))
	require.True(t, found)
	require.Equal(t, int64(0), info.StartHeight)
	require.Equal(t, int64(0), info.IndexOffset)
	require.Equal(t, int64(0), info.MissedBlocksCounter)
	require.Equal(t, time.Unix(0, 0).UTC(), info.JailedUntil)
	height := int64(0)

	// 1000 first blocks OK
	for ; height < keeper.SignedBlocksWindow(ctx); height++ {
		ctx = ctx.WithBlockHeight(height)
		keeper.HandleValidatorSignature(ctx, val.Address(), power, true)
	}
	info, found = keeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(val.Address()))
	require.True(t, found)
	require.Equal(t, int64(0), info.StartHeight)
	require.Equal(t, int64(0), info.MissedBlocksCounter)

	// 500 blocks missed
	for ; height < keeper.SignedBlocksWindow(ctx)+(keeper.SignedBlocksWindow(ctx)-keeper.MinSignedPerWindow(ctx)); height++ {
		ctx = ctx.WithBlockHeight(height)
		keeper.HandleValidatorSignature(ctx, val.Address(), power, false)
	}
	info, found = keeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(val.Address()))
	require.True(t, found)
	require.Equal(t, int64(0), info.StartHeight)
	require.Equal(t, keeper.SignedBlocksWindow(ctx)-keeper.MinSignedPerWindow(ctx), info.MissedBlocksCounter)

	// validator should be bonded still
	validator, _ := sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(val))
	require.Equal(t, sdk.Bonded, validator.GetStatus())
	bondPool := sk.GetBondedPool(ctx)
	require.True(sdk.IntEq(t, amt, bondPool.GetCoins().AmountOf(sk.BondDenom(ctx))))

	// 501st block missed
	ctx = ctx.WithBlockHeight(height)
	keeper.HandleValidatorSignature(ctx, val.Address(), power, false)
	info, found = keeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(val.Address()))
	require.True(t, found)
	require.Equal(t, int64(0), info.StartHeight)
	// counter now reset to zero
	require.Equal(t, int64(0), info.MissedBlocksCounter)

	slashAmt := amt.ToDec().Mul(keeper.SlashFractionDowntime(ctx).Mul(sdk.NewDec(power).QuoInt(sk.GetLastTotalPower(ctx)))).RoundInt64()

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should have been jailed
	validator, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(val))
	require.Equal(t, sdk.Unbonding, validator.GetStatus())

	// validator should have been slashed
	require.Equal(t, amt.Int64()-slashAmt, validator.GetTokens().Int64())

	// 502nd block *also* missed (since the LastCommit would have still included the just-unbonded validator)
	height++
	ctx = ctx.WithBlockHeight(height)
	keeper.HandleValidatorSignature(ctx, val.Address(), power, false)
	info, found = keeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(val.Address()))
	require.True(t, found)
	require.Equal(t, int64(0), info.StartHeight)
	require.Equal(t, int64(1), info.MissedBlocksCounter)

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should not have been slashed any more, since it was already jailed
	validator, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(val))
	require.Equal(t, amt.Int64()-slashAmt, validator.GetTokens().Int64())

	// unrevocation should fail prior to jail expiration
	got = slh(ctx, types.NewMsgUnjail(addr))
	require.False(t, got.IsOK())

	// unrevocation should succeed after jail expiration
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(1, 0).Add(keeper.DowntimeJailDuration(ctx))})
	got = slh(ctx, types.NewMsgUnjail(addr))
	require.True(t, got.IsOK())

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should be rebonded now
	validator, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(val))
	require.Equal(t, sdk.Bonded, validator.GetStatus())

	// validator should have been slashed
	bondPool = sk.GetBondedPool(ctx)
	require.Equal(t, amt.Int64()-slashAmt, bondPool.GetCoins().AmountOf(sk.BondDenom(ctx)).Int64())

	// Validator start height should not have been changed
	info, found = keeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(val.Address()))
	require.True(t, found)
	require.Equal(t, int64(0), info.StartHeight)
	// we've missed 2 blocks more than the maximum, so the counter was reset to 0 at 1 block more and is now 1
	require.Equal(t, int64(1), info.MissedBlocksCounter)

	// validator should not be immediately jailed again
	height++
	ctx = ctx.WithBlockHeight(height)
	keeper.HandleValidatorSignature(ctx, val.Address(), power, false)
	validator, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(val))
	require.Equal(t, sdk.Bonded, validator.GetStatus())

	// 500 signed blocks
	nextHeight := height + keeper.MinSignedPerWindow(ctx) + 1
	for ; height < nextHeight; height++ {
		ctx = ctx.WithBlockHeight(height)
		keeper.HandleValidatorSignature(ctx, val.Address(), power, false)
	}

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should be jailed again after 500 unsigned blocks
	nextHeight = height + keeper.MinSignedPerWindow(ctx) + 1
	for ; height <= nextHeight; height++ {
		ctx = ctx.WithBlockHeight(height)
		keeper.HandleValidatorSignature(ctx, val.Address(), power, false)
	}

	// end block
	staking.EndBlocker(ctx, sk)

	validator, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(val))
	require.Equal(t, sdk.Unbonding, validator.GetStatus())
}

func TestProportionalSlashingLiveness(t *testing.T) {

	// initial setup
	ctx, _, sk, _, keeper := slashingkeeper.CreateTestInput(t, slashingkeeper.TestParams())
	power := int64(10)
	addrs, vals := slashingkeeper.Addrs, slashingkeeper.Pks
	amt := sdk.TokensFromConsensusPower(power).ToDec()
	sh := staking.NewHandler(sk)
	// Give the first two validators 5% of voting power each.
	got := sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addrs[0], vals[0], amt.RoundInt()))
	require.True(t, got.IsOK())
	got = sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addrs[1], vals[1], amt.RoundInt()))
	require.True(t, got.IsOK())
	got = sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addrs[2], vals[2], amt.RoundInt().MulRaw(18)))
	require.True(t, got.IsOK(), got)
	staking.EndBlocker(ctx, sk)

	height := int64(0)

	// 1000 first blocks OK
	for ; height < keeper.SignedBlocksWindow(ctx); height++ {
		ctx = ctx.WithBlockHeight(height)
		keeper.HandleValidatorSignature(ctx, vals[0].Address(), power, true)
		keeper.HandleValidatorSignature(ctx, vals[1].Address(), power, true)
	}

	// 500 blocks missed
	for ; height < keeper.SignedBlocksWindow(ctx)+(keeper.SignedBlocksWindow(ctx)-keeper.MinSignedPerWindow(ctx)); height++ {
		ctx = ctx.WithBlockHeight(height)
		keeper.HandleValidatorSignature(ctx, vals[0].Address(), power, false)
		keeper.HandleValidatorSignature(ctx, vals[1].Address(), power, false)
	}

	// 501st block missed
	ctx = ctx.WithBlockHeight(height)
	keeper.HandleValidatorSignature(ctx, vals[0].Address(), power, false)
	keeper.HandleValidatorSignature(ctx, vals[1].Address(), power, false)

	// Each validator should be slashed SlashFractionDowntime * 0.2 ((sqrt(0.05) + sqrt(0.05))^2)

	slashAmt := amt.Mul(keeper.SlashFractionDowntime(ctx).Mul(sdk.NewDecFromIntWithPrec(sdk.NewInt(2), 1)))

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should have been jailed
	validator0, _ := sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[0]))
	validator1, _ := sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[1]))

	// validator should have been slashed
	require.Equal(t, amt.Sub(slashAmt).RoundInt64(), validator0.GetTokens().Int64())
	require.Equal(t, amt.Sub(slashAmt).RoundInt64(), validator1.GetTokens().Int64())
}

// Test proportional slashing for multiple validators double signing
func TestProportionalSlashingDoubleSign(t *testing.T) {

	// initial setup
	ctx, _, sk, _, keeper := slashingkeeper.CreateTestInput(t, slashingkeeper.TestParams())
	power := int64(100)
	addrs, vals := slashingkeeper.Addrs, slashingkeeper.Pks
	amt := sdk.TokensFromConsensusPower(power).ToDec()
	sh := staking.NewHandler(sk)
	got := sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addrs[0], vals[0], amt.RoundInt()))
	require.True(t, got.IsOK())
	got = sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addrs[1], vals[1], amt.RoundInt()))
	require.True(t, got.IsOK())
	got = sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addrs[2], vals[2], amt.RoundInt()))
	require.True(t, got.IsOK())
	got = sh(ctx, slashingkeeper.NewTestMsgCreateValidator(addrs[3], vals[3], amt.RoundInt()))
	require.True(t, got.IsOK())
	staking.EndBlocker(ctx, sk)

	height := int64(0)

	// Val 0 double signs
	keeper.HandleDoubleSign(ctx, addrs[0].Bytes(), height, ctx.BlockTime(), power, sk.GetLastTotalPower(ctx).Int64())
	slashAmt := amt.Mul(keeper.SlashFractionDoubleSign(ctx).Mul(sdk.NewDec(power).QuoInt(sk.GetLastTotalPower(ctx))))

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should have been slashed by slashAmt
	validator0, _ := sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[0]))

	require.Equal(t, amt.Sub(slashAmt).RoundInt(), validator0.GetTokens())

	// forward block time by 5 seconds
	ctx = ctx.WithBlockTime(ctx.BlockHeader().Time.Add(5 * time.Second))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 2)

	// Begin next block (prune recent slashes if needed)
	BeginBlocker(ctx, abci.RequestBeginBlock{}, keeper)

	// Val 1 double signs
	keeper.HandleDoubleSign(ctx, addrs[1].Bytes(), ctx.BlockHeight(), ctx.BlockTime(), power, sk.GetLastTotalPower(ctx).Int64())
	slashAmt2 := amt.Mul(keeper.SlashFractionDoubleSign(ctx).Mul(sdk.NewDec(power).QuoInt(sk.GetLastTotalPower(ctx))))

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should have been slashed
	validator0, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[0]))
	validator1, _ := sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[1]))
	// Check that the tokens of both validators are less than just slashAmt + slashAmt2
	require.Less(t, validator0.GetTokens().Int64(), amt.Sub(slashAmt).Sub(slashAmt2).RoundInt64())
	require.Less(t, validator1.GetTokens().Int64(), amt.Sub(slashAmt).Sub(slashAmt2).RoundInt64())

	val0Tokens := validator0.GetTokens()
	val1Tokens := validator1.GetTokens()

	// forward block time beyond unbonding period
	ctx = ctx.WithBlockTime(ctx.BlockHeader().Time.Add(sk.UnbondingTime((ctx))))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 200)

	// Begin next block (prune recent slashes if needed)
	BeginBlocker(ctx, abci.RequestBeginBlock{}, keeper)

	// Val 2 double signs
	keeper.HandleDoubleSign(ctx, addrs[2].Bytes(), ctx.BlockHeight(), ctx.BlockTime(), power, sk.GetLastTotalPower(ctx).Int64())
	slashAmt3 := amt.Mul(keeper.SlashFractionDoubleSign(ctx).Mul(sdk.NewDec(power).QuoInt(sk.GetLastTotalPower(ctx))))

	// end block
	staking.EndBlocker(ctx, sk)

	// validator should have been slashed
	validator0, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[0]))
	validator1, _ = sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[1]))
	validator2, _ := sk.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(vals[2]))
	// Check that the val2 tokens are only slashed by slashAmt3
	require.Equal(t, validator2.GetTokens(), amt.Sub(slashAmt3).RoundInt())
	// Check that val0 and val1 have not been slashed
	require.Equal(t, validator0.GetTokens(), val0Tokens)
	require.Equal(t, validator1.GetTokens(), val1Tokens)
}
