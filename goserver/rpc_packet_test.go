package main

// End-to-end validation against packets captured from a real Godot 4.7.1
// client (see testdata/dump_rpc_packet.gd). The hex strings are the actual
// bytes Godot produced for CreateAccount and SetMovePos RPCs.

import (
	"encoding/hex"
	"math"
	"testing"
)

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex: %v", err)
	}
	return b
}

func TestParseCreateAccountPacket(t *testing.T) {
	// Captured from Godot 4.7.1: CreateAccount("alice", "secret", "a@b.c", true, 0)
	hex := "203a00008000050400000005000000616c6963650000000400000006000000736563726574000004000000050000006140622e6300000081020044756d7052706300"
	buf := mustDecodeHex(t, hex)

	cmd, rpc, _, _, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if cmd != CmdRemoteCall {
		t.Fatalf("cmd = %d, want REMOTE_CALL", cmd)
	}
	if rpc == nil {
		t.Fatal("rpc is nil")
	}
	if rpc.NameID != 0 {
		t.Fatalf("NameID = %d, want 0", rpc.NameID)
	}
	if len(rpc.Args) != 5 {
		t.Fatalf("len(args) = %d, want 5", len(rpc.Args))
	}
	if rpc.Args[0].Str != "alice" || rpc.Args[1].Str != "secret" || rpc.Args[2].Str != "a@b.c" {
		t.Fatalf("string args = %q %q %q", rpc.Args[0].Str, rpc.Args[1].Str, rpc.Args[2].Str)
	}
	if !rpc.Args[3].Bool {
		t.Fatal("arg[3] = false, want true")
	}
	if rpc.Args[4].Int != 0 {
		t.Fatalf("arg[4] = %d, want 0", rpc.Args[4].Int)
	}
	if rpc.NodePath != "DumpRpc" {
		t.Fatalf("NodePath = %q, want DumpRpc", rpc.NodePath)
	}
}

func TestParseSetMovePosPacket(t *testing.T) {
	// Captured from Godot 4.7.1: SetMovePos(Vector2(1.5, -2.25))
	hex := "20130000800101050000000000c03f000010c044756d7052706300"
	buf := mustDecodeHex(t, hex)

	cmd, rpc, _, _, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if cmd != CmdRemoteCall {
		t.Fatalf("cmd = %d, want REMOTE_CALL", cmd)
	}
	if rpc == nil || len(rpc.Args) != 1 {
		t.Fatalf("rpc = %+v", rpc)
	}
	a := rpc.Args[0]
	if a.Type != VariantVector2 {
		t.Fatalf("arg type = %d, want Vector2", a.Type)
	}
	if math.Abs(float64(a.Vec2[0])-1.5) > 1e-6 || math.Abs(float64(a.Vec2[1])+2.25) > 1e-6 {
		t.Fatalf("vec2 = %v, want (1.5, -2.25)", a.Vec2)
	}
}

func TestParseSimplifyPathPacket(t *testing.T) {
	// Captured from Godot 4.7.1: SIMPLIFY_PATH cacheID=1 path="E2EClient"
	hex := "0132323166646365646234663237386266346261313238616230343230356466640001000000453245436c69656e7400"
	buf := mustDecodeHex(t, hex)

	cmd, _, cacheID, path, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if cmd != CmdSimplifyPath {
		t.Fatalf("cmd = %d, want SIMPLIFY_PATH", cmd)
	}
	if cacheID != 1 {
		t.Fatalf("cacheID = %d, want 1", cacheID)
	}
	if path != "E2EClient" {
		t.Fatalf("path = %q, want E2EClient", path)
	}
}

func TestBuildReplyPacket(t *testing.T) {
	// The Go server's reply to CreateAccount: AuthTokenResult("alice", "tok-12345")
	// targeting cacheID 1 with nameID 0.
	packet, err := BuildRPCPacket(1, 0, []Variant{NewString("alice"), NewString("tok-12345")})
	if err != nil {
		t.Fatalf("BuildRPCPacket: %v", err)
	}
	cmd, rpc, _, _, err := ParsePacket(packet)
	if err != nil {
		t.Fatalf("ParsePacket on built packet: %v", err)
	}
	if cmd != CmdRemoteCall {
		t.Fatalf("cmd = %d, want REMOTE_CALL", cmd)
	}
	if rpc.NameID != 0 {
		t.Fatalf("NameID = %d, want 0", rpc.NameID)
	}
	if len(rpc.Args) != 2 || rpc.Args[0].Str != "alice" || rpc.Args[1].Str != "tok-12345" {
		t.Fatalf("args = %+v", rpc.Args)
	}
}
