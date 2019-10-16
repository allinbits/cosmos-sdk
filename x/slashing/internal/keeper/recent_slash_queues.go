package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/internal/types"
)

// RecentSlashesQueues

// Get the prefix store for the Recent Double Signs Queue
func (k Keeper) DoubleSignQueueStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), []byte("dsqueue"))
}

// InsertDoubleSignQueue inserts a double sign event into the queue at unbonding period after the double sign
func (k Keeper) InsertDoubleSignQueue(ctx sdk.Context, slashEvent types.SlashEvent) {
	dsStore := k.DoubleSignQueueStore(ctx)
	bz := k.cdc.MustMarshalBinaryBare(slashEvent)
	dsStore.Set(slashEvent.StoreKey(), bz)
}

// Get the prefix store for the Recent Liveness Faults Queue at jail period after the liveness fault
func (k Keeper) LivenessQueueStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), []byte("livequeue"))
}

// InsertLivenessQueue inserts a liveness slash event into the queue
func (k Keeper) InsertLivenessQueue(ctx sdk.Context, slashEvent types.SlashEvent) {
	liveStore := k.LivenessQueueStore(ctx)
	bz := k.cdc.MustMarshalBinaryBare(slashEvent)
	liveStore.Set(slashEvent.StoreKey(), bz)
}

// Iterators

// IterateDoubleSignQueue iterates over the slash events in the recent double signs queue
// and performs a callback function
func (k Keeper) IterateDoubleSignQueue(ctx sdk.Context, cb func(slashEvent types.SlashEvent) (stop bool)) {
	dsStore := k.DoubleSignQueueStore(ctx)
	iterator := dsStore.Iterator(nil, nil)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var slashEvent types.SlashEvent
		k.cdc.MustUnmarshalBinaryBare(iterator.Value(), &slashEvent)

		if cb(slashEvent) {
			break
		}
	}
}

// IterateLivenessQueue iterates over the slash events in the recent liveness faults queue
// and performs a callback function
func (k Keeper) IterateLivenessQueue(ctx sdk.Context, cb func(slashEvent types.SlashEvent) (stop bool)) {
	liveStore := k.LivenessQueueStore(ctx)
	iterator := liveStore.Iterator(nil, nil)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var slashEvent types.SlashEvent
		k.cdc.MustUnmarshalBinaryBare(iterator.Value(), &slashEvent)

		if cb(slashEvent) {
			break
		}
	}
}

// Deletes all the recent double sign slash events whose expiry time is older than current block time
func (k Keeper) PruneExpiredDoubleSignQueue(ctx sdk.Context) {
	dsStore := k.DoubleSignQueueStore(ctx)
	iterator := dsStore.Iterator(nil, sdk.PrefixEndBytes(sdk.FormatTimeBytes(ctx.BlockTime())))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		dsStore.Delete(iterator.Key())
	}
}

// Deletes all the recent liveness slash events whose expiry time is older than current block time
func (k Keeper) PruneExpiredLivenessQueue(ctx sdk.Context) {
	liveStore := k.LivenessQueueStore(ctx)
	iterator := liveStore.Iterator(nil, sdk.PrefixEndBytes(sdk.FormatTimeBytes(ctx.BlockTime())))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		liveStore.Delete(iterator.Key())
	}
}
