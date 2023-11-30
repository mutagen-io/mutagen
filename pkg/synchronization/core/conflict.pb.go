// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.25.1
// source: synchronization/core/conflict.proto

package core

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

// Conflict encodes conflicting changes on alpha and beta that prevent
// synchronization of a particular path. Conflict objects should be considered
// immutable and must not be modified.
type Conflict struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Root is the root path for the conflict (relative to the synchronization
	// root). While this can (in theory) be computed based on the change lists
	// contained within the conflict, doing so relies on those change lists
	// being constructed and ordered in a particular manner that's not possible
	// to enforce. Additionally, conflicts are often sorted by their root path,
	// and dynamically computing it on every sort comparison operation would be
	// prohibitively expensive.
	Root string `protobuf:"bytes,1,opt,name=root,proto3" json:"root,omitempty"`
	// AlphaChanges are the relevant changes on alpha.
	AlphaChanges []*Change `protobuf:"bytes,2,rep,name=alphaChanges,proto3" json:"alphaChanges,omitempty"`
	// BetaChanges are the relevant changes on beta.
	BetaChanges []*Change `protobuf:"bytes,3,rep,name=betaChanges,proto3" json:"betaChanges,omitempty"`
}

func (x *Conflict) Reset() {
	*x = Conflict{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synchronization_core_conflict_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Conflict) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Conflict) ProtoMessage() {}

func (x *Conflict) ProtoReflect() protoreflect.Message {
	mi := &file_synchronization_core_conflict_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Conflict.ProtoReflect.Descriptor instead.
func (*Conflict) Descriptor() ([]byte, []int) {
	return file_synchronization_core_conflict_proto_rawDescGZIP(), []int{0}
}

func (x *Conflict) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

func (x *Conflict) GetAlphaChanges() []*Change {
	if x != nil {
		return x.AlphaChanges
	}
	return nil
}

func (x *Conflict) GetBetaChanges() []*Change {
	if x != nil {
		return x.BetaChanges
	}
	return nil
}

var File_synchronization_core_conflict_proto protoreflect.FileDescriptor

var file_synchronization_core_conflict_proto_rawDesc = []byte{
	0x0a, 0x23, 0x73, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x6c, 0x69, 0x63, 0x74, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x63, 0x6f, 0x72, 0x65, 0x1a, 0x21, 0x73, 0x79, 0x6e,
	0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x63, 0x6f, 0x72,
	0x65, 0x2f, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x80,
	0x01, 0x0a, 0x08, 0x43, 0x6f, 0x6e, 0x66, 0x6c, 0x69, 0x63, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x72,
	0x6f, 0x6f, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x12,
	0x30, 0x0a, 0x0c, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x68, 0x61,
	0x6e, 0x67, 0x65, 0x52, 0x0c, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65,
	0x73, 0x12, 0x2e, 0x0a, 0x0b, 0x62, 0x65, 0x74, 0x61, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x73,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x68,
	0x61, 0x6e, 0x67, 0x65, 0x52, 0x0b, 0x62, 0x65, 0x74, 0x61, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65,
	0x73, 0x42, 0x38, 0x5a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x6d, 0x75, 0x74, 0x61, 0x67, 0x65, 0x6e, 0x2d, 0x69, 0x6f, 0x2f, 0x6d, 0x75, 0x74, 0x61, 0x67,
	0x65, 0x6e, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x73, 0x79, 0x6e, 0x63, 0x68, 0x72, 0x6f, 0x6e, 0x69,
	0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_synchronization_core_conflict_proto_rawDescOnce sync.Once
	file_synchronization_core_conflict_proto_rawDescData = file_synchronization_core_conflict_proto_rawDesc
)

func file_synchronization_core_conflict_proto_rawDescGZIP() []byte {
	file_synchronization_core_conflict_proto_rawDescOnce.Do(func() {
		file_synchronization_core_conflict_proto_rawDescData = protoimpl.X.CompressGZIP(file_synchronization_core_conflict_proto_rawDescData)
	})
	return file_synchronization_core_conflict_proto_rawDescData
}

var file_synchronization_core_conflict_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_synchronization_core_conflict_proto_goTypes = []interface{}{
	(*Conflict)(nil), // 0: core.Conflict
	(*Change)(nil),   // 1: core.Change
}
var file_synchronization_core_conflict_proto_depIdxs = []int32{
	1, // 0: core.Conflict.alphaChanges:type_name -> core.Change
	1, // 1: core.Conflict.betaChanges:type_name -> core.Change
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_synchronization_core_conflict_proto_init() }
func file_synchronization_core_conflict_proto_init() {
	if File_synchronization_core_conflict_proto != nil {
		return
	}
	file_synchronization_core_change_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_synchronization_core_conflict_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Conflict); i {
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
			RawDescriptor: file_synchronization_core_conflict_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_synchronization_core_conflict_proto_goTypes,
		DependencyIndexes: file_synchronization_core_conflict_proto_depIdxs,
		MessageInfos:      file_synchronization_core_conflict_proto_msgTypes,
	}.Build()
	File_synchronization_core_conflict_proto = out.File
	file_synchronization_core_conflict_proto_rawDesc = nil
	file_synchronization_core_conflict_proto_goTypes = nil
	file_synchronization_core_conflict_proto_depIdxs = nil
}
