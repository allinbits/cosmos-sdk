package flag

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"

	"cosmossdk.io/core/address"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type addressStringType struct{}

func (a addressStringType) NewValue(ctx context.Context, b *Builder) Value {
	return &addressValue{addressCodec: b.AddressCodec}
}

func (a addressStringType) DefaultValue() string {
	return ""
}

type validatorAddressStringType struct{}

func (a validatorAddressStringType) NewValue(ctx context.Context, b *Builder) Value {
	return &addressValue{addressCodec: b.ValidatorAddressCodec}
}

func (a validatorAddressStringType) DefaultValue() string {
	return ""
}

type addressValue struct {
	value        string
	addressCodec address.Codec
}

func (a addressValue) Get(protoreflect.Value) (protoreflect.Value, error) {
	return protoreflect.ValueOfString(a.value), nil
}

func (a addressValue) String() string {
	return a.value
}

// Set implements the flag.Value interface for addressValue it only supports bech32 addresses.
func (a *addressValue) Set(s string) error {
	_, err := a.addressCodec.StringToBytes(s)
	if err != nil {
		return fmt.Errorf("invalid bech32 account address: %w", err)
	}

	a.value = s

	return nil
}

func (a addressValue) Type() string {
	return "bech32 account address key name"
}

type consensusAddressStringType struct{}

func (a consensusAddressStringType) NewValue(ctx context.Context, b *Builder) Value {
	return &consensusAddressValue{addressValue: addressValue{addressCodec: b.ConsensusAddressCodec}}
}

func (a consensusAddressStringType) DefaultValue() string {
	return ""
}

type consensusAddressValue struct {
	addressValue
}

func (a consensusAddressValue) Get(protoreflect.Value) (protoreflect.Value, error) {
	return protoreflect.ValueOfString(a.value), nil
}

func (a consensusAddressValue) String() string {
	return a.value
}

func (a *consensusAddressValue) Set(s string) error {
	_, err := a.addressCodec.StringToBytes(s)
	if err == nil {
		a.value = s
		return nil
	}

	// fallback to pubkey parsing
	registry := types.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	var pk cryptotypes.PubKey
	err2 := cdc.UnmarshalInterfaceJSON([]byte(s), &pk)
	if err2 != nil {
		return fmt.Errorf("input isn't a pubkey %w or is invalid bech32 account address: %w", err, err2)
	}

	a.value = sdk.ConsAddress(pk.Address()).String()
	return nil
}
