// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/havoc-io/mutagen/pkg/sync/cache.proto

package sync

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import timestamp "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// CacheEntry represents cache data for a file on disk.
type CacheEntry struct {
	// Mode stores the value of the Go os package's FileMode type. The meaning
	// of this value is defined to be stable (even if we'd have to implement its
	// computation ourselves when porting to another language), so it's safe to
	// use, and it's a relatively sane implementation based on POSIX mode bits.
	// This information is currently used in scans and transitions, but only the
	// type and executability bits are really used (or at least necessary) at
	// the moment. It's not clear whether or not we'll eventually need the other
	// permission bits, and it might be possible to get away with a type
	// enumeration instead. This might be easier than trying to replicate
	// FileMode values if moving to another language, though I'm not sure that
	// would be too difficult. But I suppose it's better to just have this
	// additional mode information available for the sake of generality and
	// extensibility. We can always drop it later, but we can't add it back. It
	// may (I'm not exactly sure how) come in useful if we want to implement
	// permission propagation or need a better change detection heuristic. At
	// the moment though, it's highly unlikely that we'll switch away from Go,
	// and I'm willing to live with this slightly "unclean" design, especially
	// given its potential and the relative ease of deprecating it if necessary.
	Mode uint32 `protobuf:"varint,1,opt,name=mode" json:"mode,omitempty"`
	// ModificationTime is the cached file modification time.
	ModificationTime *timestamp.Timestamp `protobuf:"bytes,2,opt,name=modificationTime" json:"modificationTime,omitempty"`
	// Size is the cached file size.
	Size uint64 `protobuf:"varint,3,opt,name=size" json:"size,omitempty"`
	// Digest is the cached digest.
	Digest               []byte   `protobuf:"bytes,4,opt,name=digest,proto3" json:"digest,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CacheEntry) Reset()         { *m = CacheEntry{} }
func (m *CacheEntry) String() string { return proto.CompactTextString(m) }
func (*CacheEntry) ProtoMessage()    {}
func (*CacheEntry) Descriptor() ([]byte, []int) {
	return fileDescriptor_cache_25c79c9babef23de, []int{0}
}
func (m *CacheEntry) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CacheEntry.Unmarshal(m, b)
}
func (m *CacheEntry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CacheEntry.Marshal(b, m, deterministic)
}
func (dst *CacheEntry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CacheEntry.Merge(dst, src)
}
func (m *CacheEntry) XXX_Size() int {
	return xxx_messageInfo_CacheEntry.Size(m)
}
func (m *CacheEntry) XXX_DiscardUnknown() {
	xxx_messageInfo_CacheEntry.DiscardUnknown(m)
}

var xxx_messageInfo_CacheEntry proto.InternalMessageInfo

func (m *CacheEntry) GetMode() uint32 {
	if m != nil {
		return m.Mode
	}
	return 0
}

func (m *CacheEntry) GetModificationTime() *timestamp.Timestamp {
	if m != nil {
		return m.ModificationTime
	}
	return nil
}

func (m *CacheEntry) GetSize() uint64 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *CacheEntry) GetDigest() []byte {
	if m != nil {
		return m.Digest
	}
	return nil
}

// Cache provides a store for file metadata and digets to allow for efficient
// rescans.
type Cache struct {
	// Entries is a map from scan path to cache entry.
	Entries              map[string]*CacheEntry `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}               `json:"-"`
	XXX_unrecognized     []byte                 `json:"-"`
	XXX_sizecache        int32                  `json:"-"`
}

func (m *Cache) Reset()         { *m = Cache{} }
func (m *Cache) String() string { return proto.CompactTextString(m) }
func (*Cache) ProtoMessage()    {}
func (*Cache) Descriptor() ([]byte, []int) {
	return fileDescriptor_cache_25c79c9babef23de, []int{1}
}
func (m *Cache) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Cache.Unmarshal(m, b)
}
func (m *Cache) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Cache.Marshal(b, m, deterministic)
}
func (dst *Cache) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Cache.Merge(dst, src)
}
func (m *Cache) XXX_Size() int {
	return xxx_messageInfo_Cache.Size(m)
}
func (m *Cache) XXX_DiscardUnknown() {
	xxx_messageInfo_Cache.DiscardUnknown(m)
}

var xxx_messageInfo_Cache proto.InternalMessageInfo

func (m *Cache) GetEntries() map[string]*CacheEntry {
	if m != nil {
		return m.Entries
	}
	return nil
}

func init() {
	proto.RegisterType((*CacheEntry)(nil), "sync.CacheEntry")
	proto.RegisterType((*Cache)(nil), "sync.Cache")
	proto.RegisterMapType((map[string]*CacheEntry)(nil), "sync.Cache.EntriesEntry")
}

func init() {
	proto.RegisterFile("github.com/havoc-io/mutagen/pkg/sync/cache.proto", fileDescriptor_cache_25c79c9babef23de)
}

var fileDescriptor_cache_25c79c9babef23de = []byte{
	// 279 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x8f, 0x31, 0x6b, 0xc3, 0x30,
	0x10, 0x85, 0x51, 0xe2, 0xa4, 0x54, 0x49, 0xc1, 0x68, 0x28, 0xc2, 0x4b, 0x4d, 0x86, 0xe2, 0xa5,
	0x52, 0x71, 0x97, 0xd2, 0xb5, 0xa4, 0x53, 0x27, 0x91, 0x3f, 0x20, 0xcb, 0x17, 0x59, 0x24, 0xb2,
	0x8c, 0x2d, 0x07, 0xdc, 0x7f, 0xd0, 0xbd, 0x3f, 0xb8, 0x58, 0xae, 0x21, 0xd0, 0xed, 0xdd, 0xbd,
	0xbb, 0x8f, 0xf7, 0xf0, 0xb3, 0x36, 0xbe, 0xea, 0x0b, 0xa6, 0x9c, 0xe5, 0x95, 0xbc, 0x38, 0xf5,
	0x64, 0x1c, 0xb7, 0xbd, 0x97, 0x1a, 0x6a, 0xde, 0x9c, 0x34, 0xef, 0x86, 0x5a, 0x71, 0x25, 0x55,
	0x05, 0xac, 0x69, 0x9d, 0x77, 0x24, 0x1a, 0x37, 0xc9, 0x83, 0x76, 0x4e, 0x9f, 0x81, 0x87, 0x5d,
	0xd1, 0x1f, 0xb9, 0x37, 0x16, 0x3a, 0x2f, 0x6d, 0x33, 0x9d, 0xed, 0x7e, 0x10, 0xc6, 0xef, 0xe3,
	0xdb, 0xbe, 0xf6, 0xed, 0x40, 0x08, 0x8e, 0xac, 0x2b, 0x81, 0xa2, 0x14, 0x65, 0x77, 0x22, 0x68,
	0xf2, 0x81, 0x63, 0xeb, 0x4a, 0x73, 0x34, 0x4a, 0x7a, 0xe3, 0xea, 0x83, 0xb1, 0x40, 0x17, 0x29,
	0xca, 0x36, 0x79, 0xc2, 0x26, 0x3c, 0x9b, 0xf1, 0xec, 0x30, 0xe3, 0xc5, 0xbf, 0x9f, 0x91, 0xdd,
	0x99, 0x2f, 0xa0, 0xcb, 0x14, 0x65, 0x91, 0x08, 0x9a, 0xdc, 0xe3, 0x75, 0x69, 0x34, 0x74, 0x9e,
	0x46, 0x29, 0xca, 0xb6, 0xe2, 0x6f, 0xda, 0x7d, 0x23, 0xbc, 0x0a, 0xb1, 0x48, 0x8e, 0x6f, 0xa0,
	0xf6, 0xad, 0x81, 0x8e, 0xa2, 0x74, 0x99, 0x6d, 0x72, 0xca, 0xc6, 0x66, 0x2c, 0xb8, 0x6c, 0x3f,
	0x59, 0x21, 0xbc, 0x98, 0x0f, 0x93, 0x4f, 0xbc, 0xbd, 0x36, 0x48, 0x8c, 0x97, 0x27, 0x18, 0x42,
	0xa9, 0x5b, 0x31, 0x4a, 0xf2, 0x88, 0x57, 0x17, 0x79, 0xee, 0xe7, 0x22, 0xf1, 0x15, 0x73, 0x62,
	0x4d, 0xf6, 0xdb, 0xe2, 0x15, 0x15, 0xeb, 0xd0, 0xee, 0xe5, 0x37, 0x00, 0x00, 0xff, 0xff, 0xb0,
	0xc8, 0x7a, 0x35, 0x84, 0x01, 0x00, 0x00,
}