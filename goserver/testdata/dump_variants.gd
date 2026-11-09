extends SceneTree

# Dump Godot Variant binary encodings for cross-validation against the Go
# implementation. Run with:
#   godot --headless --script testdata/dump_variants.gd
#
# Uses var_to_bytes() which calls encode_variant under the hood (full objects
# disabled, matching the multiplayer path).

func _init() -> void:
	var cases := {
		"nil": null,
		"true": true,
		"false": false,
		"int_small": 42,
		"int_neg": -7,
		"int_i16": 1000,
		"int_i32": 70000,
		"int_i64": 5000000000,
		"float32": 1.5,
		"float64": 0.1,
		"string_empty": "",
		"string_ascii": "hello",
		"string_uni": "你好",
		"vector2": Vector2(1.5, -2.25),
		"vector2i": Vector2i(3, -4),
		"array_int": [1, 2, 3],
		"array_mixed": [1, "two", true],
		"dict": {"a": 1, "b": "two"},
		"packed_bytes": PackedByteArray([1, 2, 3, 255]),
	}
	for key in cases:
		var v: Variant = cases[key]
		var buf := var_to_bytes(v)
		var hex := buf.hex_encode()
		print("%s=%s" % [key, hex])
	quit()
