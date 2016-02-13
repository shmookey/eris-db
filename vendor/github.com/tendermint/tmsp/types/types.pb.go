// Code generated by protoc-gen-go.
// source: types/types.proto
// DO NOT EDIT!

/*
Package types is a generated protocol buffer package.

It is generated from these files:
	types/types.proto

It has these top-level messages:
	Request
	Response
*/
package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type MessageType int32

const (
	MessageType_NullMessage MessageType = 0
	MessageType_Echo        MessageType = 1
	MessageType_Flush       MessageType = 2
	MessageType_Info        MessageType = 3
	MessageType_SetOption   MessageType = 4
	MessageType_Exception   MessageType = 5
	MessageType_AppendTx    MessageType = 17
	MessageType_CheckTx     MessageType = 18
	MessageType_GetHash     MessageType = 19
	MessageType_Query       MessageType = 20
)

var MessageType_name = map[int32]string{
	0:  "NullMessage",
	1:  "Echo",
	2:  "Flush",
	3:  "Info",
	4:  "SetOption",
	5:  "Exception",
	17: "AppendTx",
	18: "CheckTx",
	19: "GetHash",
	20: "Query",
}
var MessageType_value = map[string]int32{
	"NullMessage": 0,
	"Echo":        1,
	"Flush":       2,
	"Info":        3,
	"SetOption":   4,
	"Exception":   5,
	"AppendTx":    17,
	"CheckTx":     18,
	"GetHash":     19,
	"Query":       20,
}

func (x MessageType) String() string {
	return proto.EnumName(MessageType_name, int32(x))
}
func (MessageType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type CodeType int32

const (
	CodeType_OK                CodeType = 0
	CodeType_InternalError     CodeType = 1
	CodeType_Unauthorized      CodeType = 2
	CodeType_InsufficientFees  CodeType = 3
	CodeType_UnknownRequest    CodeType = 4
	CodeType_EncodingError     CodeType = 5
	CodeType_BadNonce          CodeType = 6
	CodeType_UnknownAccount    CodeType = 7
	CodeType_InsufficientFunds CodeType = 8
)

var CodeType_name = map[int32]string{
	0: "OK",
	1: "InternalError",
	2: "Unauthorized",
	3: "InsufficientFees",
	4: "UnknownRequest",
	5: "EncodingError",
	6: "BadNonce",
	7: "UnknownAccount",
	8: "InsufficientFunds",
}
var CodeType_value = map[string]int32{
	"OK":                0,
	"InternalError":     1,
	"Unauthorized":      2,
	"InsufficientFees":  3,
	"UnknownRequest":    4,
	"EncodingError":     5,
	"BadNonce":          6,
	"UnknownAccount":    7,
	"InsufficientFunds": 8,
}

func (x CodeType) String() string {
	return proto.EnumName(CodeType_name, int32(x))
}
func (CodeType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type Request struct {
	Type  MessageType `protobuf:"varint,1,opt,name=type,enum=types.MessageType" json:"type,omitempty"`
	Data  []byte      `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Key   string      `protobuf:"bytes,3,opt,name=key" json:"key,omitempty"`
	Value string      `protobuf:"bytes,4,opt,name=value" json:"value,omitempty"`
}

func (m *Request) Reset()                    { *m = Request{} }
func (m *Request) String() string            { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()               {}
func (*Request) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type Response struct {
	Type  MessageType `protobuf:"varint,1,opt,name=type,enum=types.MessageType" json:"type,omitempty"`
	Data  []byte      `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Code  CodeType    `protobuf:"varint,3,opt,name=code,enum=types.CodeType" json:"code,omitempty"`
	Error string      `protobuf:"bytes,4,opt,name=error" json:"error,omitempty"`
	Log   string      `protobuf:"bytes,5,opt,name=log" json:"log,omitempty"`
}

func (m *Response) Reset()                    { *m = Response{} }
func (m *Response) String() string            { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()               {}
func (*Response) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func init() {
	proto.RegisterType((*Request)(nil), "types.Request")
	proto.RegisterType((*Response)(nil), "types.Response")
	proto.RegisterEnum("types.MessageType", MessageType_name, MessageType_value)
	proto.RegisterEnum("types.CodeType", CodeType_name, CodeType_value)
}

var fileDescriptor0 = []byte{
	// 405 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xa4, 0x92, 0xdf, 0x6e, 0xd3, 0x30,
	0x14, 0xc6, 0x49, 0x9b, 0xf4, 0xcf, 0x69, 0xd7, 0xb9, 0x87, 0x22, 0xe5, 0x12, 0x0d, 0x09, 0xa1,
	0x5d, 0x0c, 0x69, 0x3c, 0xc1, 0x98, 0x3a, 0xa8, 0x10, 0x9b, 0x08, 0xdb, 0x03, 0x18, 0xe7, 0xb4,
	0x8d, 0x1a, 0x8e, 0x43, 0x6c, 0xc3, 0xca, 0x43, 0x70, 0xc3, 0x73, 0xf0, 0x8e, 0xd8, 0x4e, 0x27,
	0x8d, 0x6b, 0x6e, 0xa2, 0xf3, 0x7d, 0xf6, 0xf9, 0xce, 0xcf, 0x8e, 0x61, 0x6e, 0xf7, 0x0d, 0x99,
	0xd7, 0xf1, 0x7b, 0xd6, 0xb4, 0xda, 0x6a, 0xcc, 0xa2, 0x38, 0xf9, 0x0a, 0xc3, 0x82, 0xbe, 0x39,
	0x32, 0x16, 0x5f, 0x42, 0x1a, 0xbc, 0x3c, 0x79, 0x9e, 0xbc, 0x9a, 0x9d, 0xe3, 0x59, 0xb7, 0xfb,
	0x23, 0x19, 0x23, 0x37, 0x74, 0xeb, 0x45, 0x11, 0xd7, 0x11, 0x21, 0x2d, 0xa5, 0x95, 0x79, 0xcf,
	0xef, 0x9b, 0x16, 0xb1, 0x46, 0x01, 0xfd, 0x1d, 0xed, 0xf3, 0xbe, 0xb7, 0xc6, 0x45, 0x28, 0x71,
	0x01, 0xd9, 0x77, 0x59, 0x3b, 0xca, 0xd3, 0xe8, 0x75, 0xe2, 0xe4, 0x77, 0x02, 0xa3, 0x82, 0x4c,
	0xa3, 0xd9, 0xd0, 0x7f, 0x0d, 0x7c, 0x01, 0xa9, 0xd2, 0x25, 0xc5, 0x89, 0xb3, 0xf3, 0xe3, 0x43,
	0xef, 0xa5, 0xb7, 0xba, 0xc6, 0xb0, 0x18, 0x18, 0xa8, 0x6d, 0x75, 0xfb, 0xc0, 0x10, 0x45, 0x60,
	0xad, 0xf5, 0x26, 0xcf, 0x3a, 0x56, 0x5f, 0x9e, 0xfe, 0x4a, 0x60, 0xf2, 0x68, 0x2c, 0x1e, 0xc3,
	0xe4, 0xda, 0xd5, 0xf5, 0xc1, 0x12, 0x4f, 0x70, 0x04, 0xe9, 0x52, 0x6d, 0xb5, 0x48, 0x70, 0x0c,
	0xd9, 0x55, 0xed, 0xcc, 0x56, 0xf4, 0x82, 0xb9, 0xe2, 0xb5, 0x16, 0x7d, 0x3c, 0x82, 0xf1, 0x67,
	0xb2, 0x37, 0x8d, 0xad, 0x34, 0x8b, 0x34, 0xc8, 0xe5, 0xbd, 0xa2, 0x4e, 0x66, 0x38, 0x85, 0xd1,
	0x45, 0xd3, 0x10, 0x97, 0xb7, 0xf7, 0x62, 0x8e, 0x13, 0x18, 0x5e, 0x6e, 0x49, 0xed, 0xbc, 0xc0,
	0x20, 0xde, 0x91, 0x7d, 0x2f, 0x7d, 0xde, 0xd3, 0x10, 0xfd, 0xc9, 0x51, 0xbb, 0x17, 0x8b, 0xd3,
	0x3f, 0xfe, 0x9a, 0x1e, 0xce, 0x82, 0x03, 0xe8, 0xdd, 0x7c, 0xf0, 0x10, 0x73, 0x38, 0x5a, 0xb1,
	0xa5, 0x96, 0x65, 0xbd, 0x0c, 0x07, 0xf1, 0x34, 0x02, 0xa6, 0x77, 0x2c, 0x9d, 0xdd, 0xea, 0xb6,
	0xfa, 0x49, 0xa5, 0x87, 0x5a, 0x80, 0x58, 0xb1, 0x71, 0xeb, 0x75, 0xa5, 0x2a, 0x62, 0x7b, 0x45,
	0x64, 0x3c, 0x20, 0xc2, 0xec, 0x8e, 0x77, 0xac, 0x7f, 0xf0, 0xe1, 0x67, 0x7b, 0x4a, 0x1f, 0xb7,
	0x64, 0x7f, 0x4d, 0x15, 0x6f, 0xba, 0xb8, 0x48, 0xfa, 0x56, 0x96, 0xd7, 0x9a, 0x15, 0x89, 0xc1,
	0xa3, 0xa6, 0x0b, 0xa5, 0xb4, 0x63, 0x2b, 0x86, 0xf8, 0x0c, 0xe6, 0xff, 0xc4, 0x3b, 0x2e, 0x8d,
	0x18, 0x7d, 0x19, 0xc4, 0x37, 0xf5, 0xe6, 0x6f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xcf, 0x90, 0x25,
	0x05, 0x68, 0x02, 0x00, 0x00,
}