package main

import (
	"io"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeErr("read stdin", err)
		return
	}

	req := new(pluginpb.CodeGeneratorRequest)
	if err := proto.Unmarshal(in, req); err != nil {
		writeErr("unmarshal CodeGeneratorRequest", err)
		return
	}

	resp, err := Generate(req)
	if err != nil {
		writeErr("generate", err)
		return
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		writeErr("marshal CodeGeneratorResponse", err)
		return
	}

	_, _ = os.Stdout.Write(out)
}

func writeErr(stage string, err error) {
	// protoc expects CodeGeneratorResponse with error field, but if we cannot marshal,
	// fallback to stderr.
	_, _ = os.Stderr.WriteString(stage + ": " + err.Error() + "\n")
}


