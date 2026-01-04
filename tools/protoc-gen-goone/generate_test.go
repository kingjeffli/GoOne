package main

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerate_SSRPCOptionAndDeterministicImports(t *testing.T) {
	// Include descriptor.proto so our custom extension can extendee MethodOptions.
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

	// options.proto (minimal in-test version; name must match ssrpcOptFilePath)
	optionsFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString(ssrpcOptFilePath),
		Package: protoString("goone.options.v1"),
		Dependency: []string{
			"google/protobuf/descriptor.proto",
		},
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/options/v1;optionsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: protoString("SsRpc"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: protoString("cmd"), Number: protoInt32(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("cmd_resp"), Number: protoInt32(2), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("one_way"), Number: protoInt32(3), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
				},
			},
		},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Name:     protoString("ssrpc"),
				Number:   protoInt32(61001),
				Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
				Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
				TypeName: protoString(".goone.options.v1.SsRpc"),
				Extendee: protoString(".google.protobuf.MethodOptions"),
			},
		},
	}

	// ext.proto (defines response type in another package)
	extFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString("test/ext/v1/ext.proto"),
		Package: protoString("test.ext.v1"),
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/ext/v1;extv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("ExtRsp")},
		},
	}

	// svc.proto (service in another package, returns cross-package type)
	svcFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString("test/svc/v1/svc.proto"),
		Package: protoString("test.svc.v1"),
		Dependency: []string{
			ssrpcOptFilePath,
			"test/ext/v1/ext.proto",
		},
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/svc/v1;svcv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.svc.v1.Req"),
						OutputType: protoString(".test.ext.v1.ExtRsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	// Build extension type and set method option.
	extType, extMsgDesc, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, optionsFD, extFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd"), protoreflect.ValueOfUint32(0x01020002))
	// leave cmd_resp=0 to default to cmd+1
	proto.SetExtension(svcFD.Service[0].Method[0].Options, extType, optMsg)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, optionsFD, extFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}

	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	if len(resp.File) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(resp.File))
	}
	gotName := resp.File[0].GetName()
	if gotName != "api/gen/test/svc/v1/svc.goone_ssrpc.gen.go" {
		t.Fatalf("unexpected output name: %q", gotName)
	}
	out := resp.File[0].GetContent()
	// Cross-package type should be qualified with extv1 alias and import should exist.
	if !contains(out, "extv1 \"github.com/Iori372552686/GoOne/api/gen/test/ext/v1\"") {
		t.Fatalf("missing extv1 import in output:\n%s", out)
	}
	if !contains(out, "(*extv1.ExtRsp, error)") {
		t.Fatalf("missing cross-package return type in output:\n%s", out)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (stringIndex(s, sub) >= 0))) }

func stringIndex(s, sub string) int {
	// small helper to avoid importing strings in this test file
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func protoInt32(i int32) *int32 { return &i }
func labelPtr(v descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &v
}
func typePtr(v descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &v
}


