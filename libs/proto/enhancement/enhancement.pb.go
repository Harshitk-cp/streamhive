// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.28.2
// source: libs/proto/enhancement/enhancement.proto

package enhancement

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

// Frame type enumeration
type FrameType int32

const (
	FrameType_UNKNOWN  FrameType = 0
	FrameType_VIDEO    FrameType = 1
	FrameType_AUDIO    FrameType = 2
	FrameType_METADATA FrameType = 3
)

// Enum value maps for FrameType.
var (
	FrameType_name = map[int32]string{
		0: "UNKNOWN",
		1: "VIDEO",
		2: "AUDIO",
		3: "METADATA",
	}
	FrameType_value = map[string]int32{
		"UNKNOWN":  0,
		"VIDEO":    1,
		"AUDIO":    2,
		"METADATA": 3,
	}
)

func (x FrameType) Enum() *FrameType {
	p := new(FrameType)
	*p = x
	return p
}

func (x FrameType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (FrameType) Descriptor() protoreflect.EnumDescriptor {
	return file_libs_proto_enhancement_enhancement_proto_enumTypes[0].Descriptor()
}

func (FrameType) Type() protoreflect.EnumType {
	return &file_libs_proto_enhancement_enhancement_proto_enumTypes[0]
}

func (x FrameType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use FrameType.Descriptor instead.
func (FrameType) EnumDescriptor() ([]byte, []int) {
	return file_libs_proto_enhancement_enhancement_proto_rawDescGZIP(), []int{0}
}

// Frame represents a video or audio frame
type Frame struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	FrameId       string                 `protobuf:"bytes,2,opt,name=frame_id,json=frameId,proto3" json:"frame_id,omitempty"`
	Timestamp     *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Type          FrameType              `protobuf:"varint,4,opt,name=type,proto3,enum=enhancement.FrameType" json:"type,omitempty"`
	Data          []byte                 `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,6,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Sequence      int64                  `protobuf:"varint,7,opt,name=sequence,proto3" json:"sequence,omitempty"`
	IsKeyFrame    bool                   `protobuf:"varint,8,opt,name=is_key_frame,json=isKeyFrame,proto3" json:"is_key_frame,omitempty"`
	Duration      int32                  `protobuf:"varint,9,opt,name=duration,proto3" json:"duration,omitempty"` // Duration in milliseconds
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Frame) Reset() {
	*x = Frame{}
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Frame) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Frame) ProtoMessage() {}

func (x *Frame) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Frame.ProtoReflect.Descriptor instead.
func (*Frame) Descriptor() ([]byte, []int) {
	return file_libs_proto_enhancement_enhancement_proto_rawDescGZIP(), []int{0}
}

func (x *Frame) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *Frame) GetFrameId() string {
	if x != nil {
		return x.FrameId
	}
	return ""
}

func (x *Frame) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Frame) GetType() FrameType {
	if x != nil {
		return x.Type
	}
	return FrameType_UNKNOWN
}

func (x *Frame) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *Frame) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Frame) GetSequence() int64 {
	if x != nil {
		return x.Sequence
	}
	return 0
}

func (x *Frame) GetIsKeyFrame() bool {
	if x != nil {
		return x.IsKeyFrame
	}
	return false
}

func (x *Frame) GetDuration() int32 {
	if x != nil {
		return x.Duration
	}
	return 0
}

// EnhanceFramesRequest is used to enhance a batch of frames
type EnhanceFramesRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	Frames        []*Frame               `protobuf:"bytes,2,rep,name=frames,proto3" json:"frames,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EnhanceFramesRequest) Reset() {
	*x = EnhanceFramesRequest{}
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EnhanceFramesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnhanceFramesRequest) ProtoMessage() {}

func (x *EnhanceFramesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnhanceFramesRequest.ProtoReflect.Descriptor instead.
func (*EnhanceFramesRequest) Descriptor() ([]byte, []int) {
	return file_libs_proto_enhancement_enhancement_proto_rawDescGZIP(), []int{1}
}

func (x *EnhanceFramesRequest) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *EnhanceFramesRequest) GetFrames() []*Frame {
	if x != nil {
		return x.Frames
	}
	return nil
}

// EnhanceFramesResponse contains the result of enhancing frames
type EnhanceFramesResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Status        string                 `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EnhanceFramesResponse) Reset() {
	*x = EnhanceFramesResponse{}
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EnhanceFramesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnhanceFramesResponse) ProtoMessage() {}

func (x *EnhanceFramesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnhanceFramesResponse.ProtoReflect.Descriptor instead.
func (*EnhanceFramesResponse) Descriptor() ([]byte, []int) {
	return file_libs_proto_enhancement_enhancement_proto_rawDescGZIP(), []int{2}
}

func (x *EnhanceFramesResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *EnhanceFramesResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

// EnhancementConfig represents the configuration for enhancements
type EnhancementConfig struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	StreamId         string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	NoiseReduction   bool                   `protobuf:"varint,2,opt,name=noise_reduction,json=noiseReduction,proto3" json:"noise_reduction,omitempty"`
	AutoBrightness   bool                   `protobuf:"varint,3,opt,name=auto_brightness,json=autoBrightness,proto3" json:"auto_brightness,omitempty"`
	AutoContrast     bool                   `protobuf:"varint,4,opt,name=auto_contrast,json=autoContrast,proto3" json:"auto_contrast,omitempty"`
	ObjectDetection  bool                   `protobuf:"varint,5,opt,name=object_detection,json=objectDetection,proto3" json:"object_detection,omitempty"`
	FaceDetection    bool                   `protobuf:"varint,6,opt,name=face_detection,json=faceDetection,proto3" json:"face_detection,omitempty"`
	SceneDetection   bool                   `protobuf:"varint,7,opt,name=scene_detection,json=sceneDetection,proto3" json:"scene_detection,omitempty"`
	AdditionalParams map[string]string      `protobuf:"bytes,8,rep,name=additional_params,json=additionalParams,proto3" json:"additional_params,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *EnhancementConfig) Reset() {
	*x = EnhancementConfig{}
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EnhancementConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnhancementConfig) ProtoMessage() {}

func (x *EnhancementConfig) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnhancementConfig.ProtoReflect.Descriptor instead.
func (*EnhancementConfig) Descriptor() ([]byte, []int) {
	return file_libs_proto_enhancement_enhancement_proto_rawDescGZIP(), []int{3}
}

func (x *EnhancementConfig) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

func (x *EnhancementConfig) GetNoiseReduction() bool {
	if x != nil {
		return x.NoiseReduction
	}
	return false
}

func (x *EnhancementConfig) GetAutoBrightness() bool {
	if x != nil {
		return x.AutoBrightness
	}
	return false
}

func (x *EnhancementConfig) GetAutoContrast() bool {
	if x != nil {
		return x.AutoContrast
	}
	return false
}

func (x *EnhancementConfig) GetObjectDetection() bool {
	if x != nil {
		return x.ObjectDetection
	}
	return false
}

func (x *EnhancementConfig) GetFaceDetection() bool {
	if x != nil {
		return x.FaceDetection
	}
	return false
}

func (x *EnhancementConfig) GetSceneDetection() bool {
	if x != nil {
		return x.SceneDetection
	}
	return false
}

func (x *EnhancementConfig) GetAdditionalParams() map[string]string {
	if x != nil {
		return x.AdditionalParams
	}
	return nil
}

// GetEnhancementConfigRequest is used to get the enhancement configuration for a stream
type GetEnhancementConfigRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	StreamId      string                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetEnhancementConfigRequest) Reset() {
	*x = GetEnhancementConfigRequest{}
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetEnhancementConfigRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEnhancementConfigRequest) ProtoMessage() {}

func (x *GetEnhancementConfigRequest) ProtoReflect() protoreflect.Message {
	mi := &file_libs_proto_enhancement_enhancement_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEnhancementConfigRequest.ProtoReflect.Descriptor instead.
func (*GetEnhancementConfigRequest) Descriptor() ([]byte, []int) {
	return file_libs_proto_enhancement_enhancement_proto_rawDescGZIP(), []int{4}
}

func (x *GetEnhancementConfigRequest) GetStreamId() string {
	if x != nil {
		return x.StreamId
	}
	return ""
}

var File_libs_proto_enhancement_enhancement_proto protoreflect.FileDescriptor

const file_libs_proto_enhancement_enhancement_proto_rawDesc = "" +
	"\n" +
	"(libs/proto/enhancement/enhancement.proto\x12\venhancement\x1a\x1fgoogle/protobuf/timestamp.proto\x1a\x1bgoogle/protobuf/empty.proto\"\x8e\x03\n" +
	"\x05Frame\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12\x19\n" +
	"\bframe_id\x18\x02 \x01(\tR\aframeId\x128\n" +
	"\ttimestamp\x18\x03 \x01(\v2\x1a.google.protobuf.TimestampR\ttimestamp\x12*\n" +
	"\x04type\x18\x04 \x01(\x0e2\x16.enhancement.FrameTypeR\x04type\x12\x12\n" +
	"\x04data\x18\x05 \x01(\fR\x04data\x12<\n" +
	"\bmetadata\x18\x06 \x03(\v2 .enhancement.Frame.MetadataEntryR\bmetadata\x12\x1a\n" +
	"\bsequence\x18\a \x01(\x03R\bsequence\x12 \n" +
	"\fis_key_frame\x18\b \x01(\bR\n" +
	"isKeyFrame\x12\x1a\n" +
	"\bduration\x18\t \x01(\x05R\bduration\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"_\n" +
	"\x14EnhanceFramesRequest\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12*\n" +
	"\x06frames\x18\x02 \x03(\v2\x12.enhancement.FrameR\x06frames\"E\n" +
	"\x15EnhanceFramesResponse\x12\x16\n" +
	"\x06status\x18\x01 \x01(\tR\x06status\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"\xca\x03\n" +
	"\x11EnhancementConfig\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId\x12'\n" +
	"\x0fnoise_reduction\x18\x02 \x01(\bR\x0enoiseReduction\x12'\n" +
	"\x0fauto_brightness\x18\x03 \x01(\bR\x0eautoBrightness\x12#\n" +
	"\rauto_contrast\x18\x04 \x01(\bR\fautoContrast\x12)\n" +
	"\x10object_detection\x18\x05 \x01(\bR\x0fobjectDetection\x12%\n" +
	"\x0eface_detection\x18\x06 \x01(\bR\rfaceDetection\x12'\n" +
	"\x0fscene_detection\x18\a \x01(\bR\x0esceneDetection\x12a\n" +
	"\x11additional_params\x18\b \x03(\v24.enhancement.EnhancementConfig.AdditionalParamsEntryR\x10additionalParams\x1aC\n" +
	"\x15AdditionalParamsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\":\n" +
	"\x1bGetEnhancementConfigRequest\x12\x1b\n" +
	"\tstream_id\x18\x01 \x01(\tR\bstreamId*<\n" +
	"\tFrameType\x12\v\n" +
	"\aUNKNOWN\x10\x00\x12\t\n" +
	"\x05VIDEO\x10\x01\x12\t\n" +
	"\x05AUDIO\x10\x02\x12\f\n" +
	"\bMETADATA\x10\x032\xa1\x02\n" +
	"\x12EnhancementService\x12V\n" +
	"\rEnhanceFrames\x12!.enhancement.EnhanceFramesRequest\x1a\".enhancement.EnhanceFramesResponse\x12`\n" +
	"\x14GetEnhancementConfig\x12(.enhancement.GetEnhancementConfigRequest\x1a\x1e.enhancement.EnhancementConfig\x12Q\n" +
	"\x17UpdateEnhancementConfig\x12\x1e.enhancement.EnhancementConfig\x1a\x16.google.protobuf.EmptyB\x18Z\x16libs/proto/enhancementb\x06proto3"

var (
	file_libs_proto_enhancement_enhancement_proto_rawDescOnce sync.Once
	file_libs_proto_enhancement_enhancement_proto_rawDescData []byte
)

func file_libs_proto_enhancement_enhancement_proto_rawDescGZIP() []byte {
	file_libs_proto_enhancement_enhancement_proto_rawDescOnce.Do(func() {
		file_libs_proto_enhancement_enhancement_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_libs_proto_enhancement_enhancement_proto_rawDesc), len(file_libs_proto_enhancement_enhancement_proto_rawDesc)))
	})
	return file_libs_proto_enhancement_enhancement_proto_rawDescData
}

var file_libs_proto_enhancement_enhancement_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_libs_proto_enhancement_enhancement_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_libs_proto_enhancement_enhancement_proto_goTypes = []any{
	(FrameType)(0),                      // 0: enhancement.FrameType
	(*Frame)(nil),                       // 1: enhancement.Frame
	(*EnhanceFramesRequest)(nil),        // 2: enhancement.EnhanceFramesRequest
	(*EnhanceFramesResponse)(nil),       // 3: enhancement.EnhanceFramesResponse
	(*EnhancementConfig)(nil),           // 4: enhancement.EnhancementConfig
	(*GetEnhancementConfigRequest)(nil), // 5: enhancement.GetEnhancementConfigRequest
	nil,                                 // 6: enhancement.Frame.MetadataEntry
	nil,                                 // 7: enhancement.EnhancementConfig.AdditionalParamsEntry
	(*timestamppb.Timestamp)(nil),       // 8: google.protobuf.Timestamp
	(*emptypb.Empty)(nil),               // 9: google.protobuf.Empty
}
var file_libs_proto_enhancement_enhancement_proto_depIdxs = []int32{
	8, // 0: enhancement.Frame.timestamp:type_name -> google.protobuf.Timestamp
	0, // 1: enhancement.Frame.type:type_name -> enhancement.FrameType
	6, // 2: enhancement.Frame.metadata:type_name -> enhancement.Frame.MetadataEntry
	1, // 3: enhancement.EnhanceFramesRequest.frames:type_name -> enhancement.Frame
	7, // 4: enhancement.EnhancementConfig.additional_params:type_name -> enhancement.EnhancementConfig.AdditionalParamsEntry
	2, // 5: enhancement.EnhancementService.EnhanceFrames:input_type -> enhancement.EnhanceFramesRequest
	5, // 6: enhancement.EnhancementService.GetEnhancementConfig:input_type -> enhancement.GetEnhancementConfigRequest
	4, // 7: enhancement.EnhancementService.UpdateEnhancementConfig:input_type -> enhancement.EnhancementConfig
	3, // 8: enhancement.EnhancementService.EnhanceFrames:output_type -> enhancement.EnhanceFramesResponse
	4, // 9: enhancement.EnhancementService.GetEnhancementConfig:output_type -> enhancement.EnhancementConfig
	9, // 10: enhancement.EnhancementService.UpdateEnhancementConfig:output_type -> google.protobuf.Empty
	8, // [8:11] is the sub-list for method output_type
	5, // [5:8] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_libs_proto_enhancement_enhancement_proto_init() }
func file_libs_proto_enhancement_enhancement_proto_init() {
	if File_libs_proto_enhancement_enhancement_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_libs_proto_enhancement_enhancement_proto_rawDesc), len(file_libs_proto_enhancement_enhancement_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_libs_proto_enhancement_enhancement_proto_goTypes,
		DependencyIndexes: file_libs_proto_enhancement_enhancement_proto_depIdxs,
		EnumInfos:         file_libs_proto_enhancement_enhancement_proto_enumTypes,
		MessageInfos:      file_libs_proto_enhancement_enhancement_proto_msgTypes,
	}.Build()
	File_libs_proto_enhancement_enhancement_proto = out.File
	file_libs_proto_enhancement_enhancement_proto_goTypes = nil
	file_libs_proto_enhancement_enhancement_proto_depIdxs = nil
}
