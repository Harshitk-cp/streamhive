// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: libs/proto/signaling/signaling.proto

package signaling

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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

// SignalingMessage represents a WebRTC signaling message
type SignalingMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Type          string                 `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	StreamId      string                 `protobuf:"bytes,2,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	SenderId      string                 `protobuf:"bytes,3,opt,name=sender_id,json=senderId,proto3" json:"sender_id,omitempty"`
	RecipientId   string                 `protobuf:"bytes,4,opt,name=recipient_id,json=recipientId,proto3" json:"recipient_id,omitempty"`
	Payload       string                 `protobuf:"bytes,5,opt,name=payload,proto3" json:"payload,omitempty"`
	Timestamp     int64                  `protobuf:"varint,6,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SignalingMessage) Reset() {
	*x = SignalingMessage{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SignalingMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SignalingMessage) ProtoMessage() {}

func (x *SignalingMessage) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SignalingMessage.ProtoReflect.Descriptor instead.
func (*SignalingMessage) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{0}
}

func (x *SignalingMessage) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *SignalingMessage) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *SignalingMessage) GetSenderId() string {
	if x != nil {
		return x.SenderId
	}
	return ""
}

func (x *SignalingMessage) GetRecipientId() string {
	if x != nil {
		return x.RecipientId
	}
	return ""
}

func (x *SignalingMessage) GetPayload() string {
	if x != nil {
		return x.Payload
	}
	return ""
}

func (x *SignalingMessage) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

// GetStreamInfoRequest is a request to get information about a stream
type GetStreamInfoRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetStreamInfoRequest) Reset() {
	*x = GetStreamInfoRequest{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetStreamInfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetStreamInfoRequest) ProtoMessage() {}

func (x *GetStreamInfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetStreamInfoRequest.ProtoReflect.Descriptor instead.
func (*GetStreamInfoRequest) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{1}
}

func (x *GetStreamInfoRequest) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

// StreamInfo contains information about a stream
type StreamInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	ClientCount   int32                  `protobuf:"varint,2,opt,name=client_count,json=clientCount,proto3" json:"client_count,omitempty"`
	Active        bool                   `protobuf:"varint,3,opt,name=active,proto3" json:"active,omitempty"`
	CreatedAt     *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt     *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StreamInfo) Reset() {
	*x = StreamInfo{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StreamInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StreamInfo) ProtoMessage() {}

func (x *StreamInfo) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StreamInfo.ProtoReflect.Descriptor instead.
func (*StreamInfo) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{2}
}

func (x *StreamInfo) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *StreamInfo) GetClientCount() int32 {
	if x != nil {
		return x.ClientCount
	}
	return 0
}

func (x *StreamInfo) GetActive() bool {
	if x != nil {
		return x.Active
	}
	return false
}

func (x *StreamInfo) GetCreatedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

func (x *StreamInfo) GetUpdatedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.UpdatedAt
	}
	return nil
}

// NotifyStreamEndedRequest is a request to notify clients that a stream has ended
type NotifyStreamEndedRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	Reason        string                 `protobuf:"bytes,2,opt,name=reason,proto3" json:"reason,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NotifyStreamEndedRequest) Reset() {
	*x = NotifyStreamEndedRequest{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NotifyStreamEndedRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotifyStreamEndedRequest) ProtoMessage() {}

func (x *NotifyStreamEndedRequest) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotifyStreamEndedRequest.ProtoReflect.Descriptor instead.
func (*NotifyStreamEndedRequest) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{3}
}

func (x *NotifyStreamEndedRequest) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *NotifyStreamEndedRequest) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

// SignalingStats contains statistics about the signaling service
type SignalingStats struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	ConnectedClients int32                  `protobuf:"varint,1,opt,name=connected_clients,json=connectedClients,proto3" json:"connected_clients,omitempty"`
	Timestamp        *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *SignalingStats) Reset() {
	*x = SignalingStats{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SignalingStats) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SignalingStats) ProtoMessage() {}

func (x *SignalingStats) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SignalingStats.ProtoReflect.Descriptor instead.
func (*SignalingStats) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{4}
}

func (x *SignalingStats) GetConnectedClients() int32 {
	if x != nil {
		return x.ConnectedClients
	}
	return 0
}

func (x *SignalingStats) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

// SessionDescription represents an SDP session description
type SessionDescription struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Type          string                 `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Sdp           string                 `protobuf:"bytes,2,opt,name=sdp,proto3" json:"sdp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SessionDescription) Reset() {
	*x = SessionDescription{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SessionDescription) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SessionDescription) ProtoMessage() {}

func (x *SessionDescription) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SessionDescription.ProtoReflect.Descriptor instead.
func (*SessionDescription) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{5}
}

func (x *SessionDescription) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *SessionDescription) GetSdp() string {
	if x != nil {
		return x.Sdp
	}
	return ""
}

// ICECandidate represents an ICE candidate
type ICECandidate struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Candidate     string                 `protobuf:"bytes,1,opt,name=candidate,proto3" json:"candidate,omitempty"`
	SdpMid        string                 `protobuf:"bytes,2,opt,name=sdp_mid,json=sdpMid,proto3" json:"sdp_mid,omitempty"`
	SdpMlineIndex int32                  `protobuf:"varint,3,opt,name=sdp_mline_index,json=sdpMlineIndex,proto3" json:"sdp_mline_index,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ICECandidate) Reset() {
	*x = ICECandidate{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ICECandidate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ICECandidate) ProtoMessage() {}

func (x *ICECandidate) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ICECandidate.ProtoReflect.Descriptor instead.
func (*ICECandidate) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{6}
}

func (x *ICECandidate) GetCandidate() string {
	if x != nil {
		return x.Candidate
	}
	return ""
}

func (x *ICECandidate) GetSdpMid() string {
	if x != nil {
		return x.SdpMid
	}
	return ""
}

func (x *ICECandidate) GetSdpMlineIndex() int32 {
	if x != nil {
		return x.SdpMlineIndex
	}
	return 0
}

// CreateOfferRequest is the request for creating an offer
type CreateOfferRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	ClientId      string                 `protobuf:"bytes,2,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateOfferRequest) Reset() {
	*x = CreateOfferRequest{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateOfferRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateOfferRequest) ProtoMessage() {}

func (x *CreateOfferRequest) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateOfferRequest.ProtoReflect.Descriptor instead.
func (*CreateOfferRequest) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{7}
}

func (x *CreateOfferRequest) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *CreateOfferRequest) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

// CreateOfferResponse is the response for creating an offer
type CreateOfferResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Offer         *SessionDescription    `protobuf:"bytes,1,opt,name=offer,proto3" json:"offer,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateOfferResponse) Reset() {
	*x = CreateOfferResponse{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateOfferResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateOfferResponse) ProtoMessage() {}

func (x *CreateOfferResponse) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateOfferResponse.ProtoReflect.Descriptor instead.
func (*CreateOfferResponse) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{8}
}

func (x *CreateOfferResponse) GetOffer() *SessionDescription {
	if x != nil {
		return x.Offer
	}
	return nil
}

// CreateAnswerRequest is the request for creating an answer
type CreateAnswerRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	ClientId      string                 `protobuf:"bytes,2,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	Offer         *SessionDescription    `protobuf:"bytes,3,opt,name=offer,proto3" json:"offer,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateAnswerRequest) Reset() {
	*x = CreateAnswerRequest{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateAnswerRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateAnswerRequest) ProtoMessage() {}

func (x *CreateAnswerRequest) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateAnswerRequest.ProtoReflect.Descriptor instead.
func (*CreateAnswerRequest) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{9}
}

func (x *CreateAnswerRequest) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *CreateAnswerRequest) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *CreateAnswerRequest) GetOffer() *SessionDescription {
	if x != nil {
		return x.Offer
	}
	return nil
}

// CreateAnswerResponse is the response for creating an answer
type CreateAnswerResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Answer        *SessionDescription    `protobuf:"bytes,1,opt,name=answer,proto3" json:"answer,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateAnswerResponse) Reset() {
	*x = CreateAnswerResponse{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateAnswerResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateAnswerResponse) ProtoMessage() {}

func (x *CreateAnswerResponse) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateAnswerResponse.ProtoReflect.Descriptor instead.
func (*CreateAnswerResponse) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{10}
}

func (x *CreateAnswerResponse) GetAnswer() *SessionDescription {
	if x != nil {
		return x.Answer
	}
	return nil
}

// AddICECandidateRequest is the request for adding an ICE candidate
type AddICECandidateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	ClientId      string                 `protobuf:"bytes,2,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	Candidate     *ICECandidate          `protobuf:"bytes,3,opt,name=candidate,proto3" json:"candidate,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddICECandidateRequest) Reset() {
	*x = AddICECandidateRequest{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddICECandidateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddICECandidateRequest) ProtoMessage() {}

func (x *AddICECandidateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddICECandidateRequest.ProtoReflect.Descriptor instead.
func (*AddICECandidateRequest) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{11}
}

func (x *AddICECandidateRequest) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *AddICECandidateRequest) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *AddICECandidateRequest) GetCandidate() *ICECandidate {
	if x != nil {
		return x.Candidate
	}
	return nil
}

// AddICECandidateResponse is the response for adding an ICE candidate
type AddICECandidateResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddICECandidateResponse) Reset() {
	*x = AddICECandidateResponse{}
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddICECandidateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddICECandidateResponse) ProtoMessage() {}

func (x *AddICECandidateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_signaling_signaling_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddICECandidateResponse.ProtoReflect.Descriptor instead.
func (*AddICECandidateResponse) Descriptor() ([]byte, []int) {
	return file_libs_proto_signaling_signaling_proto_rawDescGZIP(), []int{12}
}

func (x *AddICECandidateResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

var File_libs_proto_signaling_signaling_proto protoreflect.FileDescriptor

const file_libs_proto_signaling_signaling_proto_rawDesc = "" +
	"\n" +
	"$libs/proto/signaling/signaling.proto\x12\tsignaling\x1a\x1fgoogle/protobuf/timestamp.proto\x1a\x1bgoogle/protobuf/empty.proto\"\xbb\x01\n" +
	"\x10SignalingMessage\x12\x12\n" +
	"\x04type\x18\x01 \x01(\tR\x04type\x12\x1b\n" +
	"\tstream_id\x18\x02 \x01(\tR\bstreamId\x12\x1b\n" +
	"\tsender_id\x18\x03 \x01(\tR\bsenderId\x12!\n" +
	"\frecipient_id\x18\x04 \x01(\tR\vrecipientId\x12\x18\n" +
	"\apayload\x18\x05 \x01(\tR\apayload\x12\x1c\n" +
	"\ttimestamp\x18\x06 \x01(\x03R\ttimestamp\"3\n" +
	"\x14GetStreamInfoRequest\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\"\xda\x01\n" +
	"\n" +
	"StreamInfo\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12!\n" +
	"\fclient_count\x18\x02 \x01(\x05R\vclientCount\x12\x16\n" +
	"\x06active\x18\x03 \x01(\bR\x06active\x129\n" +
	"\n" +
	"created_at\x18\x04 \x01(\v2\x1a.google.protobuf.TimestampR\tcreatedAt\x129\n" +
	"\n" +
	"updated_at\x18\x05 \x01(\v2\x1a.google.protobuf.TimestampR\tupdatedAt\"O\n" +
	"\x18NotifyStreamEndedRequest\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12\x16\n" +
	"\x06reason\x18\x02 \x01(\tR\x06reason\"w\n" +
	"\x0eSignalingStats\x12+\n" +
	"\x11connected_clients\x18\x01 \x01(\x05R\x10connectedClients\x128\n" +
	"\ttimestamp\x18\x02 \x01(\v2\x1a.google.protobuf.TimestampR\ttimestamp\":\n" +
	"\x12SessionDescription\x12\x12\n" +
	"\x04type\x18\x01 \x01(\tR\x04type\x12\x10\n" +
	"\x03sdp\x18\x02 \x01(\tR\x03sdp\"m\n" +
	"\fICECandidate\x12\x1c\n" +
	"\tcandidate\x18\x01 \x01(\tR\tcandidate\x12\x17\n" +
	"\asdp_mid\x18\x02 \x01(\tR\x06sdpMid\x12&\n" +
	"\x0fsdp_mline_index\x18\x03 \x01(\x05R\rsdpMlineIndex\"N\n" +
	"\x12CreateOfferRequest\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12\x1b\n" +
	"\tclient_id\x18\x02 \x01(\tR\bclientId\"J\n" +
	"\x13CreateOfferResponse\x123\n" +
	"\x05offer\x18\x01 \x01(\v2\x1d.signaling.SessionDescriptionR\x05offer\"\x84\x01\n" +
	"\x13CreateAnswerRequest\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12\x1b\n" +
	"\tclient_id\x18\x02 \x01(\tR\bclientId\x123\n" +
	"\x05offer\x18\x03 \x01(\v2\x1d.signaling.SessionDescriptionR\x05offer\"M\n" +
	"\x14CreateAnswerResponse\x125\n" +
	"\x06answer\x18\x01 \x01(\v2\x1d.signaling.SessionDescriptionR\x06answer\"\x89\x01\n" +
	"\x16AddICECandidateRequest\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12\x1b\n" +
	"\tclient_id\x18\x02 \x01(\tR\bclientId\x125\n" +
	"\tcandidate\x18\x03 \x01(\v2\x17.signaling.ICECandidateR\tcandidate\"3\n" +
	"\x17AddICECandidateResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess2\x83\x05\n" +
	"\x10SignalingService\x12L\n" +
	"\vCreateOffer\x12\x1d.signaling.CreateOfferRequest\x1a\x1e.signaling.CreateOfferResponse\x12O\n" +
	"\fCreateAnswer\x12\x1e.signaling.CreateAnswerRequest\x1a\x1f.signaling.CreateAnswerResponse\x12X\n" +
	"\x0fAddICECandidate\x12!.signaling.AddICECandidateRequest\x1a\".signaling.AddICECandidateResponse\x12K\n" +
	"\x14SendSignalingMessage\x12\x1b.signaling.SignalingMessage\x1a\x16.google.protobuf.Empty\x12G\n" +
	"\rGetStreamInfo\x12\x1f.signaling.GetStreamInfoRequest\x1a\x15.signaling.StreamInfo\x12P\n" +
	"\x11NotifyStreamEnded\x12#.signaling.NotifyStreamEndedRequest\x1a\x16.google.protobuf.Empty\x12=\n" +
	"\bGetStats\x12\x16.google.protobuf.Empty\x1a\x19.signaling.SignalingStats\x12O\n" +
	"\x0fStreamSignaling\x12\x1b.signaling.SignalingMessage\x1a\x1b.signaling.SignalingMessage(\x010\x01B\x16Z\x14libs/proto/signalingb\x06proto3"

var (
	file_libs_proto_signaling_signaling_proto_rawDescOnce sync.Once
	file_libs_proto_signaling_signaling_proto_rawDescData []byte
)

func file_libs_proto_signaling_signaling_proto_rawDescGZIP() []byte {
	file_libs_proto_signaling_signaling_proto_rawDescOnce.Do(func() {
		file_libs_proto_signaling_signaling_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_libs_proto_signaling_signaling_proto_rawDesc), len(file_libs_proto_signaling_signaling_proto_rawDesc)))
	})
	return file_libs_proto_signaling_signaling_proto_rawDescData
}

var file_libs_proto_signaling_signaling_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_libs_proto_signaling_signaling_proto_goTypes = []any{
	(*SignalingMessage)(nil),         // 0: signaling.SignalingMessage
	(*GetStreamInfoRequest)(nil),     // 1: signaling.GetStreamInfoRequest
	(*StreamInfo)(nil),               // 2: signaling.StreamInfo
	(*NotifyStreamEndedRequest)(nil), // 3: signaling.NotifyStreamEndedRequest
	(*SignalingStats)(nil),           // 4: signaling.SignalingStats
	(*SessionDescription)(nil),       // 5: signaling.SessionDescription
	(*ICECandidate)(nil),             // 6: signaling.ICECandidate
	(*CreateOfferRequest)(nil),       // 7: signaling.CreateOfferRequest
	(*CreateOfferResponse)(nil),      // 8: signaling.CreateOfferResponse
	(*CreateAnswerRequest)(nil),      // 9: signaling.CreateAnswerRequest
	(*CreateAnswerResponse)(nil),     // 10: signaling.CreateAnswerResponse
	(*AddICECandidateRequest)(nil),   // 11: signaling.AddICECandidateRequest
	(*AddICECandidateResponse)(nil),  // 12: signaling.AddICECandidateResponse
	(*timestamppb.Timestamp)(nil),    // 13: google.protobuf.Timestamp
	(*emptypb.Empty)(nil),            // 14: google.protobuf.Empty
}
var file_libs_proto_signaling_signaling_proto_depIdxs = []int32{
	13, // 0: signaling.StreamInfo.created_at:type_name -> google.protobuf.Timestamp
	13, // 1: signaling.StreamInfo.updated_at:type_name -> google.protobuf.Timestamp
	13, // 2: signaling.SignalingStats.timestamp:type_name -> google.protobuf.Timestamp
	5,  // 3: signaling.CreateOfferResponse.offer:type_name -> signaling.SessionDescription
	5,  // 4: signaling.CreateAnswerRequest.offer:type_name -> signaling.SessionDescription
	5,  // 5: signaling.CreateAnswerResponse.answer:type_name -> signaling.SessionDescription
	6,  // 6: signaling.AddICECandidateRequest.candidate:type_name -> signaling.ICECandidate
	7,  // 7: signaling.SignalingService.CreateOffer:input_type -> signaling.CreateOfferRequest
	9,  // 8: signaling.SignalingService.CreateAnswer:input_type -> signaling.CreateAnswerRequest
	11, // 9: signaling.SignalingService.AddICECandidate:input_type -> signaling.AddICECandidateRequest
	0,  // 10: signaling.SignalingService.SendSignalingMessage:input_type -> signaling.SignalingMessage
	1,  // 11: signaling.SignalingService.GetStreamInfo:input_type -> signaling.GetStreamInfoRequest
	3,  // 12: signaling.SignalingService.NotifyStreamEnded:input_type -> signaling.NotifyStreamEndedRequest
	14, // 13: signaling.SignalingService.GetStats:input_type -> google.protobuf.Empty
	0,  // 14: signaling.SignalingService.StreamSignaling:input_type -> signaling.SignalingMessage
	8,  // 15: signaling.SignalingService.CreateOffer:output_type -> signaling.CreateOfferResponse
	10, // 16: signaling.SignalingService.CreateAnswer:output_type -> signaling.CreateAnswerResponse
	12, // 17: signaling.SignalingService.AddICECandidate:output_type -> signaling.AddICECandidateResponse
	14, // 18: signaling.SignalingService.SendSignalingMessage:output_type -> google.protobuf.Empty
	2,  // 19: signaling.SignalingService.GetStreamInfo:output_type -> signaling.StreamInfo
	14, // 20: signaling.SignalingService.NotifyStreamEnded:output_type -> google.protobuf.Empty
	4,  // 21: signaling.SignalingService.GetStats:output_type -> signaling.SignalingStats
	0,  // 22: signaling.SignalingService.StreamSignaling:output_type -> signaling.SignalingMessage
	15, // [15:23] is the sub-list for method output_type
	7,  // [7:15] is the sub-list for method input_type
	7,  // [7:7] is the sub-list for extension type_name
	7,  // [7:7] is the sub-list for extension extendee
	0,  // [0:7] is the sub-list for field type_name
}

func init() { file_libs_proto_signaling_signaling_proto_init() }
func file_libs_proto_signaling_signaling_proto_init() {
	if File_libs_proto_signaling_signaling_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_libs_proto_signaling_signaling_proto_rawDesc), len(file_libs_proto_signaling_signaling_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_libs_proto_signaling_signaling_proto_goTypes,
		DependencyIndexes: file_libs_proto_signaling_signaling_proto_depIdxs,
		MessageInfos:      file_libs_proto_signaling_signaling_proto_msgTypes,
	}.Build()
	File_libs_proto_signaling_signaling_proto = out.File
	file_libs_proto_signaling_signaling_proto_goTypes = nil
	file_libs_proto_signaling_signaling_proto_depIdxs = nil
}
