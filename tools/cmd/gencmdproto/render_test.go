package main

import "testing"

func TestRenderCmdProto_AllowAliasOmittedWhenNoAlias(t *testing.T) {
	out := renderCmdProto(
		"goone.cmd.v1",
		"github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1",
		"CMD",
		true,
		[]cmdItem{
			{name: "A", val: 1},
			{name: "B", val: 2},
		},
		map[string]string{},
	)
	if contains(out, "allow_alias") {
		t.Fatalf("did not expect allow_alias when there are no aliases:\n%s", out)
	}
}

func TestRenderCmdProto_AllowAliasEmittedWhenAlias(t *testing.T) {
	out := renderCmdProto(
		"goone.cmd.v1",
		"github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1",
		"CMD",
		true,
		[]cmdItem{
			{name: "A", val: 1},
			{name: "B", val: 1},
		},
		map[string]string{},
	)
	if !contains(out, "option allow_alias = true;") {
		t.Fatalf("expected allow_alias when there are aliases:\n%s", out)
	}
}

func TestRenderCmdProto_HexVsDec(t *testing.T) {
	outHex := renderCmdProto(
		"goone.cmd.v1",
		"github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1",
		"CMD",
		true,
		[]cmdItem{{name: "A", val: 255}},
		map[string]string{},
	)
	if !contains(outHex, "A = 0xFF;") {
		t.Fatalf("expected hex literal:\n%s", outHex)
	}

	outDec := renderCmdProto(
		"goone.cmd.v1",
		"github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1",
		"CMD",
		false,
		[]cmdItem{{name: "A", val: 255}},
		map[string]string{},
	)
	if !contains(outDec, "A = 255;") {
		t.Fatalf("expected decimal literal:\n%s", outDec)
	}
}

func TestRenderCmdProto_CommentCopied(t *testing.T) {
	out := renderCmdProto(
		"goone.cmd.v1",
		"github.com/Iori372552686/GoOne/api/gen/goone/cmd/v1;cmdv1",
		"CMD",
		true,
		[]cmdItem{{name: "CMD_FOO", val: 1}},
		map[string]string{
			"CMD_FOO": "// hello\n// world",
		},
	)
	if !contains(out, "  // hello\n  // world\n  CMD_FOO = 0x1;") {
		t.Fatalf("expected comment lines above enum value:\n%s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && stringIndex(s, sub) >= 0))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}


