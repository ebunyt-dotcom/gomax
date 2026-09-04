package protocol_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
)

// ============================================================================
// 1. Header Decoder Boundary & Fuzz Testing
// ============================================================================

func TestAdversarial_DecodeHeader_Truncated(t *testing.T) {
	// Test all truncated lengths from 0 to 9 bytes
	for length := 0; length < protocol.HeaderSize; length++ {
		data := make([]byte, length)
		_, err := protocol.DecodeHeader(data)
		if err == nil {
			t.Fatalf("expected error for truncated header of length %d, got nil", length)
		}
		if !errors.Is(err, protocol.ErrHeaderTooShort) {
			t.Fatalf("expected ErrHeaderTooShort for length %d, got %v", length, err)
		}

		// Also verify ExtractPayloadLen returns ErrHeaderTooShort
		_, errLen := protocol.ExtractPayloadLen(data)
		if !errors.Is(errLen, protocol.ErrHeaderTooShort) {
			t.Fatalf("ExtractPayloadLen expected ErrHeaderTooShort for length %d, got %v", length, errLen)
		}
	}
}

func TestAdversarial_DecodeHeader_Nil(t *testing.T) {
	_, err := protocol.DecodeHeader(nil)
	if !errors.Is(err, protocol.ErrHeaderTooShort) {
		t.Fatalf("expected ErrHeaderTooShort for nil slice, got %v", err)
	}

	_, errLen := protocol.ExtractPayloadLen(nil)
	if !errors.Is(errLen, protocol.ErrHeaderTooShort) {
		t.Fatalf("ExtractPayloadLen expected ErrHeaderTooShort for nil slice, got %v", errLen)
	}
}

func TestAdversarial_DecodeHeader_ExactBoundaryValues(t *testing.T) {
	testCases := []struct {
		name       string
		ver        uint8
		cmd        protocol.Command
		seq        uint16
		opcode     protocol.Opcode
		flags      uint8
		payloadLen uint32
	}{
		{"all zeros", 0, 0, 0, 0, 0, 0},
		{"all max", 255, 255, math.MaxUint16, protocol.Opcode(math.MaxUint16), 255, protocol.MaxPayloadLen},
		{"tcp normal", protocol.VersionTcp, protocol.CmdRequest, 1, protocol.OpPing, 0, 100},
		{"ws normal", protocol.VersionWs, protocol.CmdEvent, 65535, protocol.OpNotifMessage, 4, 1024},
		{"max payload limit", protocol.VersionTcp, protocol.CmdResponse, 42, protocol.OpLogin, 0xFF, protocol.MaxPayloadLen},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hdr := &protocol.Header{
				Version:    tc.ver,
				Cmd:        tc.cmd,
				Seq:        tc.seq,
				Opcode:     tc.opcode,
				Flags:      tc.flags,
				PayloadLen: tc.payloadLen,
			}

			encoded := hdr.Encode()
			if len(encoded) != protocol.HeaderSize {
				t.Fatalf("encoded header size mismatch: got %d, expected %d", len(encoded), protocol.HeaderSize)
			}

			decoded, err := protocol.DecodeHeader(encoded)
			if err != nil {
				t.Fatalf("DecodeHeader failed: %v", err)
			}

			if decoded.Version != tc.ver || decoded.Cmd != tc.cmd ||
				decoded.Seq != tc.seq || decoded.Opcode != tc.opcode ||
				decoded.Flags != tc.flags || decoded.PayloadLen != tc.payloadLen {
				t.Fatalf("header mismatch:\nGot:      %+v\nExpected: %+v", decoded, tc)
			}

			extractedLen, err := protocol.ExtractPayloadLen(encoded)
			if err != nil || extractedLen != tc.payloadLen {
				t.Fatalf("ExtractPayloadLen mismatch: got %d, expected %d, err: %v", extractedLen, tc.payloadLen, err)
			}
		})
	}
}

func TestAdversarial_DecodeHeader_OversizedBuffers(t *testing.T) {
	// Header followed by arbitrary extra trailing data (1 byte to 64KB)
	sizes := []int{11, 16, 64, 256, 1024, 65536}
	for _, size := range sizes {
		buf := make([]byte, size)
		_, _ = rand.Read(buf)
		// Set valid header bytes in front
		buf[0] = 10 // Version
		buf[1] = 0  // Cmd

		hdr, err := protocol.DecodeHeader(buf)
		if err != nil {
			t.Fatalf("DecodeHeader failed on oversized buffer of size %d: %v", size, err)
		}
		if hdr.Version != 10 {
			t.Fatalf("expected Version=10, got %d", hdr.Version)
		}
	}
}

func TestAdversarial_DecodeHeader_FuzzRandom(t *testing.T) {
	// Fuzz with 50,000 random binary slices of random lengths (0 to 128 bytes)
	fuzzCount := 50000
	buf := make([]byte, 128)

	for i := 0; i < fuzzCount; i++ {
		length := i % 129
		slice := buf[:length]
		_, _ = rand.Read(slice)

		// Must never panic
		hdr, err := protocol.DecodeHeader(slice)
		if length < protocol.HeaderSize {
			if err == nil {
				t.Fatalf("expected error for length %d, got nil", length)
			}
		} else {
			if err != nil {
				t.Fatalf("unexpected error for length %d: %v", length, err)
			}
			if hdr == nil {
				t.Fatal("expected non-nil Header when err is nil")
			}
			// Verify PayloadLen is within 24-bit range
			if hdr.PayloadLen > protocol.MaxPayloadLen {
				t.Fatalf("PayloadLen exceeds 24-bit limit: %d", hdr.PayloadLen)
			}
		}

		// Also fuzz ExtractPayloadLen
		pLen, pErr := protocol.ExtractPayloadLen(slice)
		if length < protocol.HeaderSize {
			if pErr == nil {
				t.Fatalf("ExtractPayloadLen expected error for length %d", length)
			}
		} else {
			if pErr != nil {
				t.Fatalf("ExtractPayloadLen unexpected error: %v", pErr)
			}
			if pLen > protocol.MaxPayloadLen {
				t.Fatalf("ExtractPayloadLen returned > 24 bits: %d", pLen)
			}
		}
	}
}

func TestAdversarial_Header_Encode_OversizedPayloadLen(t *testing.T) {
	// Header with PayloadLen exceeding 24-bit maximum (16MB)
	hdr := &protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdRequest,
		Seq:        1,
		Opcode:     protocol.OpPing,
		Flags:      0,
		PayloadLen: protocol.MaxPayloadLen + 1, // 16,777,216 bytes
	}

	dst := make([]byte, protocol.HeaderSize)
	err := hdr.EncodeTo(dst)
	if !errors.Is(err, protocol.ErrPayloadLengthExceeded) {
		t.Fatalf("EncodeTo: expected ErrPayloadLengthExceeded, got %v", err)
	}

	// Verify Encode() behavior: EncodeTo returns error, Encode ignores it and returns buffer
	encoded := hdr.Encode()
	if len(encoded) != protocol.HeaderSize {
		t.Fatalf("Encode: expected 10 bytes, got %d", len(encoded))
	}
}

// ============================================================================
// 2. MsgPack Codec & Ext 1 Fuzz Testing
// ============================================================================

func TestAdversarial_MsgpackCodec_FuzzMalformedData(t *testing.T) {
	codec := protocol.NewMsgpackCodec()
	fuzzCount := 20000

	for i := 0; i < fuzzCount; i++ {
		length := (i % 256) + 1
		randomBytes := make([]byte, length)
		_, _ = rand.Read(randomBytes)

		// Must never panic on arbitrary corrupted bytes
		_, _ = codec.Decode(randomBytes)
	}
}

func TestAdversarial_MsgpackCodec_DeeplyNestedExt1(t *testing.T) {
	codec := protocol.NewMsgpackCodec()

	// Construct nested Ext 1 payloads to test recursive unwrapping
	innerVal := map[string]any{"deep_key": "deep_val"}
	innerBytes, err := codec.Encode(innerVal)
	if err != nil {
		t.Fatalf("Encode inner failed: %v", err)
	}

	currentBytes := innerBytes
	depth := 20
	for d := 0; d < depth; d++ {
		ext := &protocol.Ext{
			Code: protocol.WrappedValueExtCode,
			Data: currentBytes,
		}
		container := map[string]any{"data": ext}
		currentBytes, err = codec.Encode(container)
		if err != nil {
			t.Fatalf("Encode at depth %d failed: %v", d, err)
		}
	}

	// Decode deeply nested Ext 1 structures
	decoded, err := codec.Decode(currentBytes)
	if err != nil {
		t.Fatalf("Decode deeply nested failed: %v", err)
	}

	if decoded == nil {
		t.Fatal("expected non-nil decoded structure")
	}
}

func TestAdversarial_MsgpackCodec_NormalizeKey_AllTypes(t *testing.T) {
	codec := protocol.NewMsgpackCodec()

	testKeys := []struct {
		input    any
		expected string
	}{
		{"string_key", "string_key"},
		{int(123), "123"},
		{int(-456), "-456"},
		{int64(math.MaxInt64), fmt.Sprintf("%d", int64(math.MaxInt64))},
		{int64(math.MinInt64), fmt.Sprintf("%d", int64(math.MinInt64))},
		{uint64(math.MaxUint64), fmt.Sprintf("%d", uint64(math.MaxUint64))},
		{int32(999), "999"},
		{int16(888), "888"},
		{int8(77), "77"},
		{uint32(111), "111"},
		{uint16(222), "222"},
		{uint8(33), "33"},
		{[]byte("utf8_key"), "utf8_key"},
		{[]byte{0xFF, 0xFE, 0xFD}, "fffefd"}, // non-utf8 hex encoded
		{nil, "<nil>"},
	}

	for _, tc := range testKeys {
		got := codec.NormalizeKey(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeKey(%T %v) = %q, expected %q", tc.input, tc.input, got, tc.expected)
		}
	}
}

func TestAdversarial_MsgpackCodec_HugeMapLenSafety(t *testing.T) {
	codec := protocol.NewMsgpackCodec()

	// 0xdf is map 32 in msgpack format, followed by 4-byte big-endian uint32 length.
	// Malicious payload claiming length = 0x7FFFFFFF (2 billion entries)
	// followed by immediate EOF.
	malicious := []byte{0xdf, 0x7f, 0xff, 0xff, 0xff}

	// Test if it panics with OOM / makeslice or returns decode error cleanly
	_, err := codec.Decode(malicious)
	if err == nil {
		t.Fatal("expected error for truncated huge map, got nil")
	}
}

func TestAdversarial_MsgpackCodec_HugeExtLenSafety(t *testing.T) {
	codec := protocol.NewMsgpackCodec()

	// 0xc9 is ext 32 in msgpack format, followed by 4-byte length and 1-byte type.
	// Malicious payload claiming length = 0x7FFFFFFF (2 billion bytes)
	// followed by code 1.
	malicious := []byte{0xc9, 0x7f, 0xff, 0xff, 0xff, 0x01}

	// Test if it panics with OOM / makeslice or returns decode error cleanly
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("VULNERABILITY DETECTED: MsgPack ext32 decoder panicked with: %v", r)
		}
	}()

	_, err := codec.Decode(malicious)
	if err == nil {
		t.Fatal("expected error for huge ext32, got nil")
	}
}

// ============================================================================
// 3. LZ4 Decompressor Fuzzing & Boundary Testing
// ============================================================================

func TestAdversarial_LZ4_FuzzRandomData(t *testing.T) {
	lz4Dec := protocol.NewLZ4BlockDecompressor()
	fuzzCount := 20000

	for i := 0; i < fuzzCount; i++ {
		length := (i % 256) + 1
		randomBytes := make([]byte, length)
		_, _ = rand.Read(randomBytes)

		// Must never panic on arbitrary random data
		_, _ = lz4Dec.Decompress(randomBytes, protocol.MaxDecompressedSize)
	}
}

func TestAdversarial_LZ4_DecompressionBombSafety(t *testing.T) {
	lz4Dec := protocol.NewLZ4BlockDecompressor()

	// Construct LZ4 input that produces expanding repeated patterns:
	// 1 literal byte 'A' (token 0x10)
	// Match: repeat 'A' with large matchLen
	// Token: 0x1F (literal len 1, match len 15+...)
	// Literal: 'A' (0x41)
	// Offset: 0x0001 (offset 1)
	// Then a series of 0xFF bytes to expand match length beyond 5MB
	var bomb bytes.Buffer
	bomb.WriteByte(0x1F) // litLen = 1, matchLen = 19 + extra
	bomb.WriteByte('A')  // 1 literal byte
	binary.Write(&bomb, binary.LittleEndian, uint16(1)) // offset 1

	// Add 25,000 0xFF bytes -> match length = 25,000 * 255 = ~6.3 MB (> 5 MB)
	bomb.Write(bytes.Repeat([]byte{0xFF}, 25000))
	bomb.WriteByte(0x00) // terminate match length

	// Must reject with ErrOutputTooLarge without crashing/panicking
	_, err := lz4Dec.Decompress(bomb.Bytes(), protocol.MaxDecompressedSize)
	if err == nil {
		t.Fatal("expected ErrOutputTooLarge for decompression bomb, got nil")
	}
	if !errors.Is(err, protocol.ErrOutputTooLarge) {
		t.Fatalf("expected ErrOutputTooLarge, got %v", err)
	}
}

func TestAdversarial_LZ4_SpecificErrors(t *testing.T) {
	lz4Dec := protocol.NewLZ4BlockDecompressor()

	t.Run("ZeroOffset", func(t *testing.T) {
		// Literal 1 'A', offset 0
		input := []byte{0x10, 'A', 0x00, 0x00}
		_, err := lz4Dec.Decompress(input, 1024)
		if !errors.Is(err, protocol.ErrLZ4ZeroOffset) {
			t.Fatalf("expected ErrLZ4ZeroOffset, got %v", err)
		}
	})

	t.Run("MatchOutOfBounds", func(t *testing.T) {
		// Literal 1 'A', offset 5 (only 1 byte in dst)
		input := []byte{0x10, 'A', 0x05, 0x00}
		_, err := lz4Dec.Decompress(input, 1024)
		if !errors.Is(err, protocol.ErrLZ4MatchOutOfBounds) {
			t.Fatalf("expected ErrLZ4MatchOutOfBounds, got %v", err)
		}
	})

	t.Run("LiteralOutOfBounds", func(t *testing.T) {
		// Token claims 5 literals, but only 2 supplied
		input := []byte{0x50, 'A', 'B'}
		_, err := lz4Dec.Decompress(input, 1024)
		if !errors.Is(err, protocol.ErrLZ4LiteralOutOfBounds) {
			t.Fatalf("expected ErrLZ4LiteralOutOfBounds, got %v", err)
		}
	})

	t.Run("IncompleteOffset", func(t *testing.T) {
		// Token claims 1 literal, 1 byte supplied, followed by only 1 byte for offset
		input := []byte{0x10, 'A', 0x01}
		_, err := lz4Dec.Decompress(input, 1024)
		if !errors.Is(err, protocol.ErrLZ4IncompleteOffset) {
			t.Fatalf("expected ErrLZ4IncompleteOffset, got %v", err)
		}
	})
}

// ============================================================================
// 4. Zstd Decompressor Fuzzing & Safety Limit
// ============================================================================

func TestAdversarial_Zstd_FuzzRandomData(t *testing.T) {
	zstdDec, err := protocol.NewZstdDecompressor()
	if err != nil {
		t.Fatalf("NewZstdDecompressor failed: %v", err)
	}

	fuzzCount := 5000
	for i := 0; i < fuzzCount; i++ {
		length := (i % 128) + 1
		randomBytes := make([]byte, length)
		_, _ = rand.Read(randomBytes)

		// Must return error, never panic
		_, _ = zstdDec.Decompress(randomBytes, protocol.MaxDecompressedSize)
	}
}

func TestAdversarial_PayloadDecoder_InvalidCompressionFactor(t *testing.T) {
	codec := protocol.NewMsgpackCodec()
	decoder, err := protocol.NewPayloadDecoder(codec)
	if err != nil {
		t.Fatalf("NewPayloadDecoder failed: %v", err)
	}

	// Any flags in range [0x80, 0xFE] must return ErrInvalidCompressionFactor
	for flag := uint8(0x80); flag < 0xFF; flag++ {
		_, err := decoder.Decode([]byte{0x01}, flag)
		if !errors.Is(err, protocol.ErrInvalidCompressionFactor) {
			t.Fatalf("flag 0x%02X expected ErrInvalidCompressionFactor, got %v", flag, err)
		}
	}
}

// ============================================================================
// 5. TcpProtocol Incomplete & Corrupted Wire Frames
// ============================================================================

func TestAdversarial_TcpProtocol_IncompleteWireFrames(t *testing.T) {
	tcpProto, err := protocol.NewTcpProtocol()
	if err != nil {
		t.Fatalf("NewTcpProtocol failed: %v", err)
	}

	// Create a valid header claiming 100 bytes payload
	hdr := &protocol.Header{
		Version:    protocol.VersionTcp,
		Cmd:        protocol.CmdRequest,
		Seq:        1,
		Opcode:     protocol.OpPing,
		PayloadLen: 100,
	}
	headerBytes := hdr.Encode()

	// Test with 0 to 99 payload bytes (all incomplete)
	for payloadSize := 0; payloadSize < 100; payloadSize++ {
		partialFrame := append(headerBytes, make([]byte, payloadSize)...)
		_, err := tcpProto.Decode(partialFrame)
		if !errors.Is(err, protocol.ErrIncompletePacket) {
			t.Fatalf("expected ErrIncompletePacket for frame with %d/100 payload bytes, got %v", payloadSize, err)
		}
	}
}
