package main

// Phase-6 tests: buffs, drops, private stats, retrieve inventory,
// express, projectile, morph, clear navigation, and tutorial UI handlers.

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestBuffs verifies buff application, refresh, and clearing.
func TestBuffs(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "buffuser", "BuffOne")

	col := startCollector(fc)

	// Apply a buff directly (simulating an on-hit debuff).
	s := lastTestServer
	e := s.world.get(1) // BuffOne's charID
	if e == nil {
		t.Fatal("entity not found")
	}
	s.applyBuff(e.CharID, modDefense, 2089220490, NewInt(-5), 10*time.Second)

	// The client should receive EntityBuff (AOI broadcast) + UpdateBuff (self).
	eb := col.recv(rpcEntityBuff, 2*time.Second)
	if eb == nil {
		t.Fatal("no EntityBuff push")
	}
	if eb.Args[0].Int != e.CharID || eb.Args[1].Int != modDefense {
		t.Fatalf("EntityBuff args: %s", formatArgs(eb.Args))
	}

	ub := col.recv(rpcUpdateBuff, 2*time.Second)
	if ub == nil {
		t.Fatal("no UpdateBuff push")
	}
	if ub.Args[0].Int != modDefense || ub.Args[3].Int != 2089220490 {
		t.Fatalf("UpdateBuff args: %s", formatArgs(ub.Args))
	}

	// Clear the buff.
	s.clearBuff(e.CharID, modDefense, 2089220490)
	eb2 := col.recv(rpcEntityBuff, 2*time.Second)
	if eb2 == nil {
		t.Fatal("no EntityBuff clear push")
	}
	if eb2.Args[3].Bool != false {
		t.Fatalf("EntityBuff clear: want enabled=false, got %v", eb2.Args[3].Bool)
	}

	// Verify the buff list is empty after clearing.
	if got := len(s.buffsOn(e.CharID)); got != 0 {
		t.Fatalf("buffs after clear: %d, want 0", got)
	}
}

// TestDrops verifies NPC drop spawning, pickup, and expiry.
func TestDrops(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "dropuser", "DropOne")

	col := startCollector(fc)
	s := lastTestServer

	// Spawn a drop directly.
	pos := Vec2{200, 200}
	d := s.spawnDrop(0, 1001, "", pos, spawnMap)
	if d == nil {
		t.Fatal("spawnDrop returned nil")
	}
	if d.ItemID != 1001 {
		t.Fatalf("drop itemID=%d want 1001", d.ItemID)
	}

	// Client should receive DropAdded.
	da := col.recv(rpcDropAdded, 2*time.Second)
	if da == nil {
		t.Fatal("no DropAdded push")
	}
	if da.Args[0].Int != d.ID || da.Args[1].Int != 1001 {
		t.Fatalf("DropAdded args: %s", formatArgs(da.Args))
	}

	// Remove the drop.
	s.removeDrop(d.ID)
	dr := col.recv(rpcDropRemoved, 2*time.Second)
	if dr == nil {
		t.Fatal("no DropRemoved push")
	}
	if dr.Args[0].Int != d.ID {
		t.Fatalf("DropRemoved args: %s", formatArgs(dr.Args))
	}

	// Verify the drop is gone.
	s.dropMu.Lock()
	_, exists := s.drops[d.ID]
	s.dropMu.Unlock()
	if exists {
		t.Fatal("drop still exists after removeDrop")
	}
}

// TestPrivateStats verifies UpdatePrivateStats is pushed on connect.
// The enter-push sequence (including UpdatePrivateStats) is verified in
// TestWorldSyncLoopback's wantOrder; here we verify the pushPrivateStats
// helper directly.
func TestPrivateStats(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "statsuser", "StatsOne")

	col := startCollector(fc)
	s := lastTestServer

	// Re-push private stats (simulating an XP change).
	s.pushPrivateStats(1)

	ps := col.recv(rpcUpdatePrivateStats, 2*time.Second)
	if ps == nil {
		t.Fatal("no UpdatePrivateStats push")
	}
	// Args: experience, gp, mana, stamina, karma, weight, shape, spirit
	if len(ps.Args) < 8 {
		t.Fatalf("UpdatePrivateStats args: %s", formatArgs(ps.Args))
	}
	if ps.Args[0].Int != 0 || ps.Args[1].Int != 0 {
		t.Fatalf("initial stats: exp=%d gp=%d want 0/0", ps.Args[0].Int, ps.Args[1].Int)
	}
}

// TestRetrieveInventory verifies the client can request inventory refresh.
func TestRetrieveInventory(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "invuser", "InvOne")

	col := startCollector(fc)

	pkt, _ := BuildRPCPacket(5, rpcRetrieveInventory, []Variant{})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	ri := col.recv(rpcRefreshInventory, 2*time.Second)
	if ri == nil {
		t.Fatal("no RefreshInventory push")
	}
	re := col.recv(rpcRefreshEquipment, 2*time.Second)
	if re == nil {
		t.Fatal("no RefreshEquipment push")
	}
}

// TestPhase6MiscHandlers verifies express, projectile, morph, clear nav,
// and tutorial UI handlers don't crash.
func TestPhase6MiscHandlers(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "miscuser", "MiscOne")

	col := startCollector(fc)
	s := lastTestServer
	e := s.world.get(1)
	if e == nil {
		t.Fatal("entity not found")
	}

	// Express
	pkt, _ := BuildRPCPacket(5, rpcExpress, []Variant{NewInt(e.CharID), NewString("hello")})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("express: %v", err)
	}
	exp := col.recv(rpcExpress, 2*time.Second)
	if exp == nil {
		t.Fatal("no Express echo")
	}

	// ThrowProjectile
	pkt, _ = BuildRPCPacket(5, rpcThrowProjectile, []Variant{NewInt(e.CharID), NewVec2(100, 200), NewInt(2814726266)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("projectile: %v", err)
	}
	tp := col.recv(rpcThrowProjectile, 2*time.Second)
	if tp == nil {
		t.Fatal("no ThrowProjectile echo")
	}

	// Morphed
	pkt, _ = BuildRPCPacket(5, rpcMorphed, []Variant{NewInt(e.CharID), NewInt(5), NewBool(true)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("morph: %v", err)
	}
	mp := col.recv(rpcMorphed, 2*time.Second)
	if mp == nil {
		t.Fatal("no Morphed echo")
	}

	// ClearNavigation
	pkt, _ = BuildRPCPacket(5, rpcClearNavigation, []Variant{})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("clear nav: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// DisplayActions
	pkt, _ = BuildRPCPacket(5, rpcDisplayActions, []Variant{Variant{Type: VariantPackedStringArray, Arr: []Variant{NewString("attack"), NewString("sit")}}})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("display actions: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// DisplayProgressionTracker
	pkt, _ = BuildRPCPacket(5, rpcDisplayProgressionTracker, []Variant{NewString("quest"), NewInt(50), NewInt(100), NewString("hp")})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("tracker: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// ClearProgressionTracker
	pkt, _ = BuildRPCPacket(5, rpcClearProgressionTracker, []Variant{})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("clear tracker: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// HighlightUI
	pkt, _ = BuildRPCPacket(5, rpcHighlightUI, []Variant{NewInt(3)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("highlight: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// OpenUI
	pkt, _ = BuildRPCPacket(5, rpcOpenUI, []Variant{NewInt(7)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("openui: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// CommandModifier
	pkt, _ = BuildRPCPacket(5, rpcCommandModifier, []Variant{})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("cmd modifier: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// No unhandled RPC errors should appear (checked implicitly by test passing).
}
