package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Tally iterates over the votes and updates the tally of a proposal based on the voting power of the
// voters
func (keeper Keeper) Tally(ctx context.Context, proposal v1.Proposal) (passes, burnDeposits bool, tallyResults v1.TallyResult, err error) {
	currValidators, err := keeper.getBondedValidatorsByAddress(ctx)
	if err != nil {
		return false, false, tallyResults, err
	}

	totalVotingPower, results, err := keeper.tallyVotes(ctx, proposal, currValidators, true)
	if err != nil {
		return false, false, tallyResults, err
	}

	params, err := keeper.Params.Get(ctx)
	if err != nil {
		return false, false, tallyResults, err
	}
	tallyResults = v1.NewTallyResultFromMap(results)

	// TODO: Upgrade the spec to cover all of these cases & remove pseudocode.
	// If there is no staked coins, the proposal fails
	totalBonded, err := keeper.sk.TotalBondedTokens(ctx)
	if err != nil {
		return false, false, tallyResults, err
	}

	if totalBonded.IsZero() {
		return false, false, tallyResults, nil
	}

	// If there is not enough quorum of votes, the proposal fails
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	quorum, _ := math.LegacyNewDecFromStr(params.Quorum)
	if percentVoting.LT(quorum) {
		return false, params.BurnVoteQuorum, tallyResults, nil
	}

	// If no one votes (everyone abstains), proposal fails
	if totalVotingPower.Sub(results[v1.OptionAbstain]).Equal(math.LegacyZeroDec()) {
		return false, false, tallyResults, nil
	}

	// If more than 1/3 of voters veto, proposal fails
	vetoThreshold, _ := math.LegacyNewDecFromStr(params.VetoThreshold)
	if results[v1.OptionNoWithVeto].Quo(totalVotingPower).GT(vetoThreshold) {
		return false, params.BurnVoteVeto, tallyResults, nil
	}

	// If more than 1/2 of non-abstaining voters vote Yes, proposal passes
	// For expedited 2/3
	var thresholdStr string
	if proposal.Expedited {
		thresholdStr = params.GetExpeditedThreshold()
	} else {
		thresholdStr = params.GetThreshold()
	}

	threshold, _ := math.LegacyNewDecFromStr(thresholdStr)

	if results[v1.OptionYes].Quo(totalVotingPower.Sub(results[v1.OptionAbstain])).GT(threshold) {
		return true, false, tallyResults, nil
	}

	// If more than 1/2 of non-abstaining voters vote No, proposal fails
	return false, false, tallyResults, nil
}

// HasReachedQuorum returns whether or not a proposal has reached quorum
// this is just a stripped down version of the Tally function above
func (keeper Keeper) HasReachedQuorum(ctx context.Context, proposal v1.Proposal) (quorumPassed bool, err error) {
	currValidators, err := keeper.getBondedValidatorsByAddress(ctx)
	if err != nil {
		return false, err
	}

	params, err := keeper.Params.Get(ctx)
	if err != nil {
		return false, err
	}

	// If there is no staked coins, the proposal has not reached quorum
	totalBonded, err := keeper.sk.TotalBondedTokens(ctx)
	if err != nil {
		return false, err
	}

	if totalBonded.IsZero() {
		return false, nil
	}

	// we check first if voting power of validators alone is enough to pass quorum
	// and if so, we return true skipping the iteration over all votes
	// can speed up computation in case quorum is already reached by validator votes alone
	approxTotalVotingPower := math.LegacyZeroDec()
	for _, val := range currValidators {
		ok, err := keeper.Votes.Has(ctx, collections.Join(proposal.Id, sdk.AccAddress(val.Address)))
		if err != nil {
			return false, err
		}
		if !ok {
			continue
		}
		approxTotalVotingPower = approxTotalVotingPower.Add(math.LegacyNewDecFromInt(val.BondedTokens))
	}
	// check and return whether or not the proposal has reached quorum
	approxPercentVoting := approxTotalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	quorum, _ := math.LegacyNewDecFromStr(params.Quorum)
	if approxPercentVoting.GTE(quorum) {
		return true, nil
	}
	totalVotingPower, _, err := keeper.tallyVotes(ctx, proposal, currValidators, false)
	if err != nil {
		return false, err
	}

	// check and return whether or not the proposal has reached quorum
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	return percentVoting.GTE(quorum), nil
}

// getBondedValidatorsByAddress fetches all the bonded validators and return
// them in map using their operator address as the key.
func (keeper Keeper) getBondedValidatorsByAddress(ctx context.Context) (map[string]v1.ValidatorGovInfo, error) {
	vals := make(map[string]v1.ValidatorGovInfo)

	err := keeper.sk.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		valStr, err := keeper.sk.ValidatorAddressCodec().BytesToString(validator.GetOperator())
		if err != nil {
			return false
		}
		vals[valStr] = v1.NewValidatorGovInfo(
			validator.GetOperator(),
			validator.GetBondedTokens(),
			validator.GetDelegatorShares(),
			math.LegacyZeroDec(),
			v1.WeightedVoteOptions{},
		)

		return false
	})
	return vals, err
}

// tallyVotes returns the total voting power and tally results of the votes
// on a proposal. If `isFinal` is true, results will be stored in `results`
// map and votes will be deleted. Otherwise, only the total voting power
// will be returned and `results` will be nil.
func (keeper Keeper) tallyVotes(
	ctx context.Context, proposal v1.Proposal,
	currValidators map[string]v1.ValidatorGovInfo, isFinal bool,
) (totalVotingPower math.LegacyDec, results map[v1.VoteOption]math.LegacyDec, err error) {
	totalVotingPower = math.LegacyZeroDec()
	if isFinal {
		results = make(map[v1.VoteOption]math.LegacyDec)
		results[v1.OptionYes] = math.LegacyZeroDec()
		results[v1.OptionAbstain] = math.LegacyZeroDec()
		results[v1.OptionNo] = math.LegacyZeroDec()
		results[v1.OptionNoWithVeto] = math.LegacyZeroDec()
	}

	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)
	err = keeper.Votes.Walk(ctx, rng, func(key collections.Pair[uint64, sdk.AccAddress], vote v1.Vote) (bool, error) {
		// if validator, just record it in the map
		voter, err := keeper.authKeeper.AddressCodec().StringToBytes(vote.Voter)
		if err != nil {
			return false, err
		}

		valAddrStr, err := keeper.sk.ValidatorAddressCodec().BytesToString(voter)
		if err != nil {
			return false, err
		}
		if val, ok := currValidators[valAddrStr]; ok {
			val.Vote = vote.Options
			currValidators[valAddrStr] = val
		}

		// iterate over all delegations from voter, deduct from any delegated-to validators
		err = keeper.sk.IterateDelegations(ctx, voter, func(index int64, delegation stakingtypes.DelegationI) (stop bool) {
			valAddrStr := delegation.GetValidatorAddr()

			if val, ok := currValidators[valAddrStr]; ok {
				// There is no need to handle the special case that validator address equal to voter address.
				// Because voter's voting power will tally again even if there will be deduction of voter's voting power from validator.
				val.DelegatorDeductions = val.DelegatorDeductions.Add(delegation.GetShares())
				currValidators[valAddrStr] = val

				// delegation shares * bonded / total shares
				votingPower := delegation.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)

				if isFinal {
					for _, option := range vote.Options {
						weight, _ := math.LegacyNewDecFromStr(option.Weight)
						subPower := votingPower.Mul(weight)
						results[option.Option] = results[option.Option].Add(subPower)
					}
				}
				totalVotingPower = totalVotingPower.Add(votingPower)
			}

			return false
		})
		if err != nil {
			return false, err
		}

		if isFinal {
			return false, keeper.Votes.Remove(ctx, collections.Join(vote.ProposalId, sdk.AccAddress(voter)))
		}
		return false, nil
	})

	if err != nil && !errors.Is(err, collections.ErrInvalidIterator) {
		return totalVotingPower, nil, err
	}

	// iterate over the validators again to tally their voting power
	for _, val := range currValidators {
		if len(val.Vote) == 0 {
			continue
		}

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

		if isFinal {
			for _, option := range val.Vote {
				weight, _ := math.LegacyNewDecFromStr(option.Weight)
				subPower := votingPower.Mul(weight)
				results[option.Option] = results[option.Option].Add(subPower)
			}
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}

	return totalVotingPower, results, nil
}
