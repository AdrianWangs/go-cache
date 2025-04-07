// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: cache_server.proto

package cache_server

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Request struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Group         string                 `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"` // 组名
	Key           string                 `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`     // 键
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Request) Reset() {
	*x = Request{}
	mi := &file_cache_server_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Request) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request) ProtoMessage() {}

func (x *Request) ProtoReflect() protoreflect.Message {
	mi := &file_cache_server_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request.ProtoReflect.Descriptor instead.
func (*Request) Descriptor() ([]byte, []int) {
	return file_cache_server_proto_rawDescGZIP(), []int{0}
}

func (x *Request) GetGroup() string {
	if x != nil {
		return x.Group
	}
	return ""
}

func (x *Request) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type Response struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         []byte                 `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"` // 值
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Response) Reset() {
	*x = Response{}
	mi := &file_cache_server_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_cache_server_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response.ProtoReflect.Descriptor instead.
func (*Response) Descriptor() ([]byte, []int) {
	return file_cache_server_proto_rawDescGZIP(), []int{1}
}

func (x *Response) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

type DeleteRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Group         string                 `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"` // 组名
	Key           string                 `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`     // 键
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteRequest) Reset() {
	*x = DeleteRequest{}
	mi := &file_cache_server_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRequest) ProtoMessage() {}

func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_cache_server_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteRequest.ProtoReflect.Descriptor instead.
func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return file_cache_server_proto_rawDescGZIP(), []int{2}
}

func (x *DeleteRequest) GetGroup() string {
	if x != nil {
		return x.Group
	}
	return ""
}

func (x *DeleteRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type DeleteResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"` // 是否成功
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteResponse) Reset() {
	*x = DeleteResponse{}
	mi := &file_cache_server_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteResponse) ProtoMessage() {}

func (x *DeleteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_cache_server_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteResponse.ProtoReflect.Descriptor instead.
func (*DeleteResponse) Descriptor() ([]byte, []int) {
	return file_cache_server_proto_rawDescGZIP(), []int{3}
}

func (x *DeleteResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

var File_cache_server_proto protoreflect.FileDescriptor

const file_cache_server_proto_rawDesc = "" +
	"\n" +
	"\x12cache_server.proto\x12\bgo_cache\"1\n" +
	"\aRequest\x12\x14\n" +
	"\x05group\x18\x01 \x01(\tR\x05group\x12\x10\n" +
	"\x03key\x18\x02 \x01(\tR\x03key\" \n" +
	"\bResponse\x12\x14\n" +
	"\x05value\x18\x01 \x01(\fR\x05value\"7\n" +
	"\rDeleteRequest\x12\x14\n" +
	"\x05group\x18\x01 \x01(\tR\x05group\x12\x10\n" +
	"\x03key\x18\x02 \x01(\tR\x03key\"*\n" +
	"\x0eDeleteResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess2w\n" +
	"\n" +
	"GroupCache\x12,\n" +
	"\x03Get\x12\x11.go_cache.Request\x1a\x12.go_cache.Response\x12;\n" +
	"\x06Delete\x12\x17.go_cache.DeleteRequest\x1a\x18.go_cache.DeleteResponseB\x10Z\x0e./cache_serverb\x06proto3"

var (
	file_cache_server_proto_rawDescOnce sync.Once
	file_cache_server_proto_rawDescData []byte
)

func file_cache_server_proto_rawDescGZIP() []byte {
	file_cache_server_proto_rawDescOnce.Do(func() {
		file_cache_server_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_cache_server_proto_rawDesc), len(file_cache_server_proto_rawDesc)))
	})
	return file_cache_server_proto_rawDescData
}

var file_cache_server_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_cache_server_proto_goTypes = []any{
	(*Request)(nil),        // 0: go_cache.Request
	(*Response)(nil),       // 1: go_cache.Response
	(*DeleteRequest)(nil),  // 2: go_cache.DeleteRequest
	(*DeleteResponse)(nil), // 3: go_cache.DeleteResponse
}
var file_cache_server_proto_depIdxs = []int32{
	0, // 0: go_cache.GroupCache.Get:input_type -> go_cache.Request
	2, // 1: go_cache.GroupCache.Delete:input_type -> go_cache.DeleteRequest
	1, // 2: go_cache.GroupCache.Get:output_type -> go_cache.Response
	3, // 3: go_cache.GroupCache.Delete:output_type -> go_cache.DeleteResponse
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_cache_server_proto_init() }
func file_cache_server_proto_init() {
	if File_cache_server_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_cache_server_proto_rawDesc), len(file_cache_server_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_cache_server_proto_goTypes,
		DependencyIndexes: file_cache_server_proto_depIdxs,
		MessageInfos:      file_cache_server_proto_msgTypes,
	}.Build()
	File_cache_server_proto = out.File
	file_cache_server_proto_goTypes = nil
	file_cache_server_proto_depIdxs = nil
}
