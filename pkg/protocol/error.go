package protocol

import (
	"fmt"
	"strings"
)

// ApiError represents an application-level API error returned by the Max server.
type ApiError struct {
	Opcode           Opcode         `json:"opcode"`
	Cmd              Command        `json:"cmd"`
	ErrorStr         string         `json:"error,omitempty"`
	Message          string         `json:"message,omitempty"`
	LocalizedMessage string         `json:"localized_message,omitempty"`
	Title            string         `json:"title,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
}

// NewApiError constructs an ApiError from an InboundFrame.
func NewApiError(frame *InboundFrame) *ApiError {
	apiErr := &ApiError{
		Opcode:  frame.Opcode,
		Cmd:     frame.Cmd,
		Payload: frame.Payload,
	}

	if frame.Payload != nil {
		if val, ok := frame.Payload["error"].(string); ok {
			apiErr.ErrorStr = val
		}
		if val, ok := frame.Payload["message"].(string); ok {
			apiErr.Message = val
		}
		if val, ok := frame.Payload["localized_message"].(string); ok {
			apiErr.LocalizedMessage = val
		} else if val, ok := frame.Payload["localizedMessage"].(string); ok {
			apiErr.LocalizedMessage = val
		}
		if val, ok := frame.Payload["title"].(string); ok {
			apiErr.Title = val
		}
	}

	return apiErr
}

// Error formats the error string in parity with PyMax ApiError.
func (e *ApiError) Error() string {
	var parts []string
	if e.LocalizedMessage != "" {
		parts = append(parts, e.LocalizedMessage)
	}
	if e.Message != "" && e.Message != e.LocalizedMessage {
		parts = append(parts, e.Message)
	}
	if e.Title != "" {
		parts = append(parts, fmt.Sprintf("(%s)", e.Title))
	}
	if e.ErrorStr != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.ErrorStr))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("API request failed (opcode=%s, cmd=%s)", e.Opcode, e.Cmd)
	}

	return strings.Join(parts, " ")
}
