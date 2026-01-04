package ssrpc

import (
	"errors"
	"fmt"

	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
)

// Error is the canonical error type for GoOne RPC handlers.
//
// Generated wrappers will map returned error to g1_protocol.ErrorCode via ToErrorCode().
type Error struct {
	Code g1_protocol.ErrorCode
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil && e.Msg != "" {
		return fmt.Sprintf("%s | %v", e.Msg, e.Err)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Msg != "" {
		return e.Msg
	}
	return e.Code.String()
}

func (e *Error) Unwrap() error { return e.Err }

func E(code g1_protocol.ErrorCode, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

func Wrap(code g1_protocol.ErrorCode, msg string, err error) *Error {
	return &Error{Code: code, Msg: msg, Err: err}
}

func ToErrorCode(err error) g1_protocol.ErrorCode {
	if err == nil {
		return g1_protocol.ErrorCode_ERR_OK
	}
	var e *Error
	if errors.As(err, &e) {
		if e.Code != 0 {
			return e.Code
		}
	}
	// default mapping
	return g1_protocol.ErrorCode_ERR_INTERNAL
}


