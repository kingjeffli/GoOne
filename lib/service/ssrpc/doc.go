// Package ssrpc provides GoOne's SSPacket RPC runtime (Phase A).
//
// Goal:
// - Keep GoOne's existing execution model (TransactionMgr + SSPacket semantics)
// - Add an IDL-driven handler layer (protoc-gen-goone) with middleware chaining
// - Provide a unified Context wrapper for business handlers
package ssrpc


