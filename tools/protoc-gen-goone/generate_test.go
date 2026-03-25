package main

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/emptypb"
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
					{Name: protoString("cmd_name"), Number: protoInt32(5), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
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
	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, optionsFD, extFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd"), protoreflect.ValueOfUint32(0x01020002))
	// leave cmd_resp=0 to default to cmd+1
	svcFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

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

func Test_readSsRpcOption_DecodesFromUnknownFields(t *testing.T) {
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
					{Name: protoString("cmd_name"), Number: protoInt32(5), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
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

	extType, extMsgDesc, extNum, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, optionsFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}

	// Simulate protoc: custom options are encoded into unknown fields of MethodOptions.
	optsWithExt := &descriptorpb.MethodOptions{}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd"), protoreflect.ValueOfUint32(0x01020003))
	optMsg.Set(extMsgDesc.Fields().ByName("one_way"), protoreflect.ValueOfBool(true))
	proto.SetExtension(optsWithExt, extType, optMsg)

	wire, err := proto.Marshal(optsWithExt)
	if err != nil {
		t.Fatalf("marshal method options err: %v", err)
	}
	optsUnknown := &descriptorpb.MethodOptions{}
	if err := proto.Unmarshal(wire, optsUnknown); err != nil {
		t.Fatalf("unmarshal method options err: %v", err)
	}

	got, ok, err := readSsRpcOption(optsUnknown, extType, extMsgDesc, extNum)
	if err != nil {
		t.Fatalf("readSsRpcOption err: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true when extension exists in unknown fields")
	}
	if got.cmd != 0x01020003 || !got.oneWay {
		t.Fatalf("unexpected parsed option: %+v", got)
	}
}

func TestGenerate_SSRPCOption_CmdName(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

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
					{Name: protoString("cmd_name"), Number: protoInt32(5), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/cmdname/v1/svc.proto"),
		Package:    protoString("test.cmdname.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/cmdname/v1;cmdnamev1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.cmdname.v1.Req"),
						OutputType: protoString(".test.cmdname.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd_name"), protoreflect.ValueOfString("CMD_MAIN_LOGIN_REQ"))
	svcFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}

	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	out := resp.File[0].GetContent()
	if !contains(out, "mgr.RegisterCmd(g1_protocol.CMD_MAIN_LOGIN_REQ, ssrpc.WrapUnary") {
		t.Fatalf("expected generator to use WrapUnary and enum constant, got:\n%s", out)
	}
	if contains(out, "\"github.com/golang/protobuf/proto\"") {
		t.Fatalf("did not expect generated file to import github.com/golang/protobuf/proto:\n%s", out)
	}
}

func TestGenerate_SSRPCOption_CmdEnum(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

	// Minimal cmd.proto (enum) so options.proto can reference a real enum type.
	cmdFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString("goone/cmd/v1/cmd.proto"),
		Package: protoString("goone.cmd.v1"),
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1")},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: protoString("CMD"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: protoString("CMD_UNSPECIFIED"), Number: protoInt32(0)},
					{Name: protoString("CMD_MAIN_LOGIN_REQ"), Number: protoInt32(0x01020005)},
				},
			},
		},
	}

	optionsFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString(ssrpcOptFilePath),
		Package: protoString("goone.options.v1"),
		Dependency: []string{
			"google/protobuf/descriptor.proto",
			"goone/cmd/v1/cmd.proto",
		},
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/options/v1;optionsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: protoString("SsRpc"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: protoString("cmd"), Number: protoInt32(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("cmd_resp"), Number: protoInt32(2), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("one_way"), Number: protoInt32(3), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{
						Name:     protoString("cmd_enum"),
						Number:   protoInt32(6),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
						TypeName: protoString(".goone.cmd.v1.CMD"),
					},
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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/cmdenum/v1/svc.proto"),
		Package:    protoString("test.cmdenum.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/cmdenum/v1;cmdenumv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.cmdenum.v1.Req"),
						OutputType: protoString(".test.cmdenum.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd_enum"), protoreflect.ValueOfEnum(protoreflect.EnumNumber(0x01020005)))

	// Simulate protoc: custom options are encoded into unknown fields of MethodOptions.
	optsWithExt := &descriptorpb.MethodOptions{}
	proto.SetExtension(optsWithExt, extType, optMsg)
	wire, err := proto.Marshal(optsWithExt)
	if err != nil {
		t.Fatalf("marshal method options err: %v", err)
	}
	optsUnknown := &descriptorpb.MethodOptions{}
	if err := proto.Unmarshal(wire, optsUnknown); err != nil {
		t.Fatalf("unmarshal method options err: %v", err)
	}
	svcFD.Service[0].Method[0].Options = optsUnknown

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}
	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	if len(resp.File) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(resp.File))
	}
	out := resp.File[0].GetContent()
	if !contains(out, "mgr.RegisterCmd(g1_protocol.CMD(0x1020005), ssrpc.WrapUnary") {
		t.Fatalf("expected generator to use cmd_enum numeric value, got:\n%s", out)
	}
}

func TestGenerate_SSRPCOption_AuthSignTraceTags(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)
	cmdFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString("goone/cmd/v1/cmd.proto"),
		Package: protoString("goone.cmd.v1"),
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1")},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: protoString("CMD"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: protoString("CMD_UNSPECIFIED"), Number: protoInt32(0)},
				},
			},
		},
	}

	optionsFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString(ssrpcOptFilePath),
		Package: protoString("goone.options.v1"),
		Dependency: []string{
			"google/protobuf/descriptor.proto",
			"goone/cmd/v1/cmd.proto",
		},
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/options/v1;optionsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: protoString("SsRpc"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: protoString("cmd"), Number: protoInt32(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("cmd_resp"), Number: protoInt32(2), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("one_way"), Number: protoInt32(3), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{Name: protoString("auth"), Number: protoInt32(7), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{Name: protoString("sign"), Number: protoInt32(8), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{Name: protoString("trace_tags"), Number: protoInt32(9), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/opts/v1/svc.proto"),
		Package:    protoString("test.opts.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/opts/v1;optsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.opts.v1.Req"),
						OutputType: protoString(".test.opts.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd"), protoreflect.ValueOfUint32(0x01020002))
	optMsg.Set(extMsgDesc.Fields().ByName("auth"), protoreflect.ValueOfBool(true))
	optMsg.Set(extMsgDesc.Fields().ByName("sign"), protoreflect.ValueOfBool(true))
	optMsg.Mutable(extMsgDesc.Fields().ByName("trace_tags")).List().Append(protoreflect.ValueOfString("b=2"))
	optMsg.Mutable(extMsgDesc.Fields().ByName("trace_tags")).List().Append(protoreflect.ValueOfString("a=1"))
	svcFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}
	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	out := resp.File[0].GetContent()
	if !contains(out, "Auth: true") || !contains(out, "Sign: true") {
		t.Fatalf("expected Auth/Sign fields in MethodDesc, got:\n%s", out)
	}
	// Deterministic order should be a then b.
	if !contains(out, "TraceTags: map[string]string{\"a\": \"1\", \"b\": \"2\", },") {
		t.Fatalf("expected TraceTags map in deterministic order, got:\n%s", out)
	}
}

func TestGenerate_HTTPBinding_GinRegister(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

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
					{Name: protoString("http_path"), Number: protoInt32(20), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
					{Name: protoString("http_method"), Number: protoInt32(21), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/http/v1/svc.proto"),
		Package:    protoString("test.http.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/http/v1;httpv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.http.v1.Req"),
						OutputType: protoString(".test.http.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd"), protoreflect.ValueOfUint32(0x01020002))
	optMsg.Set(extMsgDesc.Fields().ByName("http_path"), protoreflect.ValueOfString("/v1/test/do"))
	optMsg.Set(extMsgDesc.Fields().ByName("http_method"), protoreflect.ValueOfString("get"))
	svcFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}
	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	if len(resp.File) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(resp.File))
	}
	out := resp.File[0].GetContent()
	if !contains(out, "func RegisterSvcToGin(r gin.IRoutes") {
		t.Fatalf("expected gin register function, got:\n%s", out)
	}
	if !contains(out, "r.Handle(\"GET\", \"/v1/test/do\", ssrpc.WrapHTTPGin") {
		t.Fatalf("expected r.Handle with normalized method/path, got:\n%s", out)
	}
}

func TestGenerate_HTTPOnly_NoCmd_SkipsTransactionMgr(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

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
					{Name: protoString("http_path"), Number: protoInt32(20), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
					{Name: protoString("http_method"), Number: protoInt32(21), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/http/v1/svc2.proto"),
		Package:    protoString("test.http.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/http/v1;httpv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req2")},
			{Name: protoString("Rsp2")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc2"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do2"),
						InputType:  protoString(".test.http.v1.Req2"),
						OutputType: protoString(".test.http.v1.Rsp2"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	// NOTE: intentionally omit cmd/cmd_name/cmd_enum
	optMsg.Set(extMsgDesc.Fields().ByName("http_path"), protoreflect.ValueOfString("/v1/test/do2"))
	optMsg.Set(extMsgDesc.Fields().ByName("http_method"), protoreflect.ValueOfString("POST"))
	svcFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}
	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	if len(resp.File) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(resp.File))
	}
	out := resp.File[0].GetContent()
	if contains(out, "RegisterSvc2ToTransactionMgr") {
		t.Fatalf("expected no transaction mgr register for http-only method, got:\n%s", out)
	}
	if contains(out, "lib/service/transaction") || contains(out, "game_protocol/protocol") {
		t.Fatalf("expected no transaction/g1_protocol imports for http-only method, got:\n%s", out)
	}
	if !contains(out, "func RegisterSvc2ToGin(r gin.IRoutes") {
		t.Fatalf("expected gin register function, got:\n%s", out)
	}
	if !contains(out, "func RegisterSvc2ToDispatcher(d *ssrpc.Dispatcher") {
		t.Fatalf("expected dispatcher register function, got:\n%s", out)
	}
	if !contains(out, "d.RegisterHTTP(\"POST\", \"/v1/test/do2\", ssrpc.WrapHTTPGin") {
		t.Fatalf("expected dispatcher http registration, got:\n%s", out)
	}
	if !contains(out, "Cmd: 0,") {
		t.Fatalf("expected Cmd: 0 for http-only binding, got:\n%s", out)
	}
}

func TestGenerate_SSRPCOption_UIDLockInject(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

	// Minimal cmd.proto (enum) so options.proto can reference a real enum type.
	cmdFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString("goone/cmd/v1/cmd.proto"),
		Package: protoString("goone.cmd.v1"),
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1")},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: protoString("CMD"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: protoString("CMD_UNSPECIFIED"), Number: protoInt32(0)},
				},
			},
		},
	}

	optionsFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString(ssrpcOptFilePath),
		Package: protoString("goone.options.v1"),
		Dependency: []string{
			"google/protobuf/descriptor.proto",
			"goone/cmd/v1/cmd.proto",
		},
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/options/v1;optionsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: protoString("SsRpc"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: protoString("cmd"), Number: protoInt32(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("cmd_resp"), Number: protoInt32(2), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("one_way"), Number: protoInt32(3), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{Name: protoString("uid_lock"), Number: protoInt32(4), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{Name: protoString("cmd_name"), Number: protoInt32(5), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
					{
						Name:     protoString("cmd_enum"),
						Number:   protoInt32(6),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
						TypeName: protoString(".goone.cmd.v1.CMD"),
					},
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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/uidlock/v1/svc.proto"),
		Package:    protoString("test.uidlock.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/uidlock/v1;uidlockv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.uidlock.v1.Req"),
						OutputType: protoString(".test.uidlock.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd_name"), protoreflect.ValueOfString("CMD_MAIN_LOGIN_REQ"))
	optMsg.Set(extMsgDesc.Fields().ByName("uid_lock"), protoreflect.ValueOfBool(true))
	svcFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}

	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	out := resp.File[0].GetContent()
	// Generator should express UIDLock via MethodDesc so runtime can inject middleware.
	if !contains(out, "UIDLock: true") {
		t.Fatalf("expected uid_lock to set UIDLock: true in MethodDesc, got:\n%s", out)
	}
}

func TestGenerate_SSRPCOption_TimeoutMs(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

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
					{Name: protoString("timeout_ms"), Number: protoInt32(10), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/timeout/v1/svc.proto"),
		Package:    protoString("test.timeout.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/timeout/v1;timeoutv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.timeout.v1.Req"),
						OutputType: protoString(".test.timeout.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd"), protoreflect.ValueOfUint32(0x01020003))
	optMsg.Set(extMsgDesc.Fields().ByName("timeout_ms"), protoreflect.ValueOfUint32(1500))
	svcFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}

	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	out := resp.File[0].GetContent()
	if !contains(out, "\"time\"") {
		t.Fatalf("expected time import when timeout_ms is set, got:\n%s", out)
	}
	if !contains(out, "Timeout: 1500 * time.Millisecond") {
		t.Fatalf("expected timeout_ms to set MethodDesc.Timeout, got:\n%s", out)
	}
}

func TestGenerate_Imports_EmptyPBOnlyWhenUsed(t *testing.T) {
	// Include descriptor.proto so our custom extension can extendee MethodOptions.
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)
	emptyFD := protodesc.ToFileDescriptorProto(emptypb.File_google_protobuf_empty_proto)

	// Minimal cmd.proto (enum) so options.proto can reference a real enum type.
	cmdFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString("goone/cmd/v1/cmd.proto"),
		Package: protoString("goone.cmd.v1"),
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1")},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: protoString("CMD"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: protoString("CMD_UNSPECIFIED"), Number: protoInt32(0)},
				},
			},
		},
	}

	optionsFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString(ssrpcOptFilePath),
		Package: protoString("goone.options.v1"),
		Dependency: []string{
			"google/protobuf/descriptor.proto",
			"goone/cmd/v1/cmd.proto",
		},
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/goone/options/v1;optionsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: protoString("SsRpc"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: protoString("cmd"), Number: protoInt32(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("cmd_resp"), Number: protoInt32(2), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32)},
					{Name: protoString("one_way"), Number: protoInt32(3), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{Name: protoString("uid_lock"), Number: protoInt32(4), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL)},
					{Name: protoString("cmd_name"), Number: protoInt32(5), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
					{
						Name:     protoString("cmd_enum"),
						Number:   protoInt32(6),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
						TypeName: protoString(".goone.cmd.v1.CMD"),
					},
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

	// Case 1: output type is a local message -> should NOT import emptypb.
	svcNoEmptyFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/imports/v1/no_empty.proto"),
		Package:    protoString("test.imports.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/imports/v1;importsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.imports.v1.Req"),
						OutputType: protoString(".test.imports.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType, extMsgDesc, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcNoEmptyFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg := dynamicpb.NewMessage(extMsgDesc)
	optMsg.Set(extMsgDesc.Fields().ByName("cmd_name"), protoreflect.ValueOfString("CMD_MAIN_LOGIN_REQ"))
	svcNoEmptyFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType, optMsg)

	req1 := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcNoEmptyFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, cmdFD, optionsFD, svcNoEmptyFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}
	resp1, err := Generate(req1)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	out1 := resp1.File[0].GetContent()
	if contains(out1, "emptypb \"google.golang.org/protobuf/types/known/emptypb\"") {
		t.Fatalf("did not expect emptypb import when google.protobuf.Empty is not used:\n%s", out1)
	}

	// Case 2: output type is google.protobuf.Empty -> SHOULD import emptypb.
	svcEmptyFD := &descriptorpb.FileDescriptorProto{
		Name:    protoString("test/imports/v1/with_empty.proto"),
		Package: protoString("test.imports.v1"),
		Dependency: []string{
			ssrpcOptFilePath,
			"google/protobuf/empty.proto",
		},
		Options: &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/imports/v1;importsv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc2"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do2"),
						InputType:  protoString(".test.imports.v1.Req"),
						OutputType: protoString(".google.protobuf.Empty"),
						Options:    &descriptorpb.MethodOptions{},
					},
				},
			},
		},
	}

	extType2, extMsgDesc2, _, err := buildSsRpcExtension([]*descriptorpb.FileDescriptorProto{descFD, emptyFD, cmdFD, optionsFD, svcEmptyFD})
	if err != nil {
		t.Fatalf("buildSsRpcExtension err: %v", err)
	}
	optMsg2 := dynamicpb.NewMessage(extMsgDesc2)
	optMsg2.Set(extMsgDesc2.Fields().ByName("cmd_name"), protoreflect.ValueOfString("CMD_MAIN_LOGIN_REQ"))
	svcEmptyFD.Service[0].Method[0].Options = mustMethodOptionsUnknown(t, extType2, optMsg2)

	req2 := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcEmptyFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, emptyFD, cmdFD, optionsFD, svcEmptyFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}
	resp2, err := Generate(req2)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	out2 := resp2.File[0].GetContent()
	if !contains(out2, "emptypb \"google.golang.org/protobuf/types/known/emptypb\"") {
		t.Fatalf("expected emptypb import when google.protobuf.Empty is used:\n%s", out2)
	}
}

func TestGenerate_SkipFileWithoutSSRPCMethods(t *testing.T) {
	descFD := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)

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

	svcFD := &descriptorpb.FileDescriptorProto{
		Name:       protoString("test/skip/v1/skip.proto"),
		Package:    protoString("test.skip.v1"),
		Dependency: []string{ssrpcOptFilePath},
		Options:    &descriptorpb.FileOptions{GoPackage: protoString("github.com/Iori372552686/GoOne/api/gen/test/skip/v1;skipv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: protoString("Req")},
			{Name: protoString("Rsp")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoString("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoString("Do"),
						InputType:  protoString(".test.skip.v1.Req"),
						OutputType: protoString(".test.skip.v1.Rsp"),
						Options:    &descriptorpb.MethodOptions{}, // intentionally no ssrpc option
					},
				},
			},
		},
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{svcFD.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descFD, optionsFD, svcFD},
		Parameter:      protoString("paths=import,module=github.com/Iori372552686/GoOne"),
	}

	resp, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	if len(resp.File) != 0 {
		t.Fatalf("expected no generated file for a proto without any ssrpc methods, got %d", len(resp.File))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (stringIndex(s, sub) >= 0)))
}

func stringIndex(s, sub string) int {
	// small helper to avoid importing strings in this test file
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func mustMethodOptionsUnknown(t *testing.T, extType protoreflect.ExtensionType, optMsg proto.Message) *descriptorpb.MethodOptions {
	t.Helper()
	optsWithExt := &descriptorpb.MethodOptions{}
	proto.SetExtension(optsWithExt, extType, optMsg)

	wire, err := proto.Marshal(optsWithExt)
	if err != nil {
		t.Fatalf("marshal method options err: %v", err)
	}
	optsUnknown := &descriptorpb.MethodOptions{}
	if err := proto.Unmarshal(wire, optsUnknown); err != nil {
		t.Fatalf("unmarshal method options err: %v", err)
	}
	return optsUnknown
}

func protoInt32(i int32) *int32    { return &i }
func protoString(s string) *string { return &s }
func labelPtr(v descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &v
}
func typePtr(v descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &v
}
