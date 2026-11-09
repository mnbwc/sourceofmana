package main

// RPC argument codec — implements MultiplayerAPI::encode_and_compress_variants
// and decode_and_decompress_variants from scene/main/multiplayer_api.cpp.
//
// This is the format actually used on the wire by SceneMultiplayer RPCs. It
// differs from raw encode_variant in two ways:
//
//  1. BOOL and INT are compressed into a compact form: one meta byte holding
//     the type in bits 0..5 plus an encode-mode in bits 6..7, followed by the
//     smallest signed integer representation that fits (1/2/4/8 bytes).
//  2. All other types fall through to encode_variant unchanged, except that
//     the first header byte is overwritten with the bare type so the decoder
//     can tell a compressed value from an uncompressed one.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	variantMetaTypeMask  = 0x3F
	variantMetaEmodeMask = 0xC0
	variantMetaBoolMask  = 0x80

	encode8  = 0 << 6
	encode16 = 1 << 6
	encode32 = 2 << 6
	encode64 = 3 << 6
)

// EncodeRPCVariants serializes an RPC argument list the way Godot does before
// putting it into a packet. raw is true when the single argument is a
// PackedByteArray, in which case Godot sends the bytes verbatim with no
// variant header at all (see encode_and_compress_variants).
func EncodeRPCVariants(args []Variant) (payload []byte, raw bool, err error) {
	if len(args) == 0 {
		return nil, true, nil
	}
	if len(args) == 1 && args[0].Type == VariantPackedByteArray {
		return args[0].Bytes, true, nil
	}
	for i := range args {
		var n int
		payload, n, err = encodeAndCompressVariant(&args[i], payload)
		if err != nil {
			return nil, false, fmt.Errorf("arg %d: %w", i, err)
		}
		_ = n
	}
	return payload, false, nil
}

// encodeAndCompressVariant appends the compressed encoding of v to dst and
// returns the new slice plus the number of bytes appended.
func encodeAndCompressVariant(v *Variant, dst []byte) ([]byte, int, error) {
	switch v.Type {
	case VariantBool:
		b := byte(VariantBool)
		if v.Bool {
			b |= variantMetaBoolMask
		}
		return append(dst, b), 1, nil
	case VariantInt:
		meta, width, err := intMeta(v.Int)
		if err != nil {
			return dst, 0, err
		}
		start := len(dst)
		dst = append(dst, meta)
		switch width {
		case 1:
			dst = append(dst, byte(int8(v.Int)))
		case 2:
			dst = append(dst, byte(int16(v.Int)), byte(int16(v.Int)>>8))
		case 4:
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], uint32(int32(v.Int)))
			dst = append(dst, b[:]...)
		case 8:
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(v.Int))
			dst = append(dst, b[:]...)
		}
		return dst, len(dst) - start, nil
	default:
		enc, err := EncodeVariant(v)
		if err != nil {
			return dst, 0, err
		}
		// Overwrite the low byte of the header with the bare type, matching
		// Godot's "store the type so we know how to decompress" behavior.
		enc[0] = byte(v.Type)
		return append(dst, enc...), len(enc), nil
	}
}

func intMeta(v int64) (byte, int, error) {
	switch {
	case v >= math.MinInt8 && v <= math.MaxInt8:
		return byte(VariantInt) | encode8, 1, nil
	case v >= math.MinInt16 && v <= math.MaxInt16:
		return byte(VariantInt) | encode16, 2, nil
	case v >= math.MinInt32 && v <= math.MaxInt32:
		return byte(VariantInt) | encode32, 4, nil
	default:
		return byte(VariantInt) | encode64, 8, nil
	}
}

// DecodeRPCVariants parses an RPC argument list from the start of buf,
// returning the variants, the bytes consumed, and whether the payload was
// raw (a single PackedByteArray sent verbatim).
func DecodeRPCVariants(buf []byte, argCount int) ([]Variant, int, bool, error) {
	if argCount == 0 {
		return nil, 0, true, nil
	}
	out := make([]Variant, 0, argCount)
	consumed := 0
	for i := 0; i < argCount; i++ {
		v, n, err := decodeAndDecompressVariant(buf[consumed:])
		if err != nil {
			return nil, 0, false, fmt.Errorf("arg %d: %w", i, err)
		}
		out = append(out, *v)
		consumed += n
	}
	return out, consumed, false, nil
}

// decodeAndDecompressVariant parses a single compressed variant, returning the
// variant and the bytes consumed.
func decodeAndDecompressVariant(buf []byte) (*Variant, int, error) {
	if len(buf) < 1 {
		return nil, 0, errors.New("buffer too small for variant meta")
	}
	meta := buf[0]
	vtype := uint32(meta & variantMetaTypeMask)
	if vtype >= VariantVariantMax {
		return nil, 0, fmt.Errorf("invalid variant type %d", vtype)
	}
	switch vtype {
	case VariantBool:
		return &Variant{Type: VariantBool, Bool: meta&variantMetaBoolMask > 0}, 1, nil
	case VariantInt:
		emode := meta & variantMetaEmodeMask
		switch emode {
		case encode8:
			if len(buf) < 2 {
				return nil, 0, errors.New("buffer too small for int8")
			}
			return &Variant{Type: VariantInt, Int: int64(int8(buf[1]))}, 2, nil
		case encode16:
			if len(buf) < 3 {
				return nil, 0, errors.New("buffer too small for int16")
			}
			u := binary.LittleEndian.Uint16(buf[1:])
			return &Variant{Type: VariantInt, Int: int64(int16(u))}, 3, nil
		case encode32:
			if len(buf) < 5 {
				return nil, 0, errors.New("buffer too small for int32")
			}
			u := binary.LittleEndian.Uint32(buf[1:])
			return &Variant{Type: VariantInt, Int: int64(int32(u))}, 5, nil
		case encode64:
			if len(buf) < 9 {
				return nil, 0, errors.New("buffer too small for int64")
			}
			u := binary.LittleEndian.Uint64(buf[1:])
			return &Variant{Type: VariantInt, Int: int64(u)}, 9, nil
		}
		return nil, 0, fmt.Errorf("unknown int encode mode %d", emode)
	default:
		// Uncompressed: the type was stuffed into the first header byte, so
		// rewrite the header as a proper encode_variant header before decoding.
		patched := make([]byte, len(buf))
		copy(patched, buf)
		header := binary.LittleEndian.Uint32(patched)
		header = (header &^ 0xFF) | vtype
		binary.LittleEndian.PutUint32(patched, header)
		v, n, err := DecodeVariant(patched)
		if err != nil {
			return nil, 0, err
		}
		return v, n, nil
	}
}
