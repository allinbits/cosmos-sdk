package keeper

import (
	"github.com/gogo/protobuf/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/schema/types"
)

type Keeper interface {
	RegisterKeyDescriptor(ctx sdk.Context, key sdk.StoreKey, descriptor types.KeyDescriptor)
}

type keeper struct {
	key sdk.StoreKey
}

var _ Keeper = keeper{}

func (k keeper) RegisterKeyDescriptor(ctx sdk.Context, key sdk.StoreKey, descriptor types.KeyDescriptor) {
	panic("implement me")
}

//type KeyPart struct {
//	Name string
//	Description string
//	Type proto.Message
//}

type SchemaBuilder interface {
	DescribeKey(name, description string)
}

type KeyDescriptor struct {
	Name, Description  string
	Prefix             []byte
	KeyParts           []KeyPart
	ValueProtoType     proto.Message
	ValueInterfaceName string
	ValueGoType        interface{}
}

type KeyPart interface{}

type BytesKeyPart struct {
	Name, Description string
	GoType            interface{}
	FixedWidth        int
}
