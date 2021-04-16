package baseapp

import (
	"context"
	"fmt"

	gogogrpc "github.com/gogo/protobuf/grpc"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/pkg/protohelpers"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MsgServiceRouter routes fully-qualified Msg service methods to their handler.
type MsgServiceRouter struct {
	interfaceRegistry codectypes.InterfaceRegistry
	serviceMsgRoutes  map[string]MsgServiceHandler // contains the broken google.Protobuf.Any spec routes
	msgFullnameToRPC  map[string]string            // maps tx.msgs.Any.TypeURL to the grpc.Service.Methods names saved in serviceMsgRoutes
}

var _ gogogrpc.Server = &MsgServiceRouter{}

// NewMsgServiceRouter creates a new MsgServiceRouter.
func NewMsgServiceRouter() *MsgServiceRouter {
	return &MsgServiceRouter{
		serviceMsgRoutes: make(map[string]MsgServiceHandler),
		msgFullnameToRPC: make(map[string]string),
	}
}

// MsgServiceHandler defines a function type which handles Msg service message.
type MsgServiceHandler = func(ctx sdk.Context, req sdk.MsgRequest) (*sdk.Result, error)

// Handler returns the MsgServiceHandler for a given query route path or nil
// if not found.
func (msr *MsgServiceRouter) Handler(methodName string) MsgServiceHandler {
	handler, ok := msr.serviceMsgRoutes[methodName]
	if !ok {
		return nil
	}
	return handler
}

// HandlerFor returns the handler for the given sdk.Msg
func (msr *MsgServiceRouter) HandlerFor(msg sdk.Msg) MsgServiceHandler {
	protoName := proto.MessageName(msg)
	if protoName == "" {
		panic("received a non registered proto message")
	}
	routeName, ok := msr.msgFullnameToRPC[protoName]
	if !ok {
		return nil
	}
	handler, ok := msr.serviceMsgRoutes[routeName]
	if !ok {
		return nil
	}
	return handler
}

// RegisterService implements the gRPC Server.RegisterService method. sd is a gRPC
// service description, handler is an object which implements that gRPC service.
//
// This function PANICs:
// - if it is called before the service `Msg`s have been registered using
//   RegisterInterfaces,
// - or if a service is being registered twice.
func (msr *MsgServiceRouter) RegisterService(gsd *grpc.ServiceDesc, handler interface{}) {
	// Adds a top-level query handler based on the gRPC service name.
	for _, method := range gsd.Methods {
		fqMethod := fmt.Sprintf("/%s/%s", gsd.ServiceName, method.MethodName)
		methodHandler := method.Handler

		// Check that the service Msg fully-qualified method name has already
		// been registered (via RegisterInterfaces). If the user registers a
		// service without registering according service Msg type, there might be
		// some unexpected behavior down the road. Since we can't return an error
		// (`Server.RegisterService` interface restriction) we panic (at startup).
		serviceMsg, err := msr.interfaceRegistry.Resolve(fqMethod)
		if err != nil || serviceMsg == nil {
			panic(
				fmt.Errorf(
					"type_url %s has not been registered yet. "+
						"Before calling RegisterService, you must register all interfaces by calling the `RegisterInterfaces` "+
						"method on module.BasicManager. Each module should call `msgservice.RegisterMsgServiceDesc` inside its "+
						"`RegisterInterfaces` method with the `_Msg_serviceDesc` generated by proto-gen",
					fqMethod,
				),
			)
		}

		// Check that each service is only registered once. If a service is
		// registered more than once, then we should error. Since we can't
		// return an error (`Server.RegisterService` interface restriction) we
		// panic (at startup).
		_, found := msr.serviceMsgRoutes[fqMethod]
		if found {
			panic(
				fmt.Errorf(
					"msg service %s has already been registered. Please make sure to only register each service once. "+
						"This usually means that there are conflicting modules registering the same msg service",
					fqMethod,
				),
			)
		}

		handler := func(ctx sdk.Context, req sdk.MsgRequest) (*sdk.Result, error) {
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			interceptor := func(goCtx context.Context, _ interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				goCtx = context.WithValue(goCtx, sdk.SdkContextKey, ctx)
				return handler(goCtx, req)
			}
			// Call the method handler from the service description with the handler object.
			// We don't do any decoding here because the decoding was already done.
			res, err := methodHandler(handler, sdk.WrapSDKContext(ctx), noopDecoder, interceptor)
			if err != nil {
				return nil, err
			}

			resMsg, ok := res.(proto.Message)
			if !ok {
				return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "Expecting proto.Message, got %T", resMsg)
			}

			return sdk.WrapServiceResult(ctx, resMsg, err)
		}

		msr.serviceMsgRoutes[fqMethod] = handler
	}
	// TODO(fdymylja): since the old sdk.Handlers are now deprecated, we register the sdk.Msg type URLs as handlers
	// and map those to their respective handlers. This should be the default way of handling an sdk.Msg
	// once we remove sdk.ServiceMsg which breaks google.protobuf.Any spec.
	sd, err := protohelpers.ServiceDescriptorFromGRPCServiceDesc(gsd)
	if err != nil {
		panic(err)
	}

	// here we're just mapping the request fullname, ex: cosmos.bank.MsgSend grpc MsgServer handler
	// after sdk.ServiceMsg is removed, this will register the request types
	// in the codec too, which will allow us to remove a lot of boilerplate from codec.go
	for mid := 0; mid < sd.Methods().Len(); mid++ {
		md := sd.Methods().Get(mid)
		msgFullname := md.Input().FullName()
		fqMethod := fmt.Sprintf("/%s/%s", sd.FullName(), md.Name())
		msr.msgFullnameToRPC[(string)(msgFullname)] = fqMethod
	}
}

// SetInterfaceRegistry sets the interface registry for the router.
func (msr *MsgServiceRouter) SetInterfaceRegistry(interfaceRegistry codectypes.InterfaceRegistry) {
	msr.interfaceRegistry = interfaceRegistry
}

func noopDecoder(_ interface{}) error { return nil }
