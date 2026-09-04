package protocol

import "fmt"

// InboundFrame represents an incoming decoded packet from the Max server.
type InboundFrame struct {
	Header  Header         `json:"header,omitempty"`
	Opcode  Opcode         `json:"opcode"`
	Cmd     Command        `json:"cmd"`
	Seq     uint16         `json:"seq"`
	Payload map[string]any `json:"payload,omitempty"`
	Raw     any            `json:"raw,omitempty"`
}

// IsResponse returns true if this frame resolves an outbound request.
func (f *InboundFrame) IsResponse() bool {
	return f.Cmd == CmdResponse
}

// IsEvent returns true if this frame is a push event.
func (f *InboundFrame) IsEvent() bool {
	return f.Cmd == CmdEvent || f.Opcode.IsNotification()
}

// IsError returns true if this frame indicates a server error.
func (f *InboundFrame) IsError() bool {
	return f.Cmd == CmdError
}

// ErrorString extracts any error description present in an error frame payload.
func (f *InboundFrame) ErrorString() string {
	if !f.IsError() && f.Payload == nil {
		return ""
	}
	if f.Payload != nil {
		if errVal, ok := f.Payload["error"]; ok {
			msgVal := f.Payload["message"]
			return fmt.Sprintf("MaxApiError(error=%v, message=%v)", errVal, msgVal)
		}
	}
	return fmt.Sprintf("MaxApiError(opcode=%s, cmd=%s)", f.Opcode, f.Cmd)
}

// OutboundFrame represents an outgoing packet to be serialized and transmitted to Max.
type OutboundFrame struct {
	Version uint8   `json:"ver"`
	Opcode  Opcode  `json:"opcode"`
	Cmd     Command `json:"cmd"`
	Seq     uint16  `json:"seq"`
	Flags   uint8   `json:"flags,omitempty"`
	Payload any     `json:"payload,omitempty"`
}

// NewRequest creates a standard outbound request frame with CmdRequest (0) and VersionTcp (10).
func NewRequest(opcode Opcode, seq uint16, payload any) *OutboundFrame {
	return &OutboundFrame{
		Version: VersionTcp,
		Opcode:  opcode,
		Cmd:     CmdRequest,
		Seq:     seq,
		Payload: payload,
	}
}

// NewEvent creates an outbound event frame with CmdEvent (2) and VersionTcp (10).
func NewEvent(opcode Opcode, seq uint16, payload any) *OutboundFrame {
	return &OutboundFrame{
		Version: VersionTcp,
		Opcode:  opcode,
		Cmd:     CmdEvent,
		Seq:     seq,
		Payload: payload,
	}
}

// PackedPacket represents a raw packet consisting of an unpacked Header and unparsed payload bytes.
type PackedPacket struct {
	Header       Header `json:"header"`
	PayloadBytes []byte `json:"payload_bytes"`
}
