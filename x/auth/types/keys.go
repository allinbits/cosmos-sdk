package types

import (
	prototypes "github.com/gogo/protobuf/types"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/schema/keeper"
)

const (
	// module name
	ModuleName = "auth"

	// StoreKey is string representation of the store key for auth
	StoreKey = "acc"

	// FeeCollectorName the root string for the fee collector account address
	FeeCollectorName = "fee_collector"

	// QuerierRoute is the querier route for auth
	QuerierRoute = ModuleName
)

var (
	// AddressStoreKeyPrefix prefix for account-by-address store
	AddressStoreKeyPrefix = []byte{0x01}

	// param key for global account number
	GlobalAccountNumberKey = []byte("globalAccountNumber")
)

// AddressStoreKey turn an address to key used to get it from the account store
func AddressStoreKey(addr sdk.AccAddress) []byte {
	return append(AddressStoreKeyPrefix, addr.Bytes()...)
}

func Schema() []keeper.KeyDescriptor {
	return []keeper.KeyDescriptor{
		{
			Name:        "AddressStoreKey",
			Description: "",
			Prefix:      AddressStoreKeyPrefix,
			KeyParts: []keeper.KeyPart{
				keeper.BytesKeyPart{
					Name:        "Address",
					Description: "",
					GoType:      sdk.AccAddress{},
					FixedWidth:  sdk.AddrLen,
				},
			},
			ValueProtoType:     &types.Any{},
			ValueInterfaceName: "cosmos_sdk.auth.v1.Account",
			ValueGoType:        (*exported.Account)(nil),
		},
		{
			Name:           "GlobalAccountNumber",
			Description:    "",
			Prefix:         GlobalAccountNumberKey,
			ValueProtoType: &prototypes.UInt64Value{},
		},
	}
}
