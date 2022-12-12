// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: hub.proto

package pb

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

// DownstreamClient is the client API for Downstream service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DownstreamClient interface {
	// Stream of commands from the server to the client
	Control(ctx context.Context, opts ...grpc.CallOption) (Downstream_ControlClient, error)
	// Stream of media frames from the client to the server
	// There should be at most one long-standing call to MediaUpload per agent connected
	// to the cloud.
	MediaUpload(ctx context.Context, opts ...grpc.CallOption) (Downstream_MediaUploadClient, error)
}

type downstreamClient struct {
	cc grpc.ClientConnInterface
}

func NewDownstreamClient(cc grpc.ClientConnInterface) DownstreamClient {
	return &downstreamClient{cc}
}

func (c *downstreamClient) Control(ctx context.Context, opts ...grpc.CallOption) (Downstream_ControlClient, error) {
	stream, err := c.cc.NewStream(ctx, &Downstream_ServiceDesc.Streams[0], "/cams.api.hub.Downstream/Control", opts...)
	if err != nil {
		return nil, err
	}
	x := &downstreamControlClient{stream}
	return x, nil
}

type Downstream_ControlClient interface {
	Send(*None) error
	Recv() (*DownstreamControlRequest, error)
	grpc.ClientStream
}

type downstreamControlClient struct {
	grpc.ClientStream
}

func (x *downstreamControlClient) Send(m *None) error {
	return x.ClientStream.SendMsg(m)
}

func (x *downstreamControlClient) Recv() (*DownstreamControlRequest, error) {
	m := new(DownstreamControlRequest)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *downstreamClient) MediaUpload(ctx context.Context, opts ...grpc.CallOption) (Downstream_MediaUploadClient, error) {
	stream, err := c.cc.NewStream(ctx, &Downstream_ServiceDesc.Streams[1], "/cams.api.hub.Downstream/MediaUpload", opts...)
	if err != nil {
		return nil, err
	}
	x := &downstreamMediaUploadClient{stream}
	return x, nil
}

type Downstream_MediaUploadClient interface {
	Send(*DownstreamMediaFrame) error
	CloseAndRecv() (*None, error)
	grpc.ClientStream
}

type downstreamMediaUploadClient struct {
	grpc.ClientStream
}

func (x *downstreamMediaUploadClient) Send(m *DownstreamMediaFrame) error {
	return x.ClientStream.SendMsg(m)
}

func (x *downstreamMediaUploadClient) CloseAndRecv() (*None, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(None)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DownstreamServer is the server API for Downstream service.
// All implementations must embed UnimplementedDownstreamServer
// for forward compatibility
type DownstreamServer interface {
	// Stream of commands from the server to the client
	Control(Downstream_ControlServer) error
	// Stream of media frames from the client to the server
	// There should be at most one long-standing call to MediaUpload per agent connected
	// to the cloud.
	MediaUpload(Downstream_MediaUploadServer) error
	mustEmbedUnimplementedDownstreamServer()
}

// UnimplementedDownstreamServer must be embedded to have forward compatible implementations.
type UnimplementedDownstreamServer struct {
}

func (UnimplementedDownstreamServer) Control(Downstream_ControlServer) error {
	return status.Errorf(codes.Unimplemented, "method Control not implemented")
}
func (UnimplementedDownstreamServer) MediaUpload(Downstream_MediaUploadServer) error {
	return status.Errorf(codes.Unimplemented, "method MediaUpload not implemented")
}
func (UnimplementedDownstreamServer) mustEmbedUnimplementedDownstreamServer() {}

// UnsafeDownstreamServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DownstreamServer will
// result in compilation errors.
type UnsafeDownstreamServer interface {
	mustEmbedUnimplementedDownstreamServer()
}

func RegisterDownstreamServer(s grpc.ServiceRegistrar, srv DownstreamServer) {
	s.RegisterService(&Downstream_ServiceDesc, srv)
}

func _Downstream_Control_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(DownstreamServer).Control(&downstreamControlServer{stream})
}

type Downstream_ControlServer interface {
	Send(*DownstreamControlRequest) error
	Recv() (*None, error)
	grpc.ServerStream
}

type downstreamControlServer struct {
	grpc.ServerStream
}

func (x *downstreamControlServer) Send(m *DownstreamControlRequest) error {
	return x.ServerStream.SendMsg(m)
}

func (x *downstreamControlServer) Recv() (*None, error) {
	m := new(None)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Downstream_MediaUpload_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(DownstreamServer).MediaUpload(&downstreamMediaUploadServer{stream})
}

type Downstream_MediaUploadServer interface {
	SendAndClose(*None) error
	Recv() (*DownstreamMediaFrame, error)
	grpc.ServerStream
}

type downstreamMediaUploadServer struct {
	grpc.ServerStream
}

func (x *downstreamMediaUploadServer) SendAndClose(m *None) error {
	return x.ServerStream.SendMsg(m)
}

func (x *downstreamMediaUploadServer) Recv() (*DownstreamMediaFrame, error) {
	m := new(DownstreamMediaFrame)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Downstream_ServiceDesc is the grpc.ServiceDesc for Downstream service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Downstream_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cams.api.hub.Downstream",
	HandlerType: (*DownstreamServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Control",
			Handler:       _Downstream_Control_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "MediaUpload",
			Handler:       _Downstream_MediaUpload_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "hub.proto",
}

// RegistrarClient is the client API for Registrar service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RegistrarClient interface {
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*None, error)
}

type registrarClient struct {
	cc grpc.ClientConnInterface
}

func NewRegistrarClient(cc grpc.ClientConnInterface) RegistrarClient {
	return &registrarClient{cc}
}

func (c *registrarClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*None, error) {
	out := new(None)
	err := c.cc.Invoke(ctx, "/cams.api.hub.Registrar/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RegistrarServer is the server API for Registrar service.
// All implementations must embed UnimplementedRegistrarServer
// for forward compatibility
type RegistrarServer interface {
	Register(context.Context, *RegisterRequest) (*None, error)
	mustEmbedUnimplementedRegistrarServer()
}

// UnimplementedRegistrarServer must be embedded to have forward compatible implementations.
type UnimplementedRegistrarServer struct {
}

func (UnimplementedRegistrarServer) Register(context.Context, *RegisterRequest) (*None, error) {
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
		FullMethod: "/cams.api.hub.Registrar/Register",
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
	ServiceName: "cams.api.hub.Registrar",
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

// ViewerClient is the client API for Viewer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ViewerClient interface {
	Play(ctx context.Context, in *PlayRequest, opts ...grpc.CallOption) (*None, error)
	Pause(ctx context.Context, in *PauseRequest, opts ...grpc.CallOption) (*None, error)
}

type viewerClient struct {
	cc grpc.ClientConnInterface
}

func NewViewerClient(cc grpc.ClientConnInterface) ViewerClient {
	return &viewerClient{cc}
}

func (c *viewerClient) Play(ctx context.Context, in *PlayRequest, opts ...grpc.CallOption) (*None, error) {
	out := new(None)
	err := c.cc.Invoke(ctx, "/cams.api.hub.Viewer/Play", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *viewerClient) Pause(ctx context.Context, in *PauseRequest, opts ...grpc.CallOption) (*None, error) {
	out := new(None)
	err := c.cc.Invoke(ctx, "/cams.api.hub.Viewer/Pause", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ViewerServer is the server API for Viewer service.
// All implementations must embed UnimplementedViewerServer
// for forward compatibility
type ViewerServer interface {
	Play(context.Context, *PlayRequest) (*None, error)
	Pause(context.Context, *PauseRequest) (*None, error)
	mustEmbedUnimplementedViewerServer()
}

// UnimplementedViewerServer must be embedded to have forward compatible implementations.
type UnimplementedViewerServer struct {
}

func (UnimplementedViewerServer) Play(context.Context, *PlayRequest) (*None, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Play not implemented")
}
func (UnimplementedViewerServer) Pause(context.Context, *PauseRequest) (*None, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Pause not implemented")
}
func (UnimplementedViewerServer) mustEmbedUnimplementedViewerServer() {}

// UnsafeViewerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ViewerServer will
// result in compilation errors.
type UnsafeViewerServer interface {
	mustEmbedUnimplementedViewerServer()
}

func RegisterViewerServer(s grpc.ServiceRegistrar, srv ViewerServer) {
	s.RegisterService(&Viewer_ServiceDesc, srv)
}

func _Viewer_Play_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ViewerServer).Play(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cams.api.hub.Viewer/Play",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ViewerServer).Play(ctx, req.(*PlayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Viewer_Pause_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PauseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ViewerServer).Pause(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cams.api.hub.Viewer/Pause",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ViewerServer).Pause(ctx, req.(*PauseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Viewer_ServiceDesc is the grpc.ServiceDesc for Viewer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Viewer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cams.api.hub.Viewer",
	HandlerType: (*ViewerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Play",
			Handler:    _Viewer_Play_Handler,
		},
		{
			MethodName: "Pause",
			Handler:    _Viewer_Pause_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "hub.proto",
}