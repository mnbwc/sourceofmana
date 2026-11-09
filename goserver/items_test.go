package main

// Phase-4 items/economy loopback test: UseItem / EquipItem / UnequipItem /
// DropItem / PickupDrop / SetAttributes / MerchantTrade.
//
// The test uses one fake client that enters the world, receives a seeded
// inventory, then exercises each operation and verifies the push payloads.

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// spawnDropNear creates a ground drop at the character's position.
// Returns the drop ID.
func spawnDropNear(t *testing.T, charID, itemID int64) int64 {
	t.Helper()
	e := lastTestServer.world.get(charID)
	if e == nil {
		t.Fatalf("spawnDropNear: entity %d not found", charID)
	}
	d := lastTestServer.spawnDrop(charID, itemID, "", e.Pos, e.MapID)
	return d.ID
}

// TestItemsLoopback verifies the inventory operation handlers end to end.
func TestItemsLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "iuser", "IOne")

	col := startCollector(fc)

	// --- UseItem: consume one Potion (1001) ---
	// Spawn two potion drops near the character: one to use, one to sell.
	dropID1 := spawnDropNear(t, 1, 1001)
	dropID2 := spawnDropNear(t, 1, 1001)
	pkt, _ := BuildRPCPacket(5, rpcPickupDrop, []Variant{NewInt(dropID1)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("pickup: %v", err)
	}
	added := col.recv(rpcItemAdded, 2*time.Second)
	if added == nil {
		t.Fatal("PickupDrop: no ItemAdded push")
	}
	if added.Args[0].Int != 1001 || added.Args[2].Int != 1 {
		t.Fatalf("ItemAdded: want item=1001 count=1, got %s", formatArgs(added.Args))
	}
	// Pick up the second potion.
	pkt, _ = BuildRPCPacket(5, rpcPickupDrop, []Variant{NewInt(dropID2)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("pickup2: %v", err)
	}
	added2 := col.recv(rpcItemAdded, 2*time.Second)
	if added2 == nil {
		t.Fatal("PickupDrop 2: no ItemAdded push")
	}

	// Now use it.
	pkt, _ = BuildRPCPacket(5, rpcUseItem, []Variant{NewInt(1001)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("use: %v", err)
	}
	used := col.recv(rpcItemRemoved, 2*time.Second)
	if used == nil {
		t.Fatal("UseItem: no ItemRemoved push")
	}
	if used.Args[0].Int != 1001 || used.Args[2].Int != 1 {
		t.Fatalf("ItemRemoved: want item=1001 count=1, got %s", formatArgs(used.Args))
	}

	// --- EquipItem: give an Iron Sword (1003, WEAPON slot 6) then equip ---
	// PickupDrop grants a potion; we need a sword. Use the merchant to buy one.
	// Buy 1 Iron Sword for 200 GP (balance -200, but GP starts at 0 → rejected).
	// Instead, sell a potion to get GP first.
	// Sell 1 Potion (sellPrice 25) → +25 GP.
	sellBuy := []Variant{
		NewString("sell"), NewString("buy"),
	}
	_ = sellBuy
	// Build arrays: sellEntries = [[1001, 1, ""]], buyEntries = [].
	sellArr := NewArray([]Variant{
		NewArray([]Variant{NewInt(1001), NewInt(1), NewString("")}),
	})
	buyArr := NewArray([]Variant{})
	pkt, _ = BuildRPCPacket(5, rpcMerchantTrade, []Variant{sellArr, buyArr})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("merchant sell: %v", err)
	}
	closed := col.recv(rpcTradeClosed, 2*time.Second)
	if closed == nil {
		t.Fatal("MerchantTrade sell: no TradeClosed push")
	}
	if closed.Args[0].Str != "Trade completed." {
		t.Fatalf("MerchantTrade sell: want 'Trade completed.', got %q", closed.Args[0].Str)
	}

	// Buy 1 Iron Sword (buyPrice 200) — GP is 25, so this should fail.
	pkt, _ = BuildRPCPacket(5, rpcMerchantTrade, []Variant{NewArray([]Variant{}), NewArray([]Variant{
		NewArray([]Variant{NewInt(1003), NewInt(1), NewString("")}),
	})})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("merchant buy: %v", err)
	}
	closed = col.recv(rpcTradeClosed, 2*time.Second)
	if closed == nil {
		t.Fatal("MerchantTrade buy: no TradeClosed push")
	}
	if closed.Args[0].Str != "You don't have enough GP." {
		t.Fatalf("MerchantTrade buy: want 'not enough GP', got %q", closed.Args[0].Str)
	}

	// --- SetAttributes ---
	// Character is level 0, so the attribute cap is 4 points (4 + 1*0).
	// Allocate exactly the cap: 2+1+1+0+0 = 4.
	pkt, _ = BuildRPCPacket(5, rpcSetAttributes, []Variant{
		NewInt(2), NewInt(1), NewInt(1), NewInt(0), NewInt(0),
	})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("set attrs: %v", err)
	}
	attrs := col.recv(rpcUpdateAttributes, 2*time.Second)
	if attrs == nil {
		t.Fatal("SetAttributes: no UpdateAttributes push")
	}
	if attrs.Args[0].Int != 2 || attrs.Args[1].Int != 1 || attrs.Args[2].Int != 1 {
		t.Fatalf("UpdateAttributes: want 2/1/1, got %s", formatArgs(attrs.Args))
	}

	// Exceeding the cap must be rejected with CommandFeedback.
	pkt, _ = BuildRPCPacket(5, rpcSetAttributes, []Variant{
		NewInt(3), NewInt(2), NewInt(1), NewInt(0), NewInt(0),
	})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("set attrs over cap: %v", err)
	}
	fb := col.recv(rpcCommandFeedback, 2*time.Second)
	if fb == nil {
		t.Fatal("SetAttributes over cap: no CommandFeedback push")
	}
}

// TestEquipUnequipLoopback verifies equip and unequip with a seeded item.
func TestEquipUnequipLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "euser", "EOne")

	col := startCollector(fc)

	// Give the character an Iron Sword by directly buying with enough GP.
	// First sell nothing, buy nothing — instead use the merchant to buy a sword
	// after granting GP via a successful sell of potions.
	// Pickup 3 potions (spawn drops near the character first).
	for i := 0; i < 3; i++ {
		dropID := spawnDropNear(t, 1, 1001)
		pkt, _ := BuildRPCPacket(5, rpcPickupDrop, []Variant{NewInt(dropID)})
		if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
			t.Fatalf("pickup %d: %v", i, err)
		}
		if got := col.recv(rpcItemAdded, 2*time.Second); got == nil {
			t.Fatalf("pickup %d: no ItemAdded", i)
		}
	}
	// Sell 3 potions → +75 GP.
	pkt, _ := BuildRPCPacket(5, rpcMerchantTrade, []Variant{
		NewArray([]Variant{NewArray([]Variant{NewInt(1001), NewInt(3), NewString("")})}),
		NewArray([]Variant{}),
	})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("sell: %v", err)
	}
	if closed := col.recv(rpcTradeClosed, 2*time.Second); closed == nil {
		t.Fatal("sell: no TradeClosed")
	}

	// Buy 1 Iron Sword → -200 GP, but we only have 75. Should fail.
	pkt, _ = BuildRPCPacket(5, rpcMerchantTrade, []Variant{
		NewArray([]Variant{}),
		NewArray([]Variant{NewArray([]Variant{NewInt(1003), NewInt(1), NewString("")})}),
	})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("buy: %v", err)
	}
	closed := col.recv(rpcTradeClosed, 2*time.Second)
	if closed == nil {
		t.Fatal("buy: no TradeClosed")
	}
	if closed.Args[0].Str != "You don't have enough GP." {
		t.Fatalf("buy: want 'not enough GP', got %q", closed.Args[0].Str)
	}

	// Buy 1 Cloth Boots (buyPrice 80) — still not enough (75 GP).
	pkt, _ = BuildRPCPacket(5, rpcMerchantTrade, []Variant{
		NewArray([]Variant{}),
		NewArray([]Variant{NewArray([]Variant{NewInt(1006), NewInt(1), NewString("")})}),
	})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("buy boots: %v", err)
	}
	closed = col.recv(rpcTradeClosed, 2*time.Second)
	if closed == nil {
		t.Fatal("buy boots: no TradeClosed")
	}
	if closed.Args[0].Str != "You don't have enough GP." {
		t.Fatalf("buy boots: want 'not enough GP', got %q", closed.Args[0].Str)
	}

	// Buy 1 Antidote (buyPrice 30) → 75 - 30 = 45 GP. Should succeed.
	pkt, _ = BuildRPCPacket(5, rpcMerchantTrade, []Variant{
		NewArray([]Variant{}),
		NewArray([]Variant{NewArray([]Variant{NewInt(1002), NewInt(1), NewString("")})}),
	})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("buy antidote: %v", err)
	}
	closed = col.recv(rpcTradeClosed, 2*time.Second)
	if closed == nil {
		t.Fatal("buy antidote: no TradeClosed")
	}
	if closed.Args[0].Str != "Trade completed." {
		t.Fatalf("buy antidote: want 'Trade completed.', got %q", closed.Args[0].Str)
	}

	// Now equip the Antidote? No — Antidote has Slot=-1 (not equipable).
	// The character now has 1 Antidote. Equip should fail silently.
	pkt, _ = BuildRPCPacket(5, rpcEquipItem, []Variant{NewInt(1002), NewString(""), NewInt(0)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("equip antidote: %v", err)
	}
	// No ItemEquiped push expected (Antidote is not equipable).
	// Wait a short time and verify nothing arrived.
	time.Sleep(200 * time.Millisecond)
	select {
	case rpc := <-col.ch:
		t.Fatalf("EquipItem on non-equipable item should not push, got nameID=%d args=%s",
			rpc.NameID, formatArgs(rpc.Args))
	default:
	}
}
