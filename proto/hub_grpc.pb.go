// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: hub.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// RegistrarClient is the client API for Registrar service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RegistrarClient interface {
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterReply, error)
}

type registrarClient struct {
	cc grpc.ClientConnInterface
}

func NewRegistrarClient(cc grpc.ClientConnInterface) RegistrarClient {
	return &registrarClient{cc}
}

func (c *registrarClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterReply, error) {
	out := new(RegisterReply)
	err := c.cc.Invoke(ctx, "/cams.proto.Registrar/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RegistrarServer is the server API for Registrar service.
// All implementations must embed UnimplementedRegistrarServer
// for forward compatibility
type RegistrarServer interface {
	Register(context.Context, *RegisterRequest) (*RegisterReply, error)
	mustEmbedUnimplementedRegistrarServer()
}

// UnimplementedRegistrarServer must be embedded to have forward compatible implementations.
type UnimplementedRegistrarServer struct {
}

func (UnimplementedRegistrarServer) Register(context.Context, *RegisterRequest) (*RegisterReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedRegistrarServer) mustEmbedUnimplementedRegistrarServer() {}

// UnsafeRegistrarServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RegistrarServer will
// result in compilation errors.
type UnsafeRegistrarServer interface {
	mustEmbedUnimplementedRegistrarServer()
}

func RegisterRegistrarServer(s grpc.ServiceRegistrar, srv RegistrarServer) {
	s.RegisterService(&Registrar_ServiceDesc, srv)
}

func _Registrar_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RegistrarServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cams.proto.Registrar/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RegistrarServer).Register(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Registrar_ServiceDesc is the grpc.ServiceDesc for Registrar service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Registrar_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cams.proto.Registrar",
	HandlerType: (*RegistrarServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Register",
			Handler:    _Registrar_Register_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "hub.proto",
}

// ControllerClient is the client API for Controller service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ControllerClient interface {
	// Stream of commands from the server to the client
	Control(ctx context.Context, opts ...grpc.CallOption) (Controller_ControlClient, error)
	// Stream of media frames from the client to the server
	// There should be at most one long-standing call to MediaUpload per agent connected
	// to the cloud.
	MediaUpload(ctx context.Context, opts ...grpc.CallOption) (Controller_MediaUploadClient, error)
}

type controllerClient struct {
	cc grpc.ClientConnInterface
}

func NewControllerClient(cc grpc.ClientConnInterface) ControllerClient {
	return &controllerClient{cc}
}

func (c *controllerClient) Control(ctx context.Context, opts ...grpc.CallOption) (Controller_ControlClient, error) {
	stream, err := c.cc.NewStream(ctx, &Controller_ServiceDesc.Streams[0], "/cams.proto.Controller/Control", opts...)
	if err != nil {
		return nil, err
	}
	x := &controllerControlClient{stream}
	return x, nil
}

type Controller_ControlClient interface {
	Send(*ControlReply) error
	Recv() (*ControlRequest, error)
	grpc.ClientStream
}

type controllerControlClient struct {
	grpc.ClientStream
}

func (x *controllerControlClient) Send(m *ControlReply) error {
	return x.ClientStream.SendMsg(m)
}

func (x *controllerControlClient) Recv() (*ControlRequest, error) {
	m := new(ControlRequest)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *controllerClient) MediaUpload(ctx context.Context, opts ...grpc.CallOption) (Controller_MediaUploadClient, error) {
	stream, err := c.cc.NewStream(ctx, &Controller_ServiceDesc.Streams[1], "/cams.proto.Controller/MediaUpload", opts...)
	if err != nil {
		return nil, err
	}
	x := &controllerMediaUploadClient{stream}
	return x, nil
}

type Controller_MediaUploadClient interface {
	Send(*MediaFrame) error
	CloseAndRecv() (*MediaReply, error)
	grpc.ClientStream
}

type controllerMediaUploadClient struct {
	grpc.ClientStream
}

func (x *controllerMediaUploadClient) Send(m *MediaFrame) error {
	return x.ClientStream.SendMsg(m)
}

func (x *controllerMediaUploadClient) CloseAndRecv() (*MediaReply, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(MediaReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ControllerServer is the server API for Controller service.
// All implementations must embed UnimplementedControllerServer
// for forward compatibility
type ControllerServer interface {
	// Stream of commands from the server to the client
	Control(Controller_ControlServer) error
	// Stream of media frames from the client to the server
	// There should be at most one long-standing call to MediaUpload per agent connected
	// to the cloud.
	MediaUpload(Controller_MediaUploadServer) error
	mustEmbedUnimplementedControllerServer()
}

// UnimplementedControllerServer must be embedded to have forward compatible implementations.
type UnimplementedControllerServer struct {
}

func (UnimplementedControllerServer) Control(Controller_ControlServer) error {
	return status.Errorf(codes.Unimplemented, "method Control not implemented")
}
func (UnimplementedControllerServer) MediaUpload(Controller_MediaUploadServer) error {
	return status.Errorf(codes.Unimplemented, "method MediaUpload not implemented")
}
func (UnimplementedControllerServer) mustEmbedUnimplementedControllerServer() {}

// UnsafeControllerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ControllerServer will
// result in compilation errors.
type UnsafeControllerServer interface {
	mustEmbedUnimplementedControllerServer()
}

func RegisterControllerServer(s grpc.ServiceRegistrar, srv ControllerServer) {
	s.RegisterService(&Controller_ServiceDesc, srv)
}

func _Controller_Control_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ControllerServer).Control(&controllerControlServer{stream})
}

type Controller_ControlServer interface {
	Send(*ControlRequest) error
	Recv() (*ControlReply, error)
	grpc.ServerStream
}

type controllerControlServer struct {
	grpc.ServerStream
}

func (x *controllerControlServer) Send(m *ControlRequest) error {
	return x.ServerStream.SendMsg(m)
}

func (x *controllerControlServer) Recv() (*ControlReply, error) {
	m := new(ControlReply)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Controller_MediaUpload_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ControllerServer).MediaUpload(&controllerMediaUploadServer{stream})
}

type Controller_MediaUploadServer interface {
	SendAndClose(*MediaReply) error
	Recv() (*MediaFrame, error)
	grpc.ServerStream
}

type controllerMediaUploadServer struct {
	grpc.ServerStream
}

func (x *controllerMediaUploadServer) SendAndClose(m *MediaReply) error {
	return x.ServerStream.SendMsg(m)
}

func (x *controllerMediaUploadServer) Recv() (*MediaFrame, error) {
	m := new(MediaFrame)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Controller_ServiceDesc is the grpc.ServiceDesc for Controller service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Controller_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cams.proto.Controller",
	HandlerType: (*ControllerServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Control",
			Handler:       _Controller_Control_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "MediaUpload",
			Handler:       _Controller_MediaUpload_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "hub.proto",
}

// ConsumerClient is the client API for Consumer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ConsumerClient interface {
	Play(ctx context.Context, in *StreamId, opts ...grpc.CallOption) (Consumer_PlayClient, error)
}

type consumerClient struct {
	cc grpc.ClientConnInterface
}

func NewConsumerClient(cc grpc.ClientConnInterface) ConsumerClient {
	return &consumerClient{cc}
}

func (c *consumerClient) Play(ctx context.Context, in *StreamId, opts ...grpc.CallOption) (Consumer_PlayClient, error) {
	stream, err := c.cc.NewStream(ctx, &Consumer_ServiceDesc.Streams[0], "/cams.proto.Consumer/Play", opts...)
	if err != nil {
		return nil, err
	}
	x := &consumerPlayClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Consumer_PlayClient interface {
	Recv() (*MediaFrame, error)
	grpc.ClientStream
}

type consumerPlayClient struct {
	grpc.ClientStream
}

func (x *consumerPlayClient) Recv() (*MediaFrame, error) {
	m := new(MediaFrame)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ConsumerServer is the server API for Consumer service.
// All implementations must embed UnimplementedConsumerServer
// for forward compatibility
type ConsumerServer interface {
	Play(*StreamId, Consumer_PlayServer) error
	mustEmbedUnimplementedConsumerServer()
}

// UnimplementedConsumerServer must be embedded to have forward compatible implementations.
type UnimplementedConsumerServer struct {
}

func (UnimplementedConsumerServer) Play(*StreamId, Consumer_PlayServer) error {
	return status.Errorf(codes.Unimplemented, "method Play not implemented")
}
func (UnimplementedConsumerServer) mustEmbedUnimplementedConsumerServer() {}

// UnsafeConsumerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ConsumerServer will
// result in compilation errors.
type UnsafeConsumerServer interface {
	mustEmbedUnimplementedConsumerServer()
}

func RegisterConsumerServer(s grpc.ServiceRegistrar, srv ConsumerServer) {
	s.RegisterService(&Consumer_ServiceDesc, srv)
}

func _Consumer_Play_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamId)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ConsumerServer).Play(m, &consumerPlayServer{stream})
}

type Consumer_PlayServer interface {
	Send(*MediaFrame) error
	grpc.ServerStream
}

type consumerPlayServer struct {
	grpc.ServerStream
}

func (x *consumerPlayServer) Send(m *MediaFrame) error {
	return x.ServerStream.SendMsg(m)
}

// Consumer_ServiceDesc is the grpc.ServiceDesc for Consumer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Consumer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cams.proto.Consumer",
	HandlerType: (*ConsumerServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Play",
			Handler:       _Consumer_Play_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "hub.proto",
}
