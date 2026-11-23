package contract

import (
	"context"
	"errors"
	"fmt"
)

// Code is a stable, numeric contract error code. The standard JSON-RPC 2.0
// codes are reused for protocol faults; the -320xx range carries contract
// semantics. Codes are append-only: a published code never changes meaning.
type Code int

// Error codes.
const (
	CodeParseError     Code = -32700
	CodeInvalidRequest Code = -32600
	CodeMethodNotFound Code = -32601
	CodeInvalidParams  Code = -32602
	CodeInternal       Code = -32603

	// CodeVersionUnsupported: api_version or contract major version mismatch.
	CodeVersionUnsupported Code = -32010
	// CodeEngineUnavailable: no engine configured, or the engine lacks model
	// credentials or provider access. Deterministic commands never return it.
	CodeEngineUnavailable Code = -32011
	// CodeCancelled: the caller cancelled the request.
	CodeCancelled Code = -32012
	// CodeTimeout: the request exceeded its deadline.
	CodeTimeout Code = -32013
	// CodeScopeViolation: a path is outside the task's allowed scope or
	// inside a prohibited path.
	CodeScopeViolation Code = -32014
	// CodeApprovalRequired: a valid approval record is required and absent.
	CodeApprovalRequired Code = -32015
	// CodePolicyDenied: the policy forbids the action outright.
	CodePolicyDenied Code = -32016
	// CodeInvalidDocument: a contract document failed validation.
	CodeInvalidDocument Code = -32017
	// CodeVerificationFailed: one or more required deterministic gates failed.
	CodeVerificationFailed Code = -32018
)

// codeNames maps codes to stable symbolic names used in reports and docs.
var codeNames = map[Code]string{
	CodeParseError:         "parse_error",
	CodeInvalidRequest:     "invalid_request",
	CodeMethodNotFound:     "method_not_found",
	CodeInvalidParams:      "invalid_params",
	CodeInternal:           "internal_error",
	CodeVersionUnsupported: "version_unsupported",
	CodeEngineUnavailable:  "engine_unavailable",
	CodeCancelled:          "cancelled",
	CodeTimeout:            "timeout",
	CodeScopeViolation:     "scope_violation",
	CodeApprovalRequired:   "approval_required",
	CodePolicyDenied:       "policy_denied",
	CodeInvalidDocument:    "invalid_document",
	CodeVerificationFailed: "verification_failed",
}

// String returns the stable symbolic name of the code.
func (c Code) String() string {
	if n, ok := codeNames[c]; ok {
		return n
	}
	return fmt.Sprintf("code(%d)", int(c))
}

// Codes returns every defined code in ascending numeric order.
func Codes() []Code {
	return []Code{
		CodeParseError, CodeInvalidRequest, CodeMethodNotFound, CodeInvalidParams, CodeInternal,
		CodeVersionUnsupported, CodeEngineUnavailable, CodeCancelled, CodeTimeout,
		CodeScopeViolation, CodeApprovalRequired, CodePolicyDenied, CodeInvalidDocument,
		CodeVerificationFailed,
	}
}

// Error is the contract error model. It is also the JSON-RPC error object
// exchanged with engines, so it round-trips across the process boundary.
type Error struct {
	Code    Code              `json:"code"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data,omitempty"`
}

// Error implements error.
func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// Is matches any *Error with the same code, so errors.Is(err, ErrTimeout)
// works for errors decoded from the wire.
func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Code == e.Code
}

// Errorf builds a contract error.
func Errorf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Sentinel errors for errors.Is matching.
var (
	ErrVersionUnsupported = &Error{Code: CodeVersionUnsupported, Message: "version unsupported"}
	ErrEngineUnavailable  = &Error{Code: CodeEngineUnavailable, Message: "engine unavailable"}
	ErrCancelled          = &Error{Code: CodeCancelled, Message: "cancelled"}
	ErrTimeout            = &Error{Code: CodeTimeout, Message: "timeout"}
	ErrScopeViolation     = &Error{Code: CodeScopeViolation, Message: "scope violation"}
	ErrApprovalRequired   = &Error{Code: CodeApprovalRequired, Message: "approval required"}
	ErrPolicyDenied       = &Error{Code: CodePolicyDenied, Message: "policy denied"}
	ErrInvalidDocument    = &Error{Code: CodeInvalidDocument, Message: "invalid document"}
	ErrMethodNotFound     = &Error{Code: CodeMethodNotFound, Message: "method not found"}
)

// AsError converts any error into a contract *Error, mapping context
// cancellation and deadlines onto CodeCancelled and CodeTimeout.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return &Error{Code: CodeTimeout, Message: err.Error()}
	case errors.Is(err, context.Canceled):
		return &Error{Code: CodeCancelled, Message: err.Error()}
	}
	return &Error{Code: CodeInternal, Message: err.Error()}
}
