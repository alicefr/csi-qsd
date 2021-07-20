// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.14.0
// source: pkg/qsd/qsd.proto

package qsd

import (
	proto "github.com/golang/protobuf/proto"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type Image struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID   string `protobuf:"bytes,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Size int64  `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
}

func (x *Image) Reset() {
	*x = Image{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_qsd_qsd_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Image) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Image) ProtoMessage() {}

func (x *Image) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_qsd_qsd_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Image.ProtoReflect.Descriptor instead.
func (*Image) Descriptor() ([]byte, []int) {
	return file_pkg_qsd_qsd_proto_rawDescGZIP(), []int{0}
}

func (x *Image) GetID() string {
	if x != nil {
		return x.ID
	}
	return ""
}

func (x *Image) GetSize() int64 {
	if x != nil {
		return x.Size
	}
	return 0
}

type Response struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *Response) Reset() {
	*x = Response{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_qsd_qsd_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_qsd_qsd_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
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
	return file_pkg_qsd_qsd_proto_rawDescGZIP(), []int{1}
}

func (x *Response) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *Response) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_pkg_qsd_qsd_proto protoreflect.FileDescriptor

var file_pkg_qsd_qsd_proto_rawDesc = []byte{
	0x0a, 0x11, 0x70, 0x6b, 0x67, 0x2f, 0x71, 0x73, 0x64, 0x2f, 0x71, 0x73, 0x64, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x13, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x66, 0x72, 0x2e, 0x63, 0x73, 0x69,
	0x2e, 0x70, 0x6b, 0x67, 0x2e, 0x71, 0x73, 0x64, 0x22, 0x2b, 0x0a, 0x05, 0x49, 0x6d, 0x61, 0x67,
	0x65, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x49,
	0x44, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x04, 0x73, 0x69, 0x7a, 0x65, 0x22, 0x3e, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0xa9, 0x01, 0x0a, 0x0a, 0x51, 0x73, 0x64, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x4b, 0x0a, 0x0c, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x56, 0x6f,
	0x6c, 0x75, 0x6d, 0x65, 0x12, 0x1a, 0x2e, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x66, 0x72, 0x2e, 0x63,
	0x73, 0x69, 0x2e, 0x70, 0x6b, 0x67, 0x2e, 0x71, 0x73, 0x64, 0x2e, 0x49, 0x6d, 0x61, 0x67, 0x65,
	0x1a, 0x1d, 0x2e, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x66, 0x72, 0x2e, 0x63, 0x73, 0x69, 0x2e, 0x70,
	0x6b, 0x67, 0x2e, 0x71, 0x73, 0x64, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x12, 0x4e, 0x0a, 0x0f, 0x45, 0x78, 0x70, 0x6f, 0x73, 0x65, 0x56, 0x68, 0x6f, 0x73, 0x74,
	0x55, 0x73, 0x65, 0x72, 0x12, 0x1a, 0x2e, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x66, 0x72, 0x2e, 0x63,
	0x73, 0x69, 0x2e, 0x70, 0x6b, 0x67, 0x2e, 0x71, 0x73, 0x64, 0x2e, 0x49, 0x6d, 0x61, 0x67, 0x65,
	0x1a, 0x1d, 0x2e, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x66, 0x72, 0x2e, 0x63, 0x73, 0x69, 0x2e, 0x70,
	0x6b, 0x67, 0x2e, 0x71, 0x73, 0x64, 0x2e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x42, 0x24, 0x5a, 0x22, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x61, 0x6c, 0x69, 0x63, 0x65, 0x66, 0x72, 0x2f, 0x63, 0x73, 0x69, 0x2d, 0x71, 0x73, 0x64, 0x2f,
	0x70, 0x6b, 0x67, 0x2f, 0x71, 0x73, 0x64, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_qsd_qsd_proto_rawDescOnce sync.Once
	file_pkg_qsd_qsd_proto_rawDescData = file_pkg_qsd_qsd_proto_rawDesc
)

func file_pkg_qsd_qsd_proto_rawDescGZIP() []byte {
	file_pkg_qsd_qsd_proto_rawDescOnce.Do(func() {
		file_pkg_qsd_qsd_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_qsd_qsd_proto_rawDescData)
	})
	return file_pkg_qsd_qsd_proto_rawDescData
}

var file_pkg_qsd_qsd_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_pkg_qsd_qsd_proto_goTypes = []interface{}{
	(*Image)(nil),    // 0: alicefr.csi.pkg.qsd.Image
	(*Response)(nil), // 1: alicefr.csi.pkg.qsd.Response
}
var file_pkg_qsd_qsd_proto_depIdxs = []int32{
	0, // 0: alicefr.csi.pkg.qsd.QsdService.CreateVolume:input_type -> alicefr.csi.pkg.qsd.Image
	0, // 1: alicefr.csi.pkg.qsd.QsdService.ExposeVhostUser:input_type -> alicefr.csi.pkg.qsd.Image
	1, // 2: alicefr.csi.pkg.qsd.QsdService.CreateVolume:output_type -> alicefr.csi.pkg.qsd.Response
	1, // 3: alicefr.csi.pkg.qsd.QsdService.ExposeVhostUser:output_type -> alicefr.csi.pkg.qsd.Response
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_pkg_qsd_qsd_proto_init() }
func file_pkg_qsd_qsd_proto_init() {
	if File_pkg_qsd_qsd_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_qsd_qsd_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Image); i {
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
		file_pkg_qsd_qsd_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response); i {
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
			RawDescriptor: file_pkg_qsd_qsd_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pkg_qsd_qsd_proto_goTypes,
		DependencyIndexes: file_pkg_qsd_qsd_proto_depIdxs,
		MessageInfos:      file_pkg_qsd_qsd_proto_msgTypes,
	}.Build()
	File_pkg_qsd_qsd_proto = out.File
	file_pkg_qsd_qsd_proto_rawDesc = nil
	file_pkg_qsd_qsd_proto_goTypes = nil
	file_pkg_qsd_qsd_proto_depIdxs = nil
}
