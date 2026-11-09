package main

// Cross-validation against Godot 4.7.1 var_to_bytes() output.
// The hex strings below were captured by running:
//   godot --headless --script testdata/dump_variants.gd

import (
	"encoding/hex"
	"math"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func TestDecodeGodotOutput(t *testing.T) {
	cases := []struct {
		name  string
		hex   string
		check func(*testing.T, *Variant)
	}{
		{"nil", "00000000", func(t *testing.T, v *Variant) {
			if v.Type != VariantNil {
				t.Fatalf("type = %d, want NIL", v.Type)
			}
		}},
		{"true", "0100000001000000", func(t *testing.T, v *Variant) {
			if v.Type != VariantBool || !v.Bool {
				t.Fatalf("got type=%d bool=%v, want true", v.Type, v.Bool)
			}
		}},
		{"false", "0100000000000000", func(t *testing.T, v *Variant) {
			if v.Type != VariantBool || v.Bool {
				t.Fatalf("got type=%d bool=%v, want false", v.Type, v.Bool)
			}
		}},
		{"int_small", "020000002a000000", func(t *testing.T, v *Variant) {
			if v.Int != 42 {
				t.Fatalf("int = %d, want 42", v.Int)
			}
		}},
		{"int_neg", "02000000f9ffffff", func(t *testing.T, v *Variant) {
			if v.Int != -7 {
				t.Fatalf("int = %d, want -7", v.Int)
			}
		}},
		{"int_i16", "02000000e8030000", func(t *testing.T, v *Variant) {
			if v.Int != 1000 {
				t.Fatalf("int = %d, want 1000", v.Int)
			}
		}},
		{"int_i32", "0200000070110100", func(t *testing.T, v *Variant) {
			if v.Int != 70000 {
				t.Fatalf("int = %d, want 70000", v.Int)
			}
		}},
		{"int_i64", "0200010000f2052a01000000", func(t *testing.T, v *Variant) {
			if v.Int != 5000000000 {
				t.Fatalf("int = %d, want 5000000000", v.Int)
			}
		}},
		{"float32", "030000000000c03f", func(t *testing.T, v *Variant) {
			if math.Abs(v.Float-1.5) > 1e-6 {
				t.Fatalf("float = %v, want 1.5", v.Float)
			}
		}},
		{"float64", "030001009a9999999999b93f", func(t *testing.T, v *Variant) {
			if math.Abs(v.Float-0.1) > 1e-12 {
				t.Fatalf("float = %v, want 0.1", v.Float)
			}
		}},
		{"string_empty", "0400000000000000", func(t *testing.T, v *Variant) {
			if v.Str != "" {
				t.Fatalf("str = %q, want empty", v.Str)
			}
		}},
		{"string_ascii", "040000000500000068656c6c6f000000", func(t *testing.T, v *Variant) {
			if v.Str != "hello" {
				t.Fatalf("str = %q, want hello", v.Str)
			}
		}},
		{"string_uni", "0400000006000000e4bda0e5a5bd0000", func(t *testing.T, v *Variant) {
			if v.Str != "你好" {
				t.Fatalf("str = %q, want 你好", v.Str)
			}
		}},
		{"vector2", "050000000000c03f000010c0", func(t *testing.T, v *Variant) {
			if v.Type != VariantVector2 || v.Vec2[0] != 1.5 || v.Vec2[1] != -2.25 {
				t.Fatalf("vec2 = %v, want (1.5, -2.25)", v.Vec2)
			}
		}},
		{"vector2i", "0600000003000000fcffffff", func(t *testing.T, v *Variant) {
			if v.Type != VariantVector2i {
				t.Fatalf("type = %d, want Vector2i", v.Type)
			}
			// Packed as two int32: x=3, y=-4.
			x := int32(v.Int & 0xFFFFFFFF)
			y := int32(v.Int >> 32)
			if x != 3 || y != -4 {
				t.Fatalf("vec2i = (%d, %d), want (3, -4)", x, y)
			}
		}},
		{"array_int", "1c00000003000000020000000100000002000000020000000200000003000000", func(t *testing.T, v *Variant) {
			if v.Type != VariantArray || len(v.Arr) != 3 {
				t.Fatalf("arr = %+v, want 3 elems", v)
			}
			if v.Arr[0].Int != 1 || v.Arr[1].Int != 2 || v.Arr[2].Int != 3 {
				t.Fatalf("arr = %+v, want [1,2,3]", v.Arr)
			}
		}},
		{"dict", "1b000000020000000400000001000000610000000200000001000000040000000100000062000000040000000300000074776f00", func(t *testing.T, v *Variant) {
			if v.Type != VariantDictionary || len(v.Dict) != 2 {
				t.Fatalf("dict = %+v, want 2 keys", v)
			}
			if v.Dict["a"].Int != 1 {
				t.Fatalf("dict[a] = %+v, want 1", v.Dict["a"])
			}
			if v.Dict["b"].Str != "two" {
				t.Fatalf("dict[b] = %+v, want two", v.Dict["b"])
			}
		}},
		{"packed_bytes", "1d00000004000000010203ff", func(t *testing.T, v *Variant) {
			if v.Type != VariantPackedByteArray || len(v.Bytes) != 4 {
				t.Fatalf("pba = %+v, want 4 bytes", v)
			}
			if v.Bytes[0] != 1 || v.Bytes[1] != 2 || v.Bytes[2] != 3 || v.Bytes[3] != 255 {
				t.Fatalf("pba = %v, want [1 2 3 255]", v.Bytes)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, n, err := DecodeVariant(mustHex(t, tc.hex))
			if err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if n != len(mustHex(t, tc.hex)) {
				t.Fatalf("consumed %d bytes, want %d", n, len(mustHex(t, tc.hex)))
			}
			tc.check(t, v)
		})
	}
}

// TestEncodeRoundTrip verifies our encoder produces bytes Godot can decode,
// by re-encoding decoded values and checking the round trip is stable.
func TestEncodeRoundTrip(t *testing.T) {
	originals := map[string]string{
		"int_small":    "020000002a000000",
		"string_ascii": "040000000500000068656c6c6f000000",
		"vector2":      "050000000000c03f000010c0",
	}
	for name, h := range originals {
		t.Run(name, func(t *testing.T) {
			orig := mustHex(t, h)
			v, _, err := DecodeVariant(orig)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			encoded, err := EncodeVariant(v)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if hex.EncodeToString(encoded) != h {
				t.Fatalf("round trip mismatch:\n got %s\nwant %s", hex.EncodeToString(encoded), h)
			}
		})
	}
}
