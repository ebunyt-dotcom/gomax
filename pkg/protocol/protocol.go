package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrIncompletePacket is returned when raw bytes are shorter than total packet length.
	ErrIncompletePacket = errors.New("protocol: raw packet data shorter than header indicated")
)

// Protocol defines the frame encoding and decoding interface.
type Protocol interface {
	Version() uint8
	Encode(frame *OutboundFrame) ([]byte, error)
	Decode(raw []byte) (*InboundFrame, error)
}

// TcpProtocol implements binary frame framing for TCP TLS and binary WebSocket transport.
type TcpProtocol struct {
	version        uint8
	codec          *MsgpackCodec
	payloadDecoder *PayloadDecoder
}

// NewTcpProtocol creates a new TcpProtocol instance.
func NewTcpProtocol() (*TcpProtocol, error) {
	codec := NewMsgpackCodec()
	decoder, err := NewPayloadDecoder(codec)
	if err != nil {
		return nil, fmt.Errorf("init tcp protocol: %w", err)
	}
	return &TcpProtocol{
		version:        VersionTcp,
		codec:          codec,
		payloadDecoder: decoder,
	}, nil
}

// Version returns the protocol version byte (10).
func (p *TcpProtocol) Version() uint8 {
	return p.version
}

// Encode serializes an OutboundFrame into binary wire format (10-byte header + payload).
func (p *TcpProtocol) Encode(frame *OutboundFrame) ([]byte, error) {
	var payloadBytes []byte
	if frame.Payload != nil {
		var err error
		payloadBytes, err = p.codec.Encode(frame.Payload)
		if err != nil {
			return nil, fmt.Errorf("encode payload: %w", err)
		}
	}

	ver := frame.Version
	if ver == 0 {
		ver = p.version
	}

	hdr := &Header{
		Version:    ver,
		Cmd:        frame.Cmd,
		Seq:        frame.Seq,
		Opcode:     frame.Opcode,
		Flags:      frame.Flags,
		PayloadLen: uint32(len(payloadBytes)),
	}

	totalLen := HeaderSize + len(payloadBytes)
	buf := make([]byte, totalLen)
	if err := hdr.EncodeTo(buf[:HeaderSize]); err != nil {
		return nil, err
	}
	copy(buf[HeaderSize:], payloadBytes)
	return buf, nil
}

// Decode deserializes binary wire format into an InboundFrame.
func (p *TcpProtocol) Decode(raw []byte) (*InboundFrame, error) {
	hdr, err := DecodeHeader(raw)
	if err != nil {
		return nil, err
	}

	totalLen := HeaderSize + int(hdr.PayloadLen)
	if len(raw) < totalLen {
		return nil, fmt.Errorf("%w: got %d, expected %d", ErrIncompletePacket, len(raw), totalLen)
	}

	payloadBytes := raw[HeaderSize:totalLen]
	payload, err := p.payloadDecoder.Decode(payloadBytes, hdr.Flags)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	return &InboundFrame{
		Header:  *hdr,
		Opcode:  hdr.Opcode,
		Cmd:     hdr.Cmd,
		Seq:     hdr.Seq,
		Payload: payload,
		Raw:     payload,
	}, nil
}

// WsProtocol implements WebSocket framing, supporting both JSON and binary modes.
type WsProtocol struct {
	version uint8
	binary  bool
	tcp     *TcpProtocol
}

// NewWsProtocol creates a new WsProtocol.
func NewWsProtocol(binary bool) (*WsProtocol, error) {
	tcpProto, err := NewTcpProtocol()
	if err != nil {
		return nil, err
	}
	ver := VersionWs
	if binary {
		ver = VersionTcp
	}
	return &WsProtocol{
		version: ver,
		binary:  binary,
		tcp:     tcpProto,
	}, nil
}

// Version returns the protocol version byte (11 for JSON WS, 10 for binary WS).
func (w *WsProtocol) Version() uint8 {
	return w.version
}

// Encode encodes the outbound frame as binary or JSON string bytes.
func (w *WsProtocol) Encode(frame *OutboundFrame) ([]byte, error) {
	if w.binary {
		return w.tcp.Encode(frame)
	}
	return json.Marshal(frame)
}

// Decode decodes binary or JSON frame into an InboundFrame.
func (w *WsProtocol) Decode(raw []byte) (*InboundFrame, error) {
	if len(raw) == 0 {
		return &InboundFrame{}, nil
	}

	// Auto-detect JSON vs binary: JSON begins with whitespace or '{'
	if raw[0] == '{' || !w.binary {
		var f InboundFrame
		if err := json.Unmarshal(raw, &f); err != nil {
			return &InboundFrame{}, nil
		}
		return &f, nil
	}

	return w.tcp.Decode(raw)
}
