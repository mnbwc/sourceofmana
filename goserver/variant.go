package main

// Godot Variant binary encoding (core/io/marshalls.cpp) — minimal subset
// sufficient for SoM's RPC surface: NIL, BOOL, INT, FLOAT, STRING, ARRAY,
// DICTIONARY, VECTOR2, PACKED_BYTE_ARRAY.
//
// Wire format: every variant starts with a little-endian uint32 header.
//   header bits 0..7   = Variant type (matches Variant::Type enum)
//   header bit 16      = 64-bit payload flag (INT/FLOAT)
//   header bits 16..19 = container typed-array/dictionary flags
//
// This file implements encode_variant (no compression) and decode_variant.
// The compressed path (encode_and_compress_variant in multiplayer_api.cpp) is
// handled in rpc.go because it is only used for BOOL and small INTs.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// maxVariantElements bounds array/dict/packed-array element counts to prevent
// memory exhaustion from a crafted packet declaring billions of elements.
const maxVariantElements = 100000

// Variant type constants — must match Variant::Type in core/variant/variant.h.
// Verified against Godot 4.7.1 var_to_bytes() output.
const (
	VariantNil                uint32 = 0
	VariantBool               uint32 = 1
	VariantInt                uint32 = 2
	VariantFloat              uint32 = 3
	VariantString             uint32 = 4
	VariantVector2            uint32 = 5
	VariantVector2i           uint32 = 6
	VariantRect2              uint32 = 7
	VariantRect2i             uint32 = 8
	VariantVector3            uint32 = 9
	VariantVector3i           uint32 = 10
	VariantTransform2D        uint32 = 11
	VariantVector4            uint32 = 12
	VariantVector4i           uint32 = 13
	VariantPlane              uint32 = 14
	VariantQuaternion         uint32 = 15
	VariantAABB               uint32 = 16
	VariantBasis              uint32 = 17
	VariantTransform3D        uint32 = 18
	VariantProjection         uint32 = 19
	VariantColor              uint32 = 20
	VariantStringName         uint32 = 21
	VariantNodePath           uint32 = 22
	VariantRID                uint32 = 23
	VariantObject             uint32 = 24
	VariantCallable           uint32 = 25
	VariantSignal             uint32 = 26
	VariantDictionary         uint32 = 27
	VariantArray              uint32 = 28
	VariantPackedByteArray    uint32 = 29
	VariantPackedInt32Array   uint32 = 30
	VariantPackedInt64Array   uint32 = 31
	VariantPackedFloat32Array uint32 = 32
	VariantPackedFloat64Array uint32 = 33
	VariantPackedStringArray  uint32 = 34
	VariantPackedVector2Array uint32 = 35
	VariantPackedVector3Array uint32 = 36
	VariantPackedColorArray   uint32 = 37
	VariantPackedVector4Array uint32 = 38
	VariantVariantMax         uint32 = 39
)

const (
	headerDataFlag64 = 1 << 16
)

// Variant is a minimal tagged union covering the types SoM's RPC layer uses.
type Variant struct {
	Type  uint32
	Bool  bool
	Int   int64
	Float float64
	Str   string
	Vec2  [2]float32
	Arr   []Variant
	Dict  map[string]Variant
	Bytes []byte
}

// Common constructors so the server code reads cleanly.
func NewBool(v bool) Variant               { return Variant{Type: VariantBool, Bool: v} }
func NewInt(v int64) Variant               { return Variant{Type: VariantInt, Int: v} }
func NewFloat(v float64) Variant           { return Variant{Type: VariantFloat, Float: v} }
func NewString(v string) Variant           { return Variant{Type: VariantString, Str: v} }
func NewStringName(v string) Variant       { return Variant{Type: VariantStringName, Str: v} }
func NewVec2(x, y float32) Variant         { return Variant{Type: VariantVector2, Vec2: [2]float32{x, y}} }
func NewArray(v []Variant) Variant         { return Variant{Type: VariantArray, Arr: v} }
func NewDict(v map[string]Variant) Variant { return Variant{Type: VariantDictionary, Dict: v} }

// EncodeVariant serializes a Variant using the uncompressed marshaling format.
// Returns the encoded bytes. The encoding always pads string payloads to a
// multiple of 4 bytes, matching Godot's _encode_string.
func EncodeVariant(v *Variant) ([]byte, error) {
	buf := make([]byte, 0, 64)
	if err := encodeVariantTo(v, &buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// DecodeVariant parses a Variant from the start of buf, returning the variant
// and the number of bytes consumed. Mirrors decode_variant's r_len out-param.
func DecodeVariant(buf []byte) (*Variant, int, error) {
	if len(buf) < 4 {
		return nil, 0, errors.New("buffer too small for variant header")
	}
	header := binary.LittleEndian.Uint32(buf)
	vtype := header & 0xFF
	if vtype >= VariantVariantMax {
		return nil, 0, fmt.Errorf("invalid variant type %d", vtype)
	}
	consumed := 4
	switch vtype {
	case VariantNil:
		return &Variant{Type: VariantNil}, consumed, nil
	case VariantBool:
		if len(buf) < consumed+4 {
			return nil, 0, errors.New("buffer too small for bool")
		}
		b := binary.LittleEndian.Uint32(buf[consumed:]) != 0
		return &Variant{Type: VariantBool, Bool: b}, consumed + 4, nil
	case VariantInt:
		is64 := header&headerDataFlag64 != 0
		if is64 {
			if len(buf) < consumed+8 {
				return nil, 0, errors.New("buffer too small for int64")
			}
			u := binary.LittleEndian.Uint64(buf[consumed:])
			return &Variant{Type: VariantInt, Int: int64(u)}, consumed + 8, nil
		}
		if len(buf) < consumed+4 {
			return nil, 0, errors.New("buffer too small for int32")
		}
		u := binary.LittleEndian.Uint32(buf[consumed:])
		// Godot encodes negative ints as their two's-complement uint32, so
		// sign-extend back to int64.
		return &Variant{Type: VariantInt, Int: int64(int32(u))}, consumed + 4, nil
	case VariantFloat:
		is64 := header&headerDataFlag64 != 0
		if is64 {
			if len(buf) < consumed+8 {
				return nil, 0, errors.New("buffer too small for float64")
			}
			bits := binary.LittleEndian.Uint64(buf[consumed:])
			return &Variant{Type: VariantFloat, Float: math.Float64frombits(bits)}, consumed + 8, nil
		}
		if len(buf) < consumed+4 {
			return nil, 0, errors.New("buffer too small for float32")
		}
		bits := binary.LittleEndian.Uint32(buf[consumed:])
		return &Variant{Type: VariantFloat, Float: float64(math.Float32frombits(bits))}, consumed + 4, nil
	case VariantString, VariantStringName:
		str, n, err := decodeString(buf[consumed:])
		if err != nil {
			return nil, 0, err
		}
		return &Variant{Type: vtype, Str: str}, consumed + n, nil
	case VariantVector2:
		if len(buf) < consumed+8 {
			return nil, 0, errors.New("buffer too small for vector2")
		}
		var v2 [2]float32
		v2[0] = math.Float32frombits(binary.LittleEndian.Uint32(buf[consumed:]))
		v2[1] = math.Float32frombits(binary.LittleEndian.Uint32(buf[consumed+4:]))
		return &Variant{Type: VariantVector2, Vec2: v2}, consumed + 8, nil
	case VariantVector2i:
		if len(buf) < consumed+8 {
			return nil, 0, errors.New("buffer too small for vector2i")
		}
		return &Variant{Type: VariantVector2i, Int: int64(binary.LittleEndian.Uint64(buf[consumed:]))}, consumed + 8, nil
	case VariantArray:
		// Typed-array flags live in bits 16-17 of the header. A non-zero kind
		// means the payload starts with a container-type descriptor. SoM's
		// RPC arguments are untyped arrays, so kind 0 is the expected path.
		kind := (header >> 16) & 0b11
		if kind != 0 {
			n, err := skipContainerType(buf[consumed:], kind)
			if err != nil {
				return nil, 0, err
			}
			consumed += n
		}
		if len(buf) < consumed+4 {
			return nil, 0, errors.New("buffer too small for array size")
		}
		count := binary.LittleEndian.Uint32(buf[consumed:])
		consumed += 4
		if count > maxVariantElements {
			return nil, 0, fmt.Errorf("array count %d exceeds limit %d", count, maxVariantElements)
		}
		arr := make([]Variant, 0, count)
		for i := uint32(0); i < count; i++ {
			ev, en, err := DecodeVariant(buf[consumed:])
			if err != nil {
				return nil, 0, fmt.Errorf("array elem %d: %w", i, err)
			}
			arr = append(arr, *ev)
			consumed += en
		}
		return &Variant{Type: VariantArray, Arr: arr}, consumed, nil
	case VariantDictionary:
		// Typed-dictionary flags live in bits 16-19: key kind in 16-17, value
		// kind in 18-19. SoM's RPC arguments use untyped dictionaries.
		keyKind := (header >> 16) & 0b11
		valKind := (header >> 18) & 0b11
		if keyKind != 0 {
			n, err := skipContainerType(buf[consumed:], keyKind)
			if err != nil {
				return nil, 0, err
			}
			consumed += n
		}
		if valKind != 0 {
			n, err := skipContainerType(buf[consumed:], valKind)
			if err != nil {
				return nil, 0, err
			}
			consumed += n
		}
		if len(buf) < consumed+4 {
			return nil, 0, errors.New("buffer too small for dict size")
		}
		count := binary.LittleEndian.Uint32(buf[consumed:])
		consumed += 4
		if count > maxVariantElements {
			return nil, 0, fmt.Errorf("dict count %d exceeds limit %d", count, maxVariantElements)
		}
		dict := make(map[string]Variant, count)
		for i := uint32(0); i < count; i++ {
			kv, kn, err := DecodeVariant(buf[consumed:])
			if err != nil {
				return nil, 0, fmt.Errorf("dict key %d: %w", i, err)
			}
			consumed += kn
			vv, vn, err := DecodeVariant(buf[consumed:])
			if err != nil {
				return nil, 0, fmt.Errorf("dict value %d: %w", i, err)
			}
			consumed += vn
			dict[variantKeyString(kv)] = *vv
		}
		return &Variant{Type: VariantDictionary, Dict: dict}, consumed, nil
	case VariantPackedByteArray:
		if len(buf) < consumed+4 {
			return nil, 0, errors.New("buffer too small for pba size")
		}
		count := binary.LittleEndian.Uint32(buf[consumed:])
		consumed += 4
		if count > maxVariantElements {
			return nil, 0, fmt.Errorf("pba count %d exceeds limit %d", count, maxVariantElements)
		}
		if len(buf) < consumed+int(count) {
			return nil, 0, errors.New("buffer too small for pba data")
		}
		b := make([]byte, count)
		copy(b, buf[consumed:consumed+int(count)])
		consumed += int(count)
		for consumed%4 != 0 {
			consumed++
		}
		return &Variant{Type: VariantPackedByteArray, Bytes: b}, consumed, nil
	case VariantPackedStringArray:
		if len(buf) < consumed+4 {
			return nil, 0, errors.New("buffer too small for psa size")
		}
		count := binary.LittleEndian.Uint32(buf[consumed:])
		consumed += 4
		arr := make([]Variant, 0, count)
		for i := uint32(0); i < count; i++ {
			sv, sn, err := DecodeStringAsVariant(buf[consumed:])
			if err != nil {
				return nil, 0, fmt.Errorf("psa elem %d: %w", i, err)
			}
			arr = append(arr, *sv)
			consumed += sn
		}
		return &Variant{Type: VariantArray, Arr: arr}, consumed, nil
	case VariantRID:
		if len(buf) < consumed+8 {
			return nil, 0, errors.New("buffer too small for rid")
		}
		return &Variant{Type: VariantRID, Int: int64(binary.LittleEndian.Uint64(buf[consumed:]))}, consumed + 8, nil
	case VariantObject:
		// Object-as-ID: 8-byte ObjectID. Not yet supported for encoding.
		if len(buf) < consumed+8 {
			return nil, 0, errors.New("buffer too small for object id")
		}
		return &Variant{Type: VariantObject, Int: int64(binary.LittleEndian.Uint64(buf[consumed:]))}, consumed + 8, nil
	case VariantNodePath:
		// NodePath encoding: 12-byte header + strings. Needed for scene cache.
		if len(buf) < consumed+12 {
			return nil, 0, errors.New("buffer too small for nodepath header")
		}
		nameCount := binary.LittleEndian.Uint32(buf[consumed:]) & 0x7FFFFFFF
		subCount := binary.LittleEndian.Uint32(buf[consumed+4:])
		consumed += 12
		var parts []string
		total := int(nameCount) + int(subCount)
		for i := 0; i < total; i++ {
			s, n, err := decodeString(buf[consumed:])
			if err != nil {
				return nil, 0, fmt.Errorf("nodepath part %d: %w", i, err)
			}
			parts = append(parts, s)
			consumed += n
		}
		return &Variant{Type: VariantNodePath, Str: joinNodePath(parts)}, consumed, nil
	}
	return nil, 0, fmt.Errorf("unsupported variant type %d", vtype)
}

// encodeVariantTo appends the uncompressed encoding of v to buf.
func encodeVariantTo(v *Variant, buf *[]byte) error {
	var header uint32 = v.Type
	var payload []byte
	switch v.Type {
	case VariantNil:
		// header only
	case VariantBool:
		var b uint32
		if v.Bool {
			b = 1
		}
		payload = encodeU32(b)
	case VariantInt:
		if v.Int > math.MaxInt32 || v.Int < math.MinInt32 {
			header |= headerDataFlag64
			payload = encodeU64(uint64(v.Int))
		} else {
			payload = encodeU32(uint32(v.Int))
		}
	case VariantFloat:
		f32 := float32(v.Float)
		if float64(f32) != v.Float {
			header |= headerDataFlag64
			payload = encodeU64(math.Float64bits(v.Float))
		} else {
			payload = encodeU32(math.Float32bits(f32))
		}
	case VariantString, VariantStringName:
		payload = encodeString(v.Str)
	case VariantVector2:
		payload = make([]byte, 8)
		binary.LittleEndian.PutUint32(payload[0:], math.Float32bits(v.Vec2[0]))
		binary.LittleEndian.PutUint32(payload[4:], math.Float32bits(v.Vec2[1]))
	case VariantVector2i:
		payload = encodeU64(uint64(v.Int))
	case VariantArray:
		// Untyped array: no container-type descriptor, just the element count.
		payload = append(payload, encodeU32(uint32(len(v.Arr)))...)
		for i := range v.Arr {
			if err := encodeVariantTo(&v.Arr[i], &payload); err != nil {
				return err
			}
		}
	case VariantDictionary:
		// Untyped dictionary: no container-type descriptors, just the count.
		payload = append(payload, encodeU32(uint32(len(v.Dict)))...)
		// Godot preserves insertion order; Go maps don't. For deterministic
		// encoding we sort keys. SoM's dicts are small, so this is cheap.
		for _, k := range sortedDictKeys(v.Dict) {
			kv := NewString(k)
			if err := encodeVariantTo(&kv, &payload); err != nil {
				return err
			}
			val := v.Dict[k]
			if err := encodeVariantTo(&val, &payload); err != nil {
				return err
			}
		}
	case VariantPackedByteArray:
		payload = encodeU32(uint32(len(v.Bytes)))
		payload = append(payload, v.Bytes...)
		for len(payload)%4 != 0 {
			payload = append(payload, 0)
		}
	case VariantPackedStringArray:
		payload = encodeU32(uint32(len(v.Arr)))
		for i := range v.Arr {
			payload = append(payload, encodeString(v.Arr[i].Str)...)
		}
	case VariantRID:
		payload = encodeU64(uint64(v.Int))
	default:
		return fmt.Errorf("unsupported variant type %d for encoding", v.Type)
	}
	*buf = append(*buf, encodeU32(header)...)
	*buf = append(*buf, payload...)
	return nil
}

// --- helpers ---

func encodeU32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func encodeU64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func encodeString(s string) []byte {
	b := []byte(s)
	out := encodeU32(uint32(len(b)))
	out = append(out, b...)
	for len(out)%4 != 0 {
		out = append(out, 0)
	}
	return out
}

func decodeString(buf []byte) (string, int, error) {
	if len(buf) < 4 {
		return "", 0, errors.New("buffer too small for string length")
	}
	slen := int(binary.LittleEndian.Uint32(buf))
	pad := 0
	if slen%4 != 0 {
		pad = 4 - slen%4
	}
	if slen < 0 || len(buf) < 4+slen+pad {
		return "", 0, errors.New("buffer too small for string data")
	}
	s := string(buf[4 : 4+slen])
	return s, 4 + slen + pad, nil
}

// DecodeStringAsVariant decodes a raw string (no variant header) and wraps it
// as a VariantString. Used by PackedStringArray where elements are encoded as
// raw strings, not full variants.
func DecodeStringAsVariant(buf []byte) (*Variant, int, error) {
	s, n, err := decodeString(buf)
	if err != nil {
		return nil, 0, err
	}
	return &Variant{Type: VariantString, Str: s}, n, nil
}

// skipContainerType consumes a typed container descriptor from buf, returning
// the number of bytes read. kind 0 = NONE (0 bytes), 1 = BUILTIN (uint32),
// 2 = CLASS_NAME (string), 3 = SCRIPT (string).
func skipContainerType(buf []byte, kind uint32) (int, error) {
	switch kind {
	case 0:
		return 0, nil
	case 1:
		if len(buf) < 4 {
			return 0, errors.New("buffer too small for builtin container type")
		}
		return 4, nil
	case 2, 3:
		_, n, err := decodeString(buf)
		if err != nil {
			return 0, err
		}
		return n, nil
	}
	return 0, fmt.Errorf("unknown container type kind %d", kind)
}

func variantKeyString(v *Variant) string {
	switch v.Type {
	case VariantString, VariantStringName:
		return v.Str
	case VariantInt:
		return fmt.Sprintf("%d", v.Int)
	}
	return fmt.Sprintf("%v", v.Str)
}

func joinNodePath(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "/"
		}
		out += p
	}
	return out
}

func sortedDictKeys(m map[string]Variant) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Insertion sort to avoid importing sort for a tiny slice.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
