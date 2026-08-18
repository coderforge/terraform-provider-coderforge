// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package client

import (
	"errors"
	"fmt"
	"net/http"
)

// Error codes returned by terraform-api. They are part of the contract: the
// service promises they stay stable, and the provider branches on them rather
// than on message text, which is free to change.
const (
	CodeInvalidRequest      = "invalid_request"
	CodeUnauthorized        = "unauthorized"
	CodeForbidden           = "forbidden"
	CodeNotFound            = "not_found"
	CodeConflict            = "conflict"
	CodeUpstreamError       = "upstream_error"
	CodeOperationFailed     = "operation_failed"
	CodeUpstreamUnavailable = "upstream_unavailable"
	CodeTimeout             = "timeout"
)

// ErrNotFound is what the resource Read methods test for. A resource that has
// been removed outside Terraform must be dropped from state rather than fail
// the run, and this is the sentinel that says so.
var ErrNotFound = errors.New("resource not found")

// APIError carries a structured failure from terraform-api. The message is
// written to be shown to a practitioner verbatim, and RequestID ties it to a
// line in the service log.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Resource   string
	Details    any
	RequestID  string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = fmt.Sprintf("the API returned HTTP %d", e.StatusCode)
	}
	if e.RequestID != "" {
		return fmt.Sprintf("%s (request id: %s)", msg, e.RequestID)
	}
	return msg
}

// Is lets errors.Is(err, ErrNotFound) work against the code the service sent,
// so callers never have to compare status codes themselves.
func (e *APIError) Is(target error) bool {
	if target == ErrNotFound {
		return e.Code == CodeNotFound || e.StatusCode == http.StatusNotFound
	}
	return false
}

// IsNotFound reports whether an error means "this resource no longer exists".
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsRetryable reports whether repeating the request could plausibly succeed.
// A timeout is deliberately excluded: the operation behind it is probably
// still running, and retrying would start a second one.
func IsRetryable(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case CodeUpstreamUnavailable:
		return true
	}
	return apiErr.StatusCode == http.StatusTooManyRequests ||
		apiErr.StatusCode == http.StatusBadGateway ||
		apiErr.StatusCode == http.StatusServiceUnavailable
}

// errorEnvelope is the wire shape of every failure the service reports.
type errorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Resource  string `json:"resource"`
		Details   any    `json:"details"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}
