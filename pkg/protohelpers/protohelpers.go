package protohelpers

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"

	gogoproto "github.com/gogo/protobuf/proto"
	legacyproto "github.com/golang/protobuf/proto" // nolint: staticcheck
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// ServiceDescriptorFromGRPCServiceDesc returns a protoreflect.ServiceDescriptor given
// a grpc.ServiceDesc. It is optionally possible to provide types and files registry
// that can then be used to build a descriptor with its dependencies resolved.
// If types or files are set to nil then the descriptor will not be registered.
func ServiceDescriptorFromGRPCServiceDesc(
	sd *grpc.ServiceDesc,
	files *protoregistry.Files,
	types *protoregistry.Types,
) (protoreflect.ServiceDescriptor, error) {
	fd, err := fileDescriptorFromServiceDesc(sd, files, types)
	if err != nil {
		return nil, err
	}
	rsd := fd.Services().ByName(protoreflect.FullName(sd.ServiceName).Name())
	if rsd == nil {
		return nil, fmt.Errorf("service descriptor not found for service: %s", sd.ServiceName)
	}
	return rsd, nil
}

// fileDescriptorFromServiceDesc returns the file descriptor given a gRPC service descriptor
func fileDescriptorFromServiceDesc(
	sd *grpc.ServiceDesc,
	files *protoregistry.Files,
	types *protoregistry.Types,
) (protoreflect.FileDescriptor, error) {
	if types == nil {
		types = new(protoregistry.Types)
	}
	if files == nil {
		files = new(protoregistry.Files)
	}
	var compressedFd []byte
	switch meta := sd.Metadata.(type) {
	case string:
		// TODO please remove this once we switch to protov2
		// check gogoproto registry
		compressedFd = gogoproto.FileDescriptor(meta)
		// check protobuf registry
		if len(compressedFd) == 0 {
			compressedFd = legacyproto.FileDescriptor(meta) // nolint: staticcheck
		}
	case []byte:
		compressedFd = meta
	default:
		return nil, fmt.Errorf("unknown metadata type: %T", meta)
	}

	if len(compressedFd) == 0 {
		return nil, fmt.Errorf("file descriptor not found for %s", sd.ServiceName)
	}
	// decompress file descriptor
	rawFd, err := DecompressFileDescriptor(compressedFd)
	if err != nil {
		return nil, err
	}
	// build fd with a new file and type registry as we don't need to put this into the global registry
	// we just need information
	fd, err := BuildFileDescriptor(rawFd, types, files)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

// BuildFileDescriptor returns the protoreflect.FileDescriptor from the given decompressed bytes of a protobuf raw file.
// if types and files are nil the default global proto registry will be used. In case an empty registry needs to be used
// then new(protoregistry.Types) and new(protoregistry.Files) can be used.
func BuildFileDescriptor(decompressedFd []byte, types *protoregistry.Types, files *protoregistry.Files) (fd protoreflect.FileDescriptor, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("unable to build descriptor: %v", r)
		}
	}()

	builder := protoimpl.DescBuilder{
		RawDescriptor: decompressedFd,
		TypeResolver:  types,
		FileRegistry:  files,
	}
	fd = builder.Build().File
	return fd, nil
}

// DecompressFileDescriptor decompresses the given compressed bytes of
// a protobuf file.
func DecompressFileDescriptor(compressed []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	out, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	return out, nil
}

func GogoProtoXtToProtoXt(xt *gogoproto.ExtensionDesc) *legacyproto.ExtensionDesc {
	return &legacyproto.ExtensionDesc{
		ExtendedType:  xt.ExtendedType,
		ExtensionType: xt.ExtensionType,
		Field:         xt.Field,
		Name:          xt.Name,
		Tag:           xt.Tag,
		Filename:      xt.Filename,
	}
}

// InvocationPath returns the path used to invoke a gRPC method
// from a grpc.ClientConn given its method descriptor
func InvocationPath(md protoreflect.MethodDescriptor) string {
	return fmt.Sprintf("/%s/%s", md.FullName().Parent(), md.Name())
}
