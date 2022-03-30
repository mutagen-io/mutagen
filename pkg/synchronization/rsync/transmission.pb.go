// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.4
// source: synchronization/rsync/transmission.proto

package rsync

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

// Transmission represents a single message in a transmission stream. As a
// Protocol Buffers message type, its internals are inherently public, but it
// should otherwise be treated as an opaque type with a private implementation.
type Transmission struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Done indicates that the operation stream for the current file is
	// finished. If set, there will be no operation in the response, but there
	// may be an error.
	Done bool `protobuf:"varint,1,opt,name=done,proto3" json:"done,omitempty"`
	// Operation is the next operation in the stream for the current file.
	Operation *Operation `protobuf:"bytes,2,opt,name=operation,proto3" json:"operation,omitempty"`
	// Error indicates that a non-terminal error has occurred. It will only be
	// present if Done is true.
	Error string `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *Transmission) Reset() {
	*x = Transmission{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synchronization_rsync_transmission_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Transmission) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Transmission) ProtoMessage() {}

func (x *Transmission) ProtoReflect() protoreflect.Message {
	mi := &file_synchronization_rsync_transmission_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Transmission.ProtoReflect.Descriptor instead.
func (*Transmission) Descriptor() ([]byte, []int) {
	return file_synchronization_rsync_transmission_proto_rawDescGZIP(), []int{0}
}

func (x *Transmission) GetDone() bool {
	if x != nil {
		return x.Done
	}
	return false
}

func (x *Transmission) GetOperation() *Operation {
	if x != nil {
		return x.Operation
	}
	return nil
}

func (x *Transmission) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

var File_synchronization_rsync_transmission_proto protoreflect.FileDescriptor

var file_synchronization_rsync_transmission_proto_rawDesc = []byte{
	0x0a, 0x28, 0x73, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x2f, 0x72, 0x73, 0x79, 0x6e, 0x63, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x72, 0x73, 0x79, 0x6e,
	0x63, 0x1a, 0x22, 0x73, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2f, 0x72, 0x73, 0x79, 0x6e, 0x63, 0x2f, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x68, 0x0a, 0x0c, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6d, 0x69,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x6f, 0x6e, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x04, 0x64, 0x6f, 0x6e, 0x65, 0x12, 0x2e, 0x0a, 0x09, 0x6f, 0x70, 0x65,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x72,
	0x73, 0x79, 0x6e, 0x63, 0x2e, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x09,
	0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72,
	0x6f, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x42,
	0x39, 0x5a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6d, 0x75,
	0x74, 0x61, 0x67, 0x65, 0x6e, 0x2d, 0x69, 0x6f, 0x2f, 0x6d, 0x75, 0x74, 0x61, 0x67, 0x65, 0x6e,
	0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x73, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x72, 0x73, 0x79, 0x6e, 0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_synchronization_rsync_transmission_proto_rawDescOnce sync.Once
	file_synchronization_rsync_transmission_proto_rawDescData = file_synchronization_rsync_transmission_proto_rawDesc
)

func file_synchronization_rsync_transmission_proto_rawDescGZIP() []byte {
	file_synchronization_rsync_transmission_proto_rawDescOnce.Do(func() {
		file_synchronization_rsync_transmission_proto_rawDescData = protoimpl.X.CompressGZIP(file_synchronization_rsync_transmission_proto_rawDescData)
	})
	return file_synchronization_rsync_transmission_proto_rawDescData
}

var file_synchronization_rsync_transmission_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_synchronization_rsync_transmission_proto_goTypes = []interface{}{
	(*Transmission)(nil), // 0: rsync.Transmission
	(*Operation)(nil),    // 1: rsync.Operation
}
var file_synchronization_rsync_transmission_proto_depIdxs = []int32{
	1, // 0: rsync.Transmission.operation:type_name -> rsync.Operation
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_synchronization_rsync_transmission_proto_init() }
func file_synchronization_rsync_transmission_proto_init() {
	if File_synchronization_rsync_transmission_proto != nil {
		return
	}
	file_synchronization_rsync_engine_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_synchronization_rsync_transmission_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Transmission); i {
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
			RawDescriptor: file_synchronization_rsync_transmission_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_synchronization_rsync_transmission_proto_goTypes,
		DependencyIndexes: file_synchronization_rsync_transmission_proto_depIdxs,
		MessageInfos:      file_synchronization_rsync_transmission_proto_msgTypes,
	}.Build()
	File_synchronization_rsync_transmission_proto = out.File
	file_synchronization_rsync_transmission_proto_rawDesc = nil
	file_synchronization_rsync_transmission_proto_goTypes = nil
	file_synchronization_rsync_transmission_proto_depIdxs = nil
}
