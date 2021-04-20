package module

import (
	"github.com/golang/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/pkg/protohelpers"
)

// this registers the extension to the protov2 registry so that we can reflect it
func init() {
	proto.RegisterExtension(protohelpers.GogoProtoXtToProtoXt(E_Internal))
	proto.RegisterFile("apis/module/module.proto", fileDescriptor_4b6f5bff79d8037b)
}
