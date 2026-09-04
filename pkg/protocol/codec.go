package protocol

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	// WrappedValueExtCode is the extension type ID used by Max server for wrapped serialized structures.
	WrappedValueExtCode int8 = 1
	// MaxMapEntries prevents a forged MessagePack map header from forcing a
	// huge allocation before the decoder has seen enough bytes to support it.
	MaxMapEntries = 1 << 20
	// MaxExtensionSize bounds an extension allocation independently of the
	// decompression limit; extensions are small wrappers in the Max protocol.
	MaxExtensionSize = 5 * 1024 * 1024
)

// Ext represents a MessagePack extension type with a type code and raw payload bytes.
type Ext struct {
	Code int8   `json:"code"`
	Data []byte `json:"data"`
}

// EncodeMsgpack implements msgpack.CustomEncoder for Ext, encoding arbitrary extension type codes.
func (e *Ext) EncodeMsgpack(enc *msgpack.Encoder) error {
	if e == nil {
		return enc.EncodeNil()
	}
	if err := enc.EncodeExtHeader(e.Code, len(e.Data)); err != nil {
		return err
	}
	_, err := enc.Writer().Write(e.Data)
	return err
}

func init() {
	// Register extension decoder for all possible 256 int8 extension codes.
	// This ensures msgpack decodes any extension into *Ext without throwing unknown ext errors.
	for id := -128; id <= 127; id++ {
		code := int8(id)
		msgpack.RegisterExtDecoder(code, (*Ext)(nil), func(dec *msgpack.Decoder, v reflect.Value, extLen int) error {
			if extLen < 0 || extLen > MaxExtensionSize {
				return fmt.Errorf("protocol: extension payload too large: %d bytes", extLen)
			}
			b := make([]byte, extLen)
			if err := dec.ReadFull(b); err != nil {
				return err
			}
			v.Set(reflect.ValueOf(&Ext{Code: code, Data: b}))
			return nil
		})
	}
}

// MsgpackCodec provides encoding and decoding for MessagePack payloads.
type MsgpackCodec struct{}

// NewMsgpackCodec creates a new MsgpackCodec instance.
func NewMsgpackCodec() *MsgpackCodec {
	return &MsgpackCodec{}
}

// Encode serializes a Go value or struct into MessagePack bytes.
// If payload is nil, it returns an empty byte slice.
func (c *MsgpackCodec) Encode(payload any) ([]byte, error) {
	if payload == nil {
		return []byte{}, nil
	}
	return msgpack.Marshal(payload)
}

// Decode deserializes MessagePack bytes into a normalized Go map/value.
// If payloadBytes is empty, it returns an empty map.
// If extra bytes follow the first MessagePack object in the stream, the first valid object is returned.
func (c *MsgpackCodec) Decode(payloadBytes []byte) (any, error) {
	if len(payloadBytes) == 0 {
		return make(map[string]any), nil
	}

	dec := msgpack.NewDecoder(bytes.NewReader(payloadBytes))

	// Custom map decoder normalizes all map keys (including integers and byte slices) to string
	dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
		n, err := d.DecodeMapLen()
		if err != nil {
			return nil, err
		}
		if n == -1 {
			return nil, nil
		}
		// Every map entry consumes at least a key and a value byte. Comparing
		// with the remaining input rejects impossible forged lengths while
		// still permitting the full protocol payload size.
		if n > MaxMapEntries || n > len(payloadBytes) {
			return nil, fmt.Errorf("protocol: map contains too many entries: %d", n)
		}
		m := make(map[string]any, n)
		for i := 0; i < n; i++ {
			keyVal, err := d.DecodeInterface()
			if err != nil {
				return nil, err
			}
			val, err := d.DecodeInterface()
			if err != nil {
				return nil, err
			}
			strKey := c.NormalizeKey(keyVal)
			m[strKey] = c.Normalize(val)
		}
		return m, nil
	})

	var result any
	if err := dec.Decode(&result); err != nil {
		return nil, fmt.Errorf("protocol: msgpack decode failed: %w", err)
	}

	return c.Normalize(result), nil
}

// Normalize recursively traverses decoded objects, converting:
// 1. Integer and byte keys in maps to string keys.
// 2. Wrapped extension types (Code 1) to recursively decoded values.
// 3. Slices/arrays to recursively normalized slices.
func (c *MsgpackCodec) Normalize(val any) any {
	switch v := val.(type) {
	case map[string]any:
		res := make(map[string]any, len(v))
		for k, item := range v {
			res[k] = c.Normalize(item)
		}
		return res

	case map[any]any:
		res := make(map[string]any, len(v))
		for k, item := range v {
			strKey := c.NormalizeKey(k)
			res[strKey] = c.Normalize(item)
		}
		return res

	case []any:
		res := make([]any, len(v))
		for i, item := range v {
			res[i] = c.Normalize(item)
		}
		return res

	case *Ext:
		if v.Code == WrappedValueExtCode {
			unwrapped, err := c.Decode(v.Data)
			if err == nil {
				return unwrapped
			}
		}
		return v

	case Ext:
		if v.Code == WrappedValueExtCode {
			unwrapped, err := c.Decode(v.Data)
			if err == nil {
				return unwrapped
			}
		}
		return v

	default:
		return v
	}
}

// NormalizeKey converts an arbitrary map key (int, uint, []byte, string) to a string representation.
func (c *MsgpackCodec) NormalizeKey(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case int32:
		return strconv.FormatInt(int64(k), 10)
	case int16:
		return strconv.FormatInt(int64(k), 10)
	case int8:
		return strconv.FormatInt(int64(k), 10)
	case uint:
		return strconv.FormatUint(uint64(k), 10)
	case uint64:
		return strconv.FormatUint(k, 10)
	case uint32:
		return strconv.FormatUint(uint64(k), 10)
	case uint16:
		return strconv.FormatUint(uint64(k), 10)
	case uint8:
		return strconv.FormatUint(uint64(k), 10)
	case []byte:
		if utf8.Valid(k) {
			return string(k)
		}
		return hex.EncodeToString(k)
	default:
		return fmt.Sprintf("%v", k)
	}
}
