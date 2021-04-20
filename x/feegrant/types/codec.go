package types

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterInterface(
		"cosmos.feegrant.v1beta1.FeeAllowanceI",
		(*FeeAllowanceI)(nil),
		&BasicFeeAllowance{},
		&PeriodicFeeAllowance{},
		&AllowedMsgFeeAllowance{},
	)
}
