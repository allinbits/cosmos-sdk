package shared

import (
	"time"

	"github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/params"
)

// Keeper - shared params keeper
type Keeper struct {
	paramSpace params.Subspace
}

// NewKeeper creates a slashing keeper
func NewKeeper(paramspace types.ParamSubspace) Keeper {
	return Keeper{
		paramspace: paramspace.WithKeyTable(types.ParamKeyTable()),
	}
}

// Get all parameteras as types.Params
func (k Keeper) GetParams(ctx sdk.Context) Params {
	return types.NewParams(
		k.UnbondingTime(ctx),
		k.MaxValidators(ctx),
		k.BondDenom(ctx),
	)
}

// set the params
func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// UnbondingTime
func (k Keeper) UnbondingTime(ctx sdk.Context) (res time.Duration) {
	k.paramstore.Get(ctx, types.KeyUnbondingTime, &res)
	return
}

// MaxValidators - Maximum number of validators
func (k Keeper) MaxValidators(ctx sdk.Context) (res uint16) {
	k.paramstore.Get(ctx, types.KeyMaxValidators, &res)
	return
}

// BondDenom - Bondable coin denomination
func (k Keeper) BondDenom(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyBondDenom, &res)
	return
}
