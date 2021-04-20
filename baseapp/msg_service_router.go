package baseapp

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	gogogrpc "github.com/gogo/protobuf/grpc"
	gogoproto "github.com/gogo/protobuf/proto"
	legacyproto "github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/cosmos/cosmos-sdk/apis/module"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/pkg/protohelpers"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MsgServiceRouter routes fully-qualified Msg service methods to their handler.
type MsgServiceRouter struct {
	interfaceRegistry codectypes.InterfaceRegistry

	msgHandlers         map[string]MsgServiceHandler      // maps msg name to MsgServiceHandler and maps what can be called externally
	serviceMsgToMsgName map[string]string                 // maps service msg name to msgHandlers name
	rpcInvokers         map[string]sdk.GRPCUnaryInvokerFn // maps gRPC Invoke path to the UnaryInvoker function, this maps what can be called internally
	invokerPathToServer map[string]interface{}            // maps gRPC invoke path to the actual server implementation
}

var _ gogogrpc.Server = &MsgServiceRouter{}

// NewMsgServiceRouter creates a new MsgServiceRouter.
func NewMsgServiceRouter() *MsgServiceRouter {
	return &MsgServiceRouter{
		interfaceRegistry:   nil,
		msgHandlers:         make(map[string]MsgServiceHandler),
		serviceMsgToMsgName: make(map[string]string),
		rpcInvokers:         make(map[string]sdk.GRPCUnaryInvokerFn),
		invokerPathToServer: make(map[string]interface{}),
	}
}

// MsgServiceHandler defines a function type which handles Msg service message.
type MsgServiceHandler = func(ctx sdk.Context, req sdk.MsgRequest) (*sdk.Result, error)

// Handler returns the MsgServiceHandler for a given query route path or nil
// if not found.
func (msr *MsgServiceRouter) Handler(methodName string) MsgServiceHandler {
	name, ok := msr.serviceMsgToMsgName[methodName]
	if !ok {
		return nil
	}
	handler, ok := msr.msgHandlers[name]
	if !ok {
		panic(fmt.Errorf("service msg name is registered but not the handler: %s", methodName))
	}
	return handler
}

// ExternalHandler returns the handler for the given sdk.Msg, it returns only
// the handlers which can be called externally.
func (msr *MsgServiceRouter) ExternalHandler(msg sdk.Msg) MsgServiceHandler {
	protoName := gogoproto.MessageName(msg)
	if protoName == "" {
		panic("received a non registered proto message")
	}
	handler, ok := msr.msgHandlers[protoName]
	if !ok {
		return nil
	}
	return handler
}

// InternalHandler returns the handler for the given method, it also returns internal handlers
func (msr *MsgServiceRouter) InternalHandler(method string) (handler interface{}, invoker sdk.GRPCUnaryInvokerFn) {
	invoker, exists := msr.rpcInvokers[method]
	if !exists {
		return nil, nil
	}
	handler, exists = msr.invokerPathToServer[method]
	if !exists {
		panic(fmt.Errorf("invoker for method %s exists but no service implementation was found", method))
	}
	return handler, invoker
}

func (msr *MsgServiceRouter) RegisterService(gRPCDesc *grpc.ServiceDesc, handler interface{}) {
	sd, err := protohelpers.ServiceDescriptorFromGRPCServiceDesc(gRPCDesc, nil, nil)
	if err != nil {
		panic(fmt.Errorf("unable to parse gRPC service descriptor: %w", err))
	}

	fqToHandler := make(map[string]sdk.GRPCUnaryInvokerFn, len(gRPCDesc.Methods))
	for _, method := range gRPCDesc.Methods {
		methodHandler := method.Handler
		fqToHandler[fmt.Sprintf("/%s/%s", gRPCDesc.ServiceName, method.MethodName)] = sdk.GRPCUnaryInvokerFn(methodHandler)
	}

	for i := 0; i < sd.Methods().Len(); i++ {
		md := sd.Methods().Get(i)
		err = msr.registerMethod(md, handler, fqToHandler[protohelpers.InvocationPath(md)])
		if err != nil {
			panic(fmt.Errorf("unable to register method %s: %w", sd.Methods().Get(i).FullName(), err))
		}
	}
}

// registerMethod registers the given gRPC RPC method, aside from that it will register in the interface registry the input as sdk.Msg
func (msr *MsgServiceRouter) registerMethod(md protoreflect.MethodDescriptor, srv interface{}, handler sdk.GRPCUnaryInvokerFn) error {
	// first we check if the concrete types were registered in the interface registry
	fqMethod := protohelpers.InvocationPath(md)
	typeURL := fmt.Sprintf(fmt.Sprintf("/%s", md.Input().FullName()))
	// check if they were registered
	_, err := msr.interfaceRegistry.Resolve(typeURL)
	if err == nil {
		return fmt.Errorf("input type %s was already registered", typeURL)
	}
	_, err = msr.interfaceRegistry.Resolve(fqMethod)
	if err == nil {
		return fmt.Errorf("type %s was already registered as ServiceMsg", fqMethod)
	}
	// register types in the interface registry
	rtype := gogoproto.MessageType((string)(md.Input().FullName()))
	if rtype == nil {
		return fmt.Errorf("unable to get concrete type for %s", md.Input().FullName())
	}
	concrete, ok := reflect.New(rtype).Elem().Interface().(gogoproto.Message)
	if !ok {
		return fmt.Errorf("type %s does not implement proto.Message", md.Input().FullName())
	}
	switch concrete.(type) {
	case sdk.Msg:
		msr.interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), concrete) // register as sdk.Msg
		msr.interfaceRegistry.RegisterCustomTypeURL((*sdk.Msg)(nil), fqMethod, concrete)
	case sdk.MsgRequest:
		msr.interfaceRegistry.RegisterImplementations((*sdk.MsgRequest)(nil), concrete) // register as sdk.Msg
		msr.interfaceRegistry.RegisterCustomTypeURL((*sdk.MsgRequest)(nil), fqMethod, concrete)
	default:
		return fmt.Errorf("type %s does not implement sdk.Msg or sdk.MsgRequest", md.Input().FullName())
	}
	// check if the method is internal or not
	internalRPC := false
	// mxtd is the method descriptor extension
	mxtd := md.Options().(*descriptorpb.MethodOptions)
	if mxtd != nil {
		v, err := legacyproto.GetExtension(mxtd, protohelpers.GogoProtoXtToProtoXt(module.E_Internal))
		if err != nil && !errors.Is(err, legacyproto.ErrMissingExtension) {
			return fmt.Errorf("unable to get extensions: %w", err)
		}
		internalRPC = *v.(*bool)
	}
	// check if it was already registered
	if _, exists := msr.rpcInvokers[fqMethod]; exists {
		return fmt.Errorf("the provided method %s was already registered", fqMethod)
	}
	// map what can be called internally
	msr.rpcInvokers[fqMethod] = handler
	msr.invokerPathToServer[fqMethod] = srv
	// if the method is internal then simply return
	if internalRPC {
		return nil
	}
	// if the method is not internal then map service msg type URL and
	// msg type URL to the msg handler
	msgHandler := func(ctx sdk.Context, req sdk.MsgRequest) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		interceptor := func(goCtx context.Context, _ interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			goCtx = context.WithValue(goCtx, sdk.SdkContextKey, ctx)
			return handler(goCtx, req)
		}
		// Call the method handler from the service description with the handler object.
		// We don't do any decoding here because the decoding was already done.
		res, err := handler(srv, sdk.WrapSDKContext(ctx), noopDecoder, interceptor)
		if err != nil {
			return nil, err
		}

		resMsg, ok := res.(gogoproto.Message)
		if !ok {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "Expecting proto.Message, got %T", resMsg)
		}

		return sdk.WrapServiceResult(ctx, resMsg, err)
	}
	msgName := gogoproto.MessageName(concrete) // get name
	msr.msgHandlers[msgName] = msgHandler
	msr.serviceMsgToMsgName[fqMethod] = msgName
	return nil
}

// SetInterfaceRegistry sets the interface registry for the router.
func (msr *MsgServiceRouter) SetInterfaceRegistry(interfaceRegistry codectypes.InterfaceRegistry) {
	msr.interfaceRegistry = interfaceRegistry
}

func noopDecoder(_ interface{}) error { return nil }
