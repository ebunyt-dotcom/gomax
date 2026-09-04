package protocol_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/vmihailenco/msgpack/v5"

	"gomax/pkg/protocol"
)

func TestTcpFramerUsesExpectedHeaderLayout(t *testing.T) {
	header := &protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdResponse,
		Seq:        0x0100, // 256
		Opcode:     protocol.OpPing,
		Flags:      2,
		PayloadLen: 3, // len("abc")
	}

	packed := header.Encode()
	expected := []byte{0x0A, 0x01, 0x01, 0x00, 0x00, 0x01, 0x02, 0x00, 0x00, 0x03}

	if !bytes.Equal(packed, expected) {
		t.Fatalf("Header encode mismatch.\nGot:      %#v\nExpected: %#v", packed, expected)
	}

	decoded, err := protocol.DecodeHeader(packed)
	if err != nil {
		t.Fatalf("Failed to decode header: %v", err)
	}

	if decoded.Version != 10 || decoded.Cmd != protocol.CmdResponse ||
		decoded.Seq != 0x0100 || decoded.Opcode != protocol.OpPing ||
		decoded.Flags != 2 || decoded.PayloadLen != 3 {
		t.Fatalf("Decoded header values mismatch: %+v", decoded)
	}
}

func TestTcpFramerHandlesShortAndIncompletePackets(t *testing.T) {
	// Short header (< 10 bytes)
	_, err := protocol.DecodeHeader([]byte("short"))
	if err == nil {
		t.Fatal("Expected error for short header, got nil")
	}

	// Extract payload length on valid vs short
	validHeader := (&protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdRequest,
		Seq:        1,
		Opcode:     protocol.OpPing,
		Flags:      2,
		PayloadLen: 3,
	}).Encode()

	payloadLen, err := protocol.ExtractPayloadLen(validHeader)
	if err != nil || payloadLen != 3 {
		t.Fatalf("Expected payloadLen=3, got %d, err=%v", payloadLen, err)
	}

	_, err = protocol.ExtractPayloadLen([]byte("short"))
	if err == nil {
		t.Fatal("Expected error extracting payload length from short data")
	}
}

func TestTcpProtocolSupportsTwoByteSequenceIDs(t *testing.T) {
	header := &protocol.Header{
		Version:    10,
		Cmd:        protocol.CmdRequest,
		Seq:        0xFFFF, // 65535 max uint16
		Opcode:     protocol.OpPing,
		Flags:      0,
		PayloadLen: 0,
	}

	packed := header.Encode()
	decoded, err := protocol.DecodeHeader(packed)
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if decoded.Seq != 0xFFFF {
		t.Fatalf("Expected Seq=0xFFFF (65535), got 0x%04X", decoded.Seq)
	}
}

func TestMsgpackCodecSerializesEnumsAndDecoderNormalizesKeys(t *testing.T) {
	codec := protocol.NewMsgpackCodec()
	decoder, err := protocol.NewPayloadDecoder(codec)
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}

	// Payload with integer key 1 and binary byte slice key []byte("name")
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	_ = enc.EncodeMapLen(2)
	_ = enc.EncodeInt(1) // integer key 1
	_ = enc.EncodeMapLen(1)
	_ = enc.EncodeBytes([]byte("name")) // byte slice key
	_ = enc.EncodeInt(42)
	_ = enc.EncodeString("list")
	_ = enc.EncodeArrayLen(2)
	_ = enc.EncodeInt(10)
	_ = enc.EncodeInt(20)

	encoded := buf.Bytes()

	decoded, err := decoder.Decode(encoded, protocol.CompressionNone)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify key "1" is a string
	sub, ok := decoded["1"].(map[string]any)
	if !ok {
		t.Fatalf("Expected decoded['1'] to be map[string]any, got %T: %+v", decoded["1"], decoded)
	}

	// Verify key "name" is a string
	if val, ok := sub["name"]; !ok || fmt.Sprintf("%v", val) != "42" {
		t.Fatalf("Expected sub['name'] == 42, got %v", val)
	}
}

func TestMsgpackCodecUsesFirstDictWhenStreamHasExtraData(t *testing.T) {
	codec := protocol.NewMsgpackCodec()

	// Pack {"ok": true} followed by ["ignored"]
	first, _ := msgpack.Marshal(map[string]any{"ok": true})
	second, _ := msgpack.Marshal([]any{"ignored"})
	stream := append(first, second...)

	decoded, err := codec.Decode(stream)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	asMap, ok := decoded.(map[string]any)
	if !ok || asMap["ok"] != true {
		t.Fatalf("Expected {'ok': true}, got: %+v", decoded)
	}
}

func TestMsgpackCodecDecodesWrappedValueExtension(t *testing.T) {
	codec := protocol.NewMsgpackCodec()

	expectedSub := map[string]any{
		"expiresAt":       int64(1783264954296),
		"pollingInterval": int64(5000),
		"ttl":             int64(119992),
	}

	subBytes, err := msgpack.Marshal(expectedSub)
	if err != nil {
		t.Fatalf("Marshal subBytes failed: %v", err)
	}

	// Wrapped inside ExtType 1
	wrapped := map[string]any{
		"payload": &protocol.Ext{
			Code: protocol.WrappedValueExtCode,
			Data: subBytes,
		},
	}

	encoded, err := msgpack.Marshal(wrapped)
	if err != nil {
		t.Fatalf("Marshal wrapped failed: %v", err)
	}

	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	asMap, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", decoded)
	}

	payloadMap, ok := asMap["payload"].(map[string]any)
	if !ok {
		t.Fatalf("Expected payload to be unpacked map[string]any, got %T: %+v", asMap["payload"], asMap)
	}

	if fmt.Sprintf("%v", payloadMap["expiresAt"]) != "1783264954296" {
		t.Fatalf("expiresAt mismatch: %v", payloadMap["expiresAt"])
	}
}

func TestTcpPayloadDecoderPreservesUnknownExtensions(t *testing.T) {
	codec := protocol.NewMsgpackCodec()
	extension := &protocol.Ext{Code: 42, Data: []byte("unknown")}
	encoded, err := codec.Encode(map[string]any{"extension": extension})
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	asMap, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", decoded)
	}

	ext, ok := asMap["extension"].(*protocol.Ext)
	if !ok {
		t.Fatalf("Expected *protocol.Ext, got %T: %+v", asMap["extension"], asMap)
	}

	if ext.Code != 42 || string(ext.Data) != "unknown" {
		t.Fatalf("Extension mismatch: %+v", ext)
	}
}

func TestTcpPayloadDecoderDecompressesLz4ForCompressionFactorFour(t *testing.T) {
	// Raw hex string vector from PyMax test_tcp_payload_decoder_decompresses_lz4_for_compression_factor_four
	hexData := "f40a84a6707265666978a27878a464617461b0664a73436c4b437508008fa47461696cd92a79010016dfa6726570656174d9684142434404004c504441424344"
	compressed, err := hex.DecodeString(hexData)
	if err != nil {
		t.Fatalf("hex.DecodeString failed: %v", err)
	}

	codec := protocol.NewMsgpackCodec()
	decoder, err := protocol.NewPayloadDecoder(codec)
	if err != nil {
		t.Fatalf("NewPayloadDecoder failed: %v", err)
	}

	decoded, err := decoder.Decode(compressed, 4) // flags = 4
	if err != nil {
		t.Fatalf("Decode LZ4 payload failed: %v", err)
	}

	// Verify prefix == "xx"
	if decoded["prefix"] != "xx" {
		t.Fatalf("prefix mismatch: got %v", decoded["prefix"])
	}

	// Verify data == "fJsClKCufJsClKCu"
	if decoded["data"] != "fJsClKCufJsClKCu" {
		t.Fatalf("data mismatch: got %v", decoded["data"])
	}

	// Verify tail == "y" * 42
	expectedTail := strings.Repeat("y", 42)
	if decoded["tail"] != expectedTail {
		t.Fatalf("tail length mismatch: got len=%d, expected=42", len(fmt.Sprintf("%v", decoded["tail"])))
	}

	// Verify repeat == "ABCD" * 26
	expectedRepeat := strings.Repeat("ABCD", 26)
	if decoded["repeat"] != expectedRepeat {
		t.Fatalf("repeat mismatch: got len=%d, expected=104", len(fmt.Sprintf("%v", decoded["repeat"])))
	}
}

func TestTcpPayloadDecoderDecompressesZstd(t *testing.T) {
	expected := map[string]any{
		"error":   "FAIL_LOGIN_TOKEN",
		"message": "Token expired",
	}

	msgBytes, err := msgpack.Marshal(expected)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var compressedBuf bytes.Buffer
	enc, err := zstd.NewWriter(&compressedBuf)
	if err != nil {
		t.Fatalf("zstd.NewWriter failed: %v", err)
	}
	_, _ = enc.Write(msgBytes)
	_ = enc.Close()

	codec := protocol.NewMsgpackCodec()
	decoder, err := protocol.NewPayloadDecoder(codec)
	if err != nil {
		t.Fatalf("NewPayloadDecoder failed: %v", err)
	}

	decoded, err := decoder.Decode(compressedBuf.Bytes(), protocol.CompressionZstd)
	if err != nil {
		t.Fatalf("Decode Zstd failed: %v", err)
	}

	if decoded["error"] != "FAIL_LOGIN_TOKEN" || decoded["message"] != "Token expired" {
		t.Fatalf("Zstd decoded content mismatch: %+v", decoded)
	}
}

func TestZstdDecompressionRejectsOversizedOutput(t *testing.T) {
	zstdDec, err := protocol.NewZstdDecompressor()
	if err != nil {
		t.Fatalf("NewZstdDecompressor failed: %v", err)
	}

	// Compress 128 bytes
	raw := bytes.Repeat([]byte("x"), 128)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	_, _ = enc.Write(raw)
	_ = enc.Close()

	// Try decompressing with maxOutput = 64
	_, err = zstdDec.Decompress(buf.Bytes(), 64)
	if err == nil {
		t.Fatal("Expected ErrOutputTooLarge, got nil")
	}
}

func TestLz4DecompressesLiteralsAndRejectsInvalidBlocks(t *testing.T) {
	lz4Dec := protocol.NewLZ4BlockDecompressor()

	// Literal test: token = 0x50 (5 literals), data = "hello"
	input := append([]byte{0x50}, []byte("hello")...)
	out, err := lz4Dec.Decompress(input, 1024)
	if err != nil || string(out) != "hello" {
		t.Fatalf("Literal decompress failed: out=%q, err=%v", string(out), err)
	}

	// Zero offset test: token 0x01, offset 0x0000
	zeroOffset := []byte{0x01, 0x00, 0x00}
	_, err = lz4Dec.Decompress(zeroOffset, 1024)
	if err == nil || !strings.Contains(err.Error(), "zero offset") {
		t.Fatalf("Expected zero offset error, got: %v", err)
	}
}

func TestInvalidCompressionFactor(t *testing.T) {
	codec := protocol.NewMsgpackCodec()
	decoder, err := protocol.NewPayloadDecoder(codec)
	if err != nil {
		t.Fatalf("NewPayloadDecoder failed: %v", err)
	}

	_, err = decoder.Decode([]byte{0x01, 0x02}, 0x80)
	if err == nil {
		t.Fatal("Expected error for compression factor 0x80, got nil")
	}

	_, err = decoder.Decode([]byte{0x01, 0x02}, 0xFE)
	if err == nil {
		t.Fatal("Expected error for compression factor 0xFE, got nil")
	}
}

func TestAll114OpcodesCoverageAndStringer(t *testing.T) {
	if protocol.CountOpcodes() != 177 {
		t.Fatalf("Expected 177 opcodes, got %d", protocol.CountOpcodes())
	}

	// Verify key opcodes have non-fallback String() values
	testCases := []struct {
		op   protocol.Opcode
		name string
	}{
		{protocol.OpPing, "PING"},
		{protocol.OpSessionInit, "SESSION_INIT"},
		{protocol.OpLogin, "LOGIN"},
		{protocol.OpMsgSend, "MSG_SEND"},
		{protocol.OpNotifMessage, "NOTIF_MESSAGE"},
		{protocol.OpGetQr, "GET_QR"},
		{protocol.OpGetPollUpdates, "GET_POLL_UPDATES"},
	}

	for _, tc := range testCases {
		if tc.op.String() != tc.name {
			t.Fatalf("Opcode %d name mismatch: expected %q, got %q", tc.op, tc.name, tc.op.String())
		}
	}

	// Verify notification classification
	if !protocol.OpNotifMessage.IsNotification() {
		t.Fatal("OpNotifMessage must be classified as a notification")
	}
	if protocol.OpPing.IsNotification() {
		t.Fatal("OpPing must NOT be classified as a notification")
	}
}

func TestTcpProtocolRoundtrip(t *testing.T) {
	tcpProto, err := protocol.NewTcpProtocol()
	if err != nil {
		t.Fatalf("NewTcpProtocol failed: %v", err)
	}

	outbound := protocol.NewRequest(protocol.OpChatHistory, 3, map[string]any{
		"chatId": 100,
	})

	encoded, err := tcpProto.Encode(outbound)
	if err != nil {
		t.Fatalf("tcpProto.Encode failed: %v", err)
	}

	decoded, err := tcpProto.Decode(encoded)
	if err != nil {
		t.Fatalf("tcpProto.Decode failed: %v", err)
	}

	if decoded.Opcode != protocol.OpChatHistory {
		t.Fatalf("expected OpChatHistory, got %s", decoded.Opcode)
	}
	if decoded.Seq != 3 {
		t.Fatalf("expected Seq=3, got %d", decoded.Seq)
	}
	if decoded.Cmd != protocol.CmdRequest {
		t.Fatalf("expected CmdRequest, got %s", decoded.Cmd)
	}
	if fmt.Sprintf("%v", decoded.Payload["chatId"]) != "100" {
		t.Fatalf("expected chatId 100, got %v", decoded.Payload["chatId"])
	}
}
