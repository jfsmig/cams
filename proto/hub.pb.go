// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.12.4
// source: hub.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Status struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Code   uint32 `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Status string `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *Status) Reset() {
	*x = Status{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Status) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Status) ProtoMessage() {}

func (x *Status) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Status.ProtoReflect.Descriptor instead.
func (*Status) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{0}
}

func (x *Status) GetCode() uint32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *Status) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

type MediaFrame struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Payload []byte       `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	Banner  *MediaBanner `protobuf:"bytes,2,opt,name=banner,proto3" json:"banner,omitempty"`
}

func (x *MediaFrame) Reset() {
	*x = MediaFrame{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MediaFrame) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MediaFrame) ProtoMessage() {}

func (x *MediaFrame) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MediaFrame.ProtoReflect.Descriptor instead.
func (*MediaFrame) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{1}
}

func (x *MediaFrame) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *MediaFrame) GetBanner() *MediaBanner {
	if x != nil {
		return x.Banner
	}
	return nil
}

type MediaReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id     *StreamId `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Status *Status   `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *MediaReply) Reset() {
	*x = MediaReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MediaReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MediaReply) ProtoMessage() {}

func (x *MediaReply) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MediaReply.ProtoReflect.Descriptor instead.
func (*MediaReply) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{2}
}

func (x *MediaReply) GetId() *StreamId {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *MediaReply) GetStatus() *Status {
	if x != nil {
		return x.Status
	}
	return nil
}

type MediaBanner struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	User   string `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	Stream string `protobuf:"bytes,2,opt,name=stream,proto3" json:"stream,omitempty"`
	Token  string `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *MediaBanner) Reset() {
	*x = MediaBanner{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MediaBanner) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MediaBanner) ProtoMessage() {}

func (x *MediaBanner) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MediaBanner.ProtoReflect.Descriptor instead.
func (*MediaBanner) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{3}
}

func (x *MediaBanner) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *MediaBanner) GetStream() string {
	if x != nil {
		return x.Stream
	}
	return ""
}

func (x *MediaBanner) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type RegisterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id *StreamId `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *RegisterRequest) Reset() {
	*x = RegisterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RegisterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterRequest) ProtoMessage() {}

func (x *RegisterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterRequest.ProtoReflect.Descriptor instead.
func (*RegisterRequest) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{4}
}

func (x *RegisterRequest) GetId() *StreamId {
	if x != nil {
		return x.Id
	}
	return nil
}

type RegisterReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status *Status `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *RegisterReply) Reset() {
	*x = RegisterReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RegisterReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterReply) ProtoMessage() {}

func (x *RegisterReply) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterReply.ProtoReflect.Descriptor instead.
func (*RegisterReply) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{5}
}

func (x *RegisterReply) GetStatus() *Status {
	if x != nil {
		return x.Status
	}
	return nil
}

type StreamId struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	User   string `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	Stream string `protobuf:"bytes,2,opt,name=stream,proto3" json:"stream,omitempty"`
}

func (x *StreamId) Reset() {
	*x = StreamId{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StreamId) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StreamId) ProtoMessage() {}

func (x *StreamId) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StreamId.ProtoReflect.Descriptor instead.
func (*StreamId) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{6}
}

func (x *StreamId) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *StreamId) GetStream() string {
	if x != nil {
		return x.Stream
	}
	return ""
}

type ControlReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Playing *RtspPlayNotify `protobuf:"bytes,1,opt,name=playing,proto3" json:"playing,omitempty"`
}

func (x *ControlReply) Reset() {
	*x = ControlReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ControlReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ControlReply) ProtoMessage() {}

func (x *ControlReply) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ControlReply.ProtoReflect.Descriptor instead.
func (*ControlReply) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{7}
}

func (x *ControlReply) GetPlaying() *RtspPlayNotify {
	if x != nil {
		return x.Playing
	}
	return nil
}

type ControlRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Play     *RtspPlay     `protobuf:"bytes,1,opt,name=play,proto3" json:"play,omitempty"`
	Pause    *RtspPause    `protobuf:"bytes,2,opt,name=pause,proto3" json:"pause,omitempty"`
	Teardown *RtspTeardown `protobuf:"bytes,3,opt,name=teardown,proto3" json:"teardown,omitempty"`
}

func (x *ControlRequest) Reset() {
	*x = ControlRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ControlRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ControlRequest) ProtoMessage() {}

func (x *ControlRequest) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ControlRequest.ProtoReflect.Descriptor instead.
func (*ControlRequest) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{8}
}

func (x *ControlRequest) GetPlay() *RtspPlay {
	if x != nil {
		return x.Play
	}
	return nil
}

func (x *ControlRequest) GetPause() *RtspPause {
	if x != nil {
		return x.Pause
	}
	return nil
}

func (x *ControlRequest) GetTeardown() *RtspTeardown {
	if x != nil {
		return x.Teardown
	}
	return nil
}

type RtspPlayNotify struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamID string `protobuf:"bytes,1,opt,name=streamID,proto3" json:"streamID,omitempty"`
}

func (x *RtspPlayNotify) Reset() {
	*x = RtspPlayNotify{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RtspPlayNotify) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RtspPlayNotify) ProtoMessage() {}

func (x *RtspPlayNotify) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RtspPlayNotify.ProtoReflect.Descriptor instead.
func (*RtspPlayNotify) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{9}
}

func (x *RtspPlayNotify) GetStreamID() string {
	if x != nil {
		return x.StreamID
	}
	return ""
}

type RtspPlay struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamID string `protobuf:"bytes,1,opt,name=streamID,proto3" json:"streamID,omitempty"`
}

func (x *RtspPlay) Reset() {
	*x = RtspPlay{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RtspPlay) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RtspPlay) ProtoMessage() {}

func (x *RtspPlay) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RtspPlay.ProtoReflect.Descriptor instead.
func (*RtspPlay) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{10}
}

func (x *RtspPlay) GetStreamID() string {
	if x != nil {
		return x.StreamID
	}
	return ""
}

type RtspPause struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamID string `protobuf:"bytes,1,opt,name=streamID,proto3" json:"streamID,omitempty"`
}

func (x *RtspPause) Reset() {
	*x = RtspPause{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RtspPause) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RtspPause) ProtoMessage() {}

func (x *RtspPause) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RtspPause.ProtoReflect.Descriptor instead.
func (*RtspPause) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{11}
}

func (x *RtspPause) GetStreamID() string {
	if x != nil {
		return x.StreamID
	}
	return ""
}

type RtspTeardown struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StreamID string `protobuf:"bytes,1,opt,name=streamID,proto3" json:"streamID,omitempty"`
}

func (x *RtspTeardown) Reset() {
	*x = RtspTeardown{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hub_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RtspTeardown) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RtspTeardown) ProtoMessage() {}

func (x *RtspTeardown) ProtoReflect() protoreflect.Message {
	mi := &file_hub_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RtspTeardown.ProtoReflect.Descriptor instead.
func (*RtspTeardown) Descriptor() ([]byte, []int) {
	return file_hub_proto_rawDescGZIP(), []int{12}
}

func (x *RtspTeardown) GetStreamID() string {
	if x != nil {
		return x.StreamID
	}
	return ""
}

var File_hub_proto protoreflect.FileDescriptor

var file_hub_proto_rawDesc = []byte{
	0x0a, 0x09, 0x68, 0x75, 0x62, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x63, 0x61, 0x6d,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x34, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x57, 0x0a,
	0x0a, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x70,
	0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61,
	0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x2f, 0x0a, 0x06, 0x62, 0x61, 0x6e, 0x6e, 0x65, 0x72, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x42, 0x61, 0x6e, 0x6e, 0x65, 0x72, 0x52, 0x06,
	0x62, 0x61, 0x6e, 0x6e, 0x65, 0x72, 0x22, 0x5e, 0x0a, 0x0a, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x52,
	0x65, 0x70, 0x6c, 0x79, 0x12, 0x24, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x14, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x74,
	0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x52, 0x02, 0x69, 0x64, 0x12, 0x2a, 0x0a, 0x06, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x63, 0x61, 0x6d,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06,
	0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x4f, 0x0a, 0x0b, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x42,
	0x61, 0x6e, 0x6e, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x37, 0x0a, 0x0f, 0x52, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x24, 0x0a, 0x02, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x52, 0x02, 0x69, 0x64,
	0x22, 0x3b, 0x0a, 0x0d, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x70, 0x6c,
	0x79, 0x12, 0x2a, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x12, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x36, 0x0a,
	0x08, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x75, 0x73, 0x65,
	0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x12, 0x16, 0x0a,
	0x06, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x22, 0x44, 0x0a, 0x0c, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x34, 0x0a, 0x07, 0x70, 0x6c, 0x61, 0x79, 0x69, 0x6e, 0x67,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x74, 0x73, 0x70, 0x50, 0x6c, 0x61, 0x79, 0x4e, 0x6f, 0x74, 0x69,
	0x66, 0x79, 0x52, 0x07, 0x70, 0x6c, 0x61, 0x79, 0x69, 0x6e, 0x67, 0x22, 0x9d, 0x01, 0x0a, 0x0e,
	0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x28,
	0x0a, 0x04, 0x70, 0x6c, 0x61, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x63,
	0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x74, 0x73, 0x70, 0x50, 0x6c,
	0x61, 0x79, 0x52, 0x04, 0x70, 0x6c, 0x61, 0x79, 0x12, 0x2b, 0x0a, 0x05, 0x70, 0x61, 0x75, 0x73,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x74, 0x73, 0x70, 0x50, 0x61, 0x75, 0x73, 0x65, 0x52, 0x05,
	0x70, 0x61, 0x75, 0x73, 0x65, 0x12, 0x34, 0x0a, 0x08, 0x74, 0x65, 0x61, 0x72, 0x64, 0x6f, 0x77,
	0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x74, 0x73, 0x70, 0x54, 0x65, 0x61, 0x72, 0x64, 0x6f, 0x77,
	0x6e, 0x52, 0x08, 0x74, 0x65, 0x61, 0x72, 0x64, 0x6f, 0x77, 0x6e, 0x22, 0x2c, 0x0a, 0x0e, 0x52,
	0x74, 0x73, 0x70, 0x50, 0x6c, 0x61, 0x79, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x12, 0x1a, 0x0a,
	0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x44, 0x22, 0x26, 0x0a, 0x08, 0x52, 0x74, 0x73,
	0x70, 0x50, 0x6c, 0x61, 0x79, 0x12, 0x1a, 0x0a, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49,
	0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49,
	0x44, 0x22, 0x27, 0x0a, 0x09, 0x52, 0x74, 0x73, 0x70, 0x50, 0x61, 0x75, 0x73, 0x65, 0x12, 0x1a,
	0x0a, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x44, 0x22, 0x2a, 0x0a, 0x0c, 0x52, 0x74,
	0x73, 0x70, 0x54, 0x65, 0x61, 0x72, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x73, 0x74,
	0x72, 0x65, 0x61, 0x6d, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x73, 0x74,
	0x72, 0x65, 0x61, 0x6d, 0x49, 0x44, 0x32, 0x51, 0x0a, 0x09, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x61, 0x72, 0x12, 0x44, 0x0a, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x12,
	0x1b, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x63,
	0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x65, 0x72, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x32, 0x96, 0x01, 0x0a, 0x0a, 0x43, 0x6f,
	0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x12, 0x45, 0x0a, 0x07, 0x43, 0x6f, 0x6e, 0x74,
	0x72, 0x6f, 0x6c, 0x12, 0x18, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x1a, 0x1a, 0x2e,
	0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72,
	0x6f, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x12,
	0x41, 0x0a, 0x0b, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x16,
	0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4d, 0x65, 0x64, 0x69,
	0x61, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x1a, 0x16, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00,
	0x28, 0x01, 0x32, 0x44, 0x0a, 0x08, 0x43, 0x6f, 0x6e, 0x73, 0x75, 0x6d, 0x65, 0x72, 0x12, 0x38,
	0x0a, 0x04, 0x50, 0x6c, 0x61, 0x79, 0x12, 0x14, 0x2e, 0x63, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x1a, 0x16, 0x2e, 0x63,
	0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4d, 0x65, 0x64, 0x69, 0x61, 0x46,
	0x72, 0x61, 0x6d, 0x65, 0x22, 0x00, 0x30, 0x01, 0x42, 0x09, 0x5a, 0x07, 0x2e, 0x3b, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_hub_proto_rawDescOnce sync.Once
	file_hub_proto_rawDescData = file_hub_proto_rawDesc
)

func file_hub_proto_rawDescGZIP() []byte {
	file_hub_proto_rawDescOnce.Do(func() {
		file_hub_proto_rawDescData = protoimpl.X.CompressGZIP(file_hub_proto_rawDescData)
	})
	return file_hub_proto_rawDescData
}

var file_hub_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_hub_proto_goTypes = []interface{}{
	(*Status)(nil),          // 0: cams.proto.Status
	(*MediaFrame)(nil),      // 1: cams.proto.MediaFrame
	(*MediaReply)(nil),      // 2: cams.proto.MediaReply
	(*MediaBanner)(nil),     // 3: cams.proto.MediaBanner
	(*RegisterRequest)(nil), // 4: cams.proto.RegisterRequest
	(*RegisterReply)(nil),   // 5: cams.proto.RegisterReply
	(*StreamId)(nil),        // 6: cams.proto.StreamId
	(*ControlReply)(nil),    // 7: cams.proto.ControlReply
	(*ControlRequest)(nil),  // 8: cams.proto.ControlRequest
	(*RtspPlayNotify)(nil),  // 9: cams.proto.RtspPlayNotify
	(*RtspPlay)(nil),        // 10: cams.proto.RtspPlay
	(*RtspPause)(nil),       // 11: cams.proto.RtspPause
	(*RtspTeardown)(nil),    // 12: cams.proto.RtspTeardown
}
var file_hub_proto_depIdxs = []int32{
	3,  // 0: cams.proto.MediaFrame.banner:type_name -> cams.proto.MediaBanner
	6,  // 1: cams.proto.MediaReply.id:type_name -> cams.proto.StreamId
	0,  // 2: cams.proto.MediaReply.status:type_name -> cams.proto.Status
	6,  // 3: cams.proto.RegisterRequest.id:type_name -> cams.proto.StreamId
	0,  // 4: cams.proto.RegisterReply.status:type_name -> cams.proto.Status
	9,  // 5: cams.proto.ControlReply.playing:type_name -> cams.proto.RtspPlayNotify
	10, // 6: cams.proto.ControlRequest.play:type_name -> cams.proto.RtspPlay
	11, // 7: cams.proto.ControlRequest.pause:type_name -> cams.proto.RtspPause
	12, // 8: cams.proto.ControlRequest.teardown:type_name -> cams.proto.RtspTeardown
	4,  // 9: cams.proto.Registrar.Register:input_type -> cams.proto.RegisterRequest
	7,  // 10: cams.proto.Controller.Control:input_type -> cams.proto.ControlReply
	1,  // 11: cams.proto.Controller.MediaUpload:input_type -> cams.proto.MediaFrame
	6,  // 12: cams.proto.Consumer.Play:input_type -> cams.proto.StreamId
	5,  // 13: cams.proto.Registrar.Register:output_type -> cams.proto.RegisterReply
	8,  // 14: cams.proto.Controller.Control:output_type -> cams.proto.ControlRequest
	2,  // 15: cams.proto.Controller.MediaUpload:output_type -> cams.proto.MediaReply
	1,  // 16: cams.proto.Consumer.Play:output_type -> cams.proto.MediaFrame
	13, // [13:17] is the sub-list for method output_type
	9,  // [9:13] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_hub_proto_init() }
func file_hub_proto_init() {
	if File_hub_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_hub_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Status); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MediaFrame); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MediaReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MediaBanner); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RegisterRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RegisterReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StreamId); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ControlReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ControlRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RtspPlayNotify); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RtspPlay); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RtspPause); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_hub_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RtspTeardown); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_hub_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   3,
		},
		GoTypes:           file_hub_proto_goTypes,
		DependencyIndexes: file_hub_proto_depIdxs,
		MessageInfos:      file_hub_proto_msgTypes,
	}.Build()
	File_hub_proto = out.File
	file_hub_proto_rawDesc = nil
	file_hub_proto_goTypes = nil
	file_hub_proto_depIdxs = nil
}