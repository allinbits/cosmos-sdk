package types

import (
	types "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/authz/exported"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {

	registry.RegisterInterface(
		"cosmos.authz.v1beta1.Authorization",
		(*exported.Authorization)(nil),
		&bank.SendAuthorization{},
		&GenericAuthorization{},
		&staking.StakeAuthorization{},
	)
}
