// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: test/test.proto

package test

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

type UserType int32

const (
	UserType_USER_TYPE_UNSPECIFIED UserType = 0
	UserType_CUSTOMER              UserType = 1
	UserType_STAFF                 UserType = 2
)

// Enum value maps for UserType.
var (
	UserType_name = map[int32]string{
		0: "USER_TYPE_UNSPECIFIED",
		1: "CUSTOMER",
		2: "STAFF",
	}
	UserType_value = map[string]int32{
		"USER_TYPE_UNSPECIFIED": 0,
		"CUSTOMER":              1,
		"STAFF":                 2,
	}
)

func (x UserType) Enum() *UserType {
	p := new(UserType)
	*p = x
	return p
}

func (x UserType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (UserType) Descriptor() protoreflect.EnumDescriptor {
	return file_test_test_proto_enumTypes[0].Descriptor()
}

func (UserType) Type() protoreflect.EnumType {
	return &file_test_test_proto_enumTypes[0]
}

func (x UserType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use UserType.Descriptor instead.
func (UserType) EnumDescriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{0}
}

type State int32

const (
	State_STATE_UNSPECIFIED State = 0
	State_ACTIVE            State = 1
)

// Enum value maps for State.
var (
	State_name = map[int32]string{
		0: "STATE_UNSPECIFIED",
		1: "ACTIVE",
	}
	State_value = map[string]int32{
		"STATE_UNSPECIFIED": 0,
		"ACTIVE":            1,
	}
)

func (x State) Enum() *State {
	p := new(State)
	*p = x
	return p
}

func (x State) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (State) Descriptor() protoreflect.EnumDescriptor {
	return file_test_test_proto_enumTypes[1].Descriptor()
}

func (State) Type() protoreflect.EnumType {
	return &file_test_test_proto_enumTypes[1]
}

func (x State) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use State.Descriptor instead.
func (State) EnumDescriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{1}
}

type EchoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MessageId   string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	MessageBody string `protobuf:"bytes,2,opt,name=message_body,json=messageBody,proto3" json:"message_body,omitempty"`
}

func (x *EchoRequest) Reset() {
	*x = EchoRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EchoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EchoRequest) ProtoMessage() {}

func (x *EchoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EchoRequest.ProtoReflect.Descriptor instead.
func (*EchoRequest) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{0}
}

func (x *EchoRequest) GetMessageId() string {
	if x != nil {
		return x.MessageId
	}
	return ""
}

func (x *EchoRequest) GetMessageBody() string {
	if x != nil {
		return x.MessageBody
	}
	return ""
}

type EchoResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MessageId   string   `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	MessageBody string   `protobuf:"bytes,2,opt,name=message_body,json=messageBody,proto3" json:"message_body,omitempty"`
	ReceivedAt  int64    `protobuf:"varint,3,opt,name=received_at,json=receivedAt,proto3" json:"received_at,omitempty"`
	UserType    UserType `protobuf:"varint,4,opt,name=user_type,json=userType,proto3,enum=scenarigo.testdata.test.UserType" json:"user_type,omitempty"`
	// Types that are assignable to UserId:
	//
	//	*EchoResponse_CustomerId
	//	*EchoResponse_StaffId
	UserId         isEchoResponse_UserId `protobuf_oneof:"user_id"`
	State          State                 `protobuf:"varint,7,opt,name=state,proto3,enum=scenarigo.testdata.test.State" json:"state,omitempty"`
	NullableString *StringValue          `protobuf:"bytes,11,opt,name=nullable_string,json=nullableString,proto3" json:"nullable_string,omitempty"`
}

func (x *EchoResponse) Reset() {
	*x = EchoResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EchoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EchoResponse) ProtoMessage() {}

func (x *EchoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EchoResponse.ProtoReflect.Descriptor instead.
func (*EchoResponse) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{1}
}

func (x *EchoResponse) GetMessageId() string {
	if x != nil {
		return x.MessageId
	}
	return ""
}

func (x *EchoResponse) GetMessageBody() string {
	if x != nil {
		return x.MessageBody
	}
	return ""
}

func (x *EchoResponse) GetReceivedAt() int64 {
	if x != nil {
		return x.ReceivedAt
	}
	return 0
}

func (x *EchoResponse) GetUserType() UserType {
	if x != nil {
		return x.UserType
	}
	return UserType_USER_TYPE_UNSPECIFIED
}

func (m *EchoResponse) GetUserId() isEchoResponse_UserId {
	if m != nil {
		return m.UserId
	}
	return nil
}

func (x *EchoResponse) GetCustomerId() string {
	if x, ok := x.GetUserId().(*EchoResponse_CustomerId); ok {
		return x.CustomerId
	}
	return ""
}

func (x *EchoResponse) GetStaffId() string {
	if x, ok := x.GetUserId().(*EchoResponse_StaffId); ok {
		return x.StaffId
	}
	return ""
}

func (x *EchoResponse) GetState() State {
	if x != nil {
		return x.State
	}
	return State_STATE_UNSPECIFIED
}

func (x *EchoResponse) GetNullableString() *StringValue {
	if x != nil {
		return x.NullableString
	}
	return nil
}

type isEchoResponse_UserId interface {
	isEchoResponse_UserId()
}

type EchoResponse_CustomerId struct {
	CustomerId string `protobuf:"bytes,5,opt,name=customer_id,json=customerId,proto3,oneof"`
}

type EchoResponse_StaffId struct {
	StaffId string `protobuf:"bytes,6,opt,name=staff_id,json=staffId,proto3,oneof"`
}

func (*EchoResponse_CustomerId) isEchoResponse_UserId() {}

func (*EchoResponse_StaffId) isEchoResponse_UserId() {}

// ref. google/protobuf/wrappers.proto
type StringValue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *StringValue) Reset() {
	*x = StringValue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StringValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StringValue) ProtoMessage() {}

func (x *StringValue) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StringValue.ProtoReflect.Descriptor instead.
func (*StringValue) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{2}
}

func (x *StringValue) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

var File_test_test_proto protoreflect.FileDescriptor

var file_test_test_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x17, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x67, 0x6f, 0x2e, 0x74, 0x65, 0x73,
	0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x22, 0x4f, 0x0a, 0x0b, 0x45, 0x63,
	0x68, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x5f, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x42, 0x6f, 0x64, 0x79, 0x22, 0x81, 0x03, 0x0a, 0x0c,
	0x45, 0x63, 0x68, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1d, 0x0a, 0x0a,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x42, 0x6f, 0x64, 0x79, 0x12, 0x1f,
	0x0a, 0x0b, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x0a, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x64, 0x41, 0x74, 0x12,
	0x3e, 0x0a, 0x09, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x21, 0x2e, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x67, 0x6f, 0x2e, 0x74,
	0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x55, 0x73, 0x65,
	0x72, 0x54, 0x79, 0x70, 0x65, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x21, 0x0a, 0x0b, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0a, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65, 0x72,
	0x49, 0x64, 0x12, 0x1b, 0x0a, 0x08, 0x73, 0x74, 0x61, 0x66, 0x66, 0x5f, 0x69, 0x64, 0x18, 0x06,
	0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x07, 0x73, 0x74, 0x61, 0x66, 0x66, 0x49, 0x64, 0x12,
	0x34, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1e,
	0x2e, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x67, 0x6f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x64,
	0x61, 0x74, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x4d, 0x0a, 0x0f, 0x6e, 0x75, 0x6c, 0x6c, 0x61, 0x62, 0x6c,
	0x65, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24,
	0x2e, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x67, 0x6f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x64,
	0x61, 0x74, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x0e, 0x6e, 0x75, 0x6c, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x42, 0x09, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x22,
	0x23, 0x0a, 0x0b, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x2a, 0x3e, 0x0a, 0x08, 0x55, 0x73, 0x65, 0x72, 0x54, 0x79, 0x70, 0x65,
	0x12, 0x19, 0x0a, 0x15, 0x55, 0x53, 0x45, 0x52, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x4e,
	0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x43,
	0x55, 0x53, 0x54, 0x4f, 0x4d, 0x45, 0x52, 0x10, 0x01, 0x12, 0x09, 0x0a, 0x05, 0x53, 0x54, 0x41,
	0x46, 0x46, 0x10, 0x02, 0x2a, 0x2a, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x15, 0x0a,
	0x11, 0x53, 0x54, 0x41, 0x54, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49,
	0x45, 0x44, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x41, 0x43, 0x54, 0x49, 0x56, 0x45, 0x10, 0x01,
	0x32, 0x5d, 0x0a, 0x04, 0x54, 0x65, 0x73, 0x74, 0x12, 0x55, 0x0a, 0x04, 0x45, 0x63, 0x68, 0x6f,
	0x12, 0x24, 0x2e, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x67, 0x6f, 0x2e, 0x74, 0x65, 0x73,
	0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x45, 0x63, 0x68, 0x6f, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x25, 0x2e, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69,
	0x67, 0x6f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74,
	0x2e, 0x45, 0x63, 0x68, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42,
	0x38, 0x5a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x7a, 0x6f,
	0x6e, 0x63, 0x6f, 0x65, 0x6e, 0x2f, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x67, 0x6f, 0x2f,
	0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x62, 0x2f,
	0x74, 0x65, 0x73, 0x74, 0x3b, 0x74, 0x65, 0x73, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_test_test_proto_rawDescOnce sync.Once
	file_test_test_proto_rawDescData = file_test_test_proto_rawDesc
)

func file_test_test_proto_rawDescGZIP() []byte {
	file_test_test_proto_rawDescOnce.Do(func() {
		file_test_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_test_test_proto_rawDescData)
	})
	return file_test_test_proto_rawDescData
}

var file_test_test_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_test_test_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_test_test_proto_goTypes = []interface{}{
	(UserType)(0),        // 0: scenarigo.testdata.test.UserType
	(State)(0),           // 1: scenarigo.testdata.test.State
	(*EchoRequest)(nil),  // 2: scenarigo.testdata.test.EchoRequest
	(*EchoResponse)(nil), // 3: scenarigo.testdata.test.EchoResponse
	(*StringValue)(nil),  // 4: scenarigo.testdata.test.StringValue
}
var file_test_test_proto_depIdxs = []int32{
	0, // 0: scenarigo.testdata.test.EchoResponse.user_type:type_name -> scenarigo.testdata.test.UserType
	1, // 1: scenarigo.testdata.test.EchoResponse.state:type_name -> scenarigo.testdata.test.State
	4, // 2: scenarigo.testdata.test.EchoResponse.nullable_string:type_name -> scenarigo.testdata.test.StringValue
	2, // 3: scenarigo.testdata.test.Test.Echo:input_type -> scenarigo.testdata.test.EchoRequest
	3, // 4: scenarigo.testdata.test.Test.Echo:output_type -> scenarigo.testdata.test.EchoResponse
	4, // [4:5] is the sub-list for method output_type
	3, // [3:4] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_test_test_proto_init() }
func file_test_test_proto_init() {
	if File_test_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_test_test_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EchoRequest); i {
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
		file_test_test_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EchoResponse); i {
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
		file_test_test_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StringValue); i {
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
	file_test_test_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*EchoResponse_CustomerId)(nil),
		(*EchoResponse_StaffId)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_test_test_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_test_test_proto_goTypes,
		DependencyIndexes: file_test_test_proto_depIdxs,
		EnumInfos:         file_test_test_proto_enumTypes,
		MessageInfos:      file_test_test_proto_msgTypes,
	}.Build()
	File_test_test_proto = out.File
	file_test_test_proto_rawDesc = nil
	file_test_test_proto_goTypes = nil
	file_test_test_proto_depIdxs = nil
}
