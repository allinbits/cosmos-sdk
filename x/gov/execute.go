package gov

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Execute(ctx sdk.Context, k Keeper, p Proposal) (err error) {
	switch p.GetProposalType() {
	case ProposalTypeParameterChange:
		return ParamChangeProposalExecute(ctx, k, p.(*ParamChangeProposal))
	}
	return nil
}

func ParamChangeProposalExecute(ctx sdk.Context, k Keeper, p *ParamChangeProposal) (err error) {

	logger := ctx.Logger().With("module", "x/gov")
	logger.Info("Execute ParamChange begin", "info", fmt.Sprintf("current height:%d", ctx.BlockHeight()))

	pc := p.GetParamChange()

	subspace, found := k.paramsKeeper.GetSubspace(pc.Subspace)
	if !found {
		return ErrInvalidParamChange(DefaultCodespace, fmt.Sprintf("invalid subspace %s", pc.Subspace))
	}

	subspace.Set(ctx, pc.Key, pc.Value)

	return nil
}
