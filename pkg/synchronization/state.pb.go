// Code generated by protoc-gen-go. DO NOT EDIT.
// source: synchronization/state.proto

package synchronization

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	core "github.com/mutagen-io/mutagen/pkg/synchronization/core"
	rsync "github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Status int32

const (
	Status_Disconnected           Status = 0
	Status_HaltedOnRootDeletion   Status = 1
	Status_HaltedOnRootTypeChange Status = 2
	Status_ConnectingAlpha        Status = 3
	Status_ConnectingBeta         Status = 4
	Status_Watching               Status = 5
	Status_Scanning               Status = 6
	Status_WaitingForRescan       Status = 7
	Status_Reconciling            Status = 8
	Status_StagingAlpha           Status = 9
	Status_StagingBeta            Status = 10
	Status_Transitioning          Status = 11
	Status_Saving                 Status = 12
)

var Status_name = map[int32]string{
	0:  "Disconnected",
	1:  "HaltedOnRootDeletion",
	2:  "HaltedOnRootTypeChange",
	3:  "ConnectingAlpha",
	4:  "ConnectingBeta",
	5:  "Watching",
	6:  "Scanning",
	7:  "WaitingForRescan",
	8:  "Reconciling",
	9:  "StagingAlpha",
	10: "StagingBeta",
	11: "Transitioning",
	12: "Saving",
}

var Status_value = map[string]int32{
	"Disconnected":           0,
	"HaltedOnRootDeletion":   1,
	"HaltedOnRootTypeChange": 2,
	"ConnectingAlpha":        3,
	"ConnectingBeta":         4,
	"Watching":               5,
	"Scanning":               6,
	"WaitingForRescan":       7,
	"Reconciling":            8,
	"StagingAlpha":           9,
	"StagingBeta":            10,
	"Transitioning":          11,
	"Saving":                 12,
}

func (x Status) String() string {
	return proto.EnumName(Status_name, int32(x))
}

func (Status) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8699c6f4e92f6557, []int{0}
}

type State struct {
	Session                         *Session              `protobuf:"bytes,1,opt,name=session,proto3" json:"session,omitempty"`
	Status                          Status                `protobuf:"varint,2,opt,name=status,proto3,enum=synchronization.Status" json:"status,omitempty"`
	AlphaConnected                  bool                  `protobuf:"varint,3,opt,name=alphaConnected,proto3" json:"alphaConnected,omitempty"`
	BetaConnected                   bool                  `protobuf:"varint,4,opt,name=betaConnected,proto3" json:"betaConnected,omitempty"`
	LastError                       string                `protobuf:"bytes,5,opt,name=lastError,proto3" json:"lastError,omitempty"`
	SuccessfulSynchronizationCycles uint64                `protobuf:"varint,6,opt,name=successfulSynchronizationCycles,proto3" json:"successfulSynchronizationCycles,omitempty"`
	StagingStatus                   *rsync.ReceiverStatus `protobuf:"bytes,7,opt,name=stagingStatus,proto3" json:"stagingStatus,omitempty"`
	Conflicts                       []*core.Conflict      `protobuf:"bytes,8,rep,name=conflicts,proto3" json:"conflicts,omitempty"`
	AlphaProblems                   []*core.Problem       `protobuf:"bytes,9,rep,name=alphaProblems,proto3" json:"alphaProblems,omitempty"`
	BetaProblems                    []*core.Problem       `protobuf:"bytes,10,rep,name=betaProblems,proto3" json:"betaProblems,omitempty"`
	XXX_NoUnkeyedLiteral            struct{}              `json:"-"`
	XXX_unrecognized                []byte                `json:"-"`
	XXX_sizecache                   int32                 `json:"-"`
}

func (m *State) Reset()         { *m = State{} }
func (m *State) String() string { return proto.CompactTextString(m) }
func (*State) ProtoMessage()    {}
func (*State) Descriptor() ([]byte, []int) {
	return fileDescriptor_8699c6f4e92f6557, []int{0}
}

func (m *State) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_State.Unmarshal(m, b)
}
func (m *State) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_State.Marshal(b, m, deterministic)
}
func (m *State) XXX_Merge(src proto.Message) {
	xxx_messageInfo_State.Merge(m, src)
}
func (m *State) XXX_Size() int {
	return xxx_messageInfo_State.Size(m)
}
func (m *State) XXX_DiscardUnknown() {
	xxx_messageInfo_State.DiscardUnknown(m)
}

var xxx_messageInfo_State proto.InternalMessageInfo

func (m *State) GetSession() *Session {
	if m != nil {
		return m.Session
	}
	return nil
}

func (m *State) GetStatus() Status {
	if m != nil {
		return m.Status
	}
	return Status_Disconnected
}

func (m *State) GetAlphaConnected() bool {
	if m != nil {
		return m.AlphaConnected
	}
	return false
}

func (m *State) GetBetaConnected() bool {
	if m != nil {
		return m.BetaConnected
	}
	return false
}

func (m *State) GetLastError() string {
	if m != nil {
		return m.LastError
	}
	return ""
}

func (m *State) GetSuccessfulSynchronizationCycles() uint64 {
	if m != nil {
		return m.SuccessfulSynchronizationCycles
	}
	return 0
}

func (m *State) GetStagingStatus() *rsync.ReceiverStatus {
	if m != nil {
		return m.StagingStatus
	}
	return nil
}

func (m *State) GetConflicts() []*core.Conflict {
	if m != nil {
		return m.Conflicts
	}
	return nil
}

func (m *State) GetAlphaProblems() []*core.Problem {
	if m != nil {
		return m.AlphaProblems
	}
	return nil
}

func (m *State) GetBetaProblems() []*core.Problem {
	if m != nil {
		return m.BetaProblems
	}
	return nil
}

func init() {
	proto.RegisterEnum("synchronization.Status", Status_name, Status_value)
	proto.RegisterType((*State)(nil), "synchronization.State")
}

func init() { proto.RegisterFile("synchronization/state.proto", fileDescriptor_8699c6f4e92f6557) }

var fileDescriptor_8699c6f4e92f6557 = []byte{
	// 522 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x93, 0x5f, 0x6f, 0xd3, 0x30,
	0x14, 0xc5, 0xc9, 0xba, 0xa6, 0xed, 0xed, 0xbf, 0x70, 0x19, 0x10, 0x15, 0x10, 0xd1, 0x40, 0x28,
	0x42, 0x90, 0x68, 0xed, 0x23, 0x4f, 0xac, 0x03, 0xed, 0x0d, 0x94, 0x4c, 0x9a, 0xc4, 0x9b, 0xeb,
	0x79, 0xa9, 0x45, 0x6a, 0x57, 0xb6, 0x3b, 0xa9, 0x7c, 0x6f, 0x5e, 0x11, 0x72, 0xe2, 0xd2, 0xb5,
	0x9d, 0xc4, 0x9b, 0x7d, 0xee, 0xef, 0xd8, 0xd7, 0xf7, 0x24, 0xf0, 0x42, 0xaf, 0x05, 0x9d, 0x2b,
	0x29, 0xf8, 0x2f, 0x62, 0xb8, 0x14, 0xa9, 0x36, 0xc4, 0xb0, 0x64, 0xa9, 0xa4, 0x91, 0x38, 0xdc,
	0x2b, 0x8e, 0xde, 0xec, 0xd3, 0xca, 0x0a, 0xa9, 0x62, 0x94, 0xf1, 0x3b, 0xe7, 0x1a, 0xbd, 0x3a,
	0x38, 0x92, 0x69, 0xcd, 0xa5, 0x70, 0xe5, 0x83, 0x33, 0xa8, 0x54, 0x2c, 0xa5, 0x52, 0xdc, 0x96,
	0x9c, 0x1a, 0x07, 0x9d, 0x3e, 0x08, 0x2d, 0x95, 0x9c, 0x95, 0x6c, 0x51, 0x33, 0xa7, 0xbf, 0x1b,
	0xd0, 0xcc, 0x6d, 0xb7, 0x38, 0x86, 0x96, 0xbb, 0x23, 0xf4, 0x22, 0x2f, 0xee, 0x8e, 0xc3, 0x64,
	0xcf, 0x9f, 0xe4, 0x75, 0x3d, 0xdb, 0x80, 0x98, 0x82, 0x6f, 0x9f, 0xba, 0xd2, 0xe1, 0x51, 0xe4,
	0xc5, 0x83, 0xf1, 0xf3, 0x43, 0x4b, 0x55, 0xce, 0x1c, 0x86, 0xef, 0x60, 0x40, 0xca, 0xe5, 0x9c,
	0x4c, 0xa5, 0x10, 0x8c, 0x1a, 0x76, 0x13, 0x36, 0x22, 0x2f, 0x6e, 0x67, 0x7b, 0x2a, 0xbe, 0x85,
	0xfe, 0x8c, 0x99, 0x7b, 0xd8, 0x71, 0x85, 0xed, 0x8a, 0xf8, 0x12, 0x3a, 0x25, 0xd1, 0xe6, 0x8b,
	0x52, 0x52, 0x85, 0xcd, 0xc8, 0x8b, 0x3b, 0xd9, 0x56, 0xc0, 0x4b, 0x78, 0xad, 0x57, 0x94, 0x32,
	0xad, 0x6f, 0x57, 0x65, 0xbe, 0xdb, 0xd7, 0x74, 0x4d, 0x4b, 0xa6, 0x43, 0x3f, 0xf2, 0xe2, 0xe3,
	0xec, 0x7f, 0x18, 0x7e, 0x82, 0xbe, 0x36, 0xa4, 0xe0, 0xa2, 0xa8, 0x9f, 0x13, 0xb6, 0xaa, 0x01,
	0x3d, 0x4d, 0xaa, 0xe4, 0x92, 0xac, 0x4e, 0x4e, 0xb9, 0xb7, 0xee, 0xb2, 0xf8, 0x01, 0x3a, 0x9b,
	0x5c, 0x74, 0xd8, 0x8e, 0x1a, 0x71, 0x77, 0x3c, 0x48, 0x6c, 0x12, 0xc9, 0xd4, 0xc9, 0xd9, 0x16,
	0xc0, 0x09, 0xf4, 0xab, 0x51, 0x7c, 0xaf, 0x53, 0xd2, 0x61, 0xa7, 0x72, 0xf4, 0x6b, 0x87, 0x53,
	0xb3, 0x5d, 0x06, 0xcf, 0xa0, 0x67, 0x07, 0xf3, 0xcf, 0x03, 0x0f, 0x79, 0x76, 0x90, 0xf7, 0x7f,
	0x3c, 0xf0, 0x5d, 0x83, 0x01, 0xf4, 0x2e, 0xb8, 0xa6, 0x9b, 0xa9, 0x06, 0x8f, 0x30, 0x84, 0x93,
	0x4b, 0x52, 0x1a, 0x76, 0xf3, 0x4d, 0x64, 0x52, 0x9a, 0x0b, 0x56, 0x32, 0x3b, 0x8d, 0xc0, 0xc3,
	0x11, 0x3c, 0xbb, 0x5f, 0xb9, 0x5a, 0x2f, 0xd9, 0x74, 0x4e, 0x44, 0xc1, 0x82, 0x23, 0x7c, 0x02,
	0x43, 0x17, 0x0d, 0x17, 0xc5, 0x67, 0xdb, 0x60, 0xd0, 0x40, 0x84, 0xc1, 0x56, 0x3c, 0x67, 0x86,
	0x04, 0xc7, 0xd8, 0x83, 0xf6, 0x35, 0x31, 0x74, 0xce, 0x45, 0x11, 0x34, 0xed, 0x2e, 0xa7, 0x44,
	0x08, 0xbb, 0xf3, 0xf1, 0x04, 0x82, 0x6b, 0xc2, 0x2d, 0xfc, 0x55, 0xaa, 0x8c, 0x69, 0x4a, 0x44,
	0xd0, 0xc2, 0x21, 0x74, 0x33, 0x46, 0xa5, 0xa0, 0xbc, 0xb4, 0x58, 0xdb, 0xf6, 0x9c, 0xd7, 0x53,
	0xae, 0x2f, 0xea, 0x58, 0xc4, 0x29, 0xd5, 0x2d, 0x80, 0x8f, 0xa1, 0x7f, 0xa5, 0x88, 0xd0, 0xdc,
	0xb6, 0x6e, 0x5d, 0x5d, 0x04, 0xf0, 0x73, 0x72, 0x67, 0xd7, 0xbd, 0xf3, 0xc9, 0x8f, 0xb3, 0x82,
	0x9b, 0xf9, 0x6a, 0x96, 0x50, 0xb9, 0x48, 0x17, 0x2b, 0x43, 0x0a, 0x26, 0x3e, 0x72, 0xb9, 0x59,
	0xa6, 0xcb, 0x9f, 0x45, 0xba, 0xf7, 0x35, 0xcf, 0xfc, 0xea, 0xa7, 0x99, 0xfc, 0x0d, 0x00, 0x00,
	0xff, 0xff, 0x18, 0x15, 0x33, 0x78, 0xf1, 0x03, 0x00, 0x00,
}