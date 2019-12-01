# ADR 016: Headers Keeper

## Changelog

- 2019 Nov 22: Initial draft

## Context

Applications such as IBC needs to be able to access historical block headers to perform application logic.
However, the underlying Tendermint may not store this information depending on the pruning setting, which
can produce indeterministic results on each node if used directly.

Therefore, this ADR is to add support for storing block headers in the SDK, and adding keeper interface for accessing
it.

## Decision

The new (where we should add this?) keeper is responsible for the ability to query a header based on height.

```go
func (k Keeper) GetBlockHeader(ctx Context, int64 height) error {
     // What kind of code do we need here?
     // Do we need to also write out how the headers gets stored?
}
```

## Status

Proposed

## Consequences

### Positive

- SDK Applications has the ability to query and utilize all historical block headers, which
  is an essentital requirement for some functionality.

### Negative

- More state is being stored in the SDK apps which leads to larger storage requirements.
  Block headers are small (how large are the size?) but can accumalate overtime.

### Neutral

## References

- [#4554](https://github.com/cosmos/cosmos-sdk/issues/4554)
