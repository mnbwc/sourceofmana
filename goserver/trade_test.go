package main

// Phase-3 trade loopback test: /trade -> TradeRequested -> TradeRespond ->
// TradeOpened + TradeUpdate -> TradeSetOffer -> TradeLock (both) -> TradeClosed.

import (
	"log"
	"testing"

	"github.com/gorilla/websocket"
)

// TestTradeLoopback verifies the trade state machine between two simulated
// clients: request, accept, offer, lock, commit.
func TestTradeLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")

	enterPartyWorld(t, a, "tuser1", "Telia")
	enterPartyWorld(t, b, "tuser2", "Tpop")

	// A initiates a trade via /trade "Tpop".
	pkt, _ := BuildRPCPacket(5, rpcTriggerCommand, []Variant{NewString("trade \"Tpop\"")})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	tr := expectPush(t, b, rpcTradeRequested)
	if tr.Args[0].Str != "Telia" {
		t.Fatalf("TradeRequested: want requester Telia, got %s", formatArgs(tr.Args))
	}
	// A also gets a PushNotification (system chat).
	_ = expectPush(t, a, rpcPushNotification)

	// B accepts -> both get TradeOpened, then TradeUpdate (revision 0).
	pkt, _ = BuildRPCPacket(5, rpcTradeRespond, []Variant{NewBool(true)})
	b.conn.WriteMessage(websocket.BinaryMessage, pkt)
	openA := expectPush(t, a, rpcTradeOpened)
	openB := expectPush(t, b, rpcTradeOpened)
	if openA.Args[0].Str != "Tpop" || openB.Args[0].Str != "Telia" {
		t.Fatalf("TradeOpened: A=%s B=%s", formatArgs(openA.Args), formatArgs(openB.Args))
	}
	updA := expectPush(t, a, rpcTradeUpdate)
	updB := expectPush(t, b, rpcTradeUpdate)
	if updA.Args[6].Int != 0 || updB.Args[6].Int != 0 {
		t.Fatalf("initial TradeUpdate revision: A=%d B=%d", updA.Args[6].Int, updB.Args[6].Int)
	}

	// A sets an offer (2x item 42) + 10 GP -> both get TradeUpdate revision 1,
	// both unlocked.
	offer := Variant{Type: VariantArray, Arr: []Variant{
		{Type: VariantArray, Arr: []Variant{NewInt(42), NewInt(2), NewString("")}},
	}}
	pkt, _ = BuildRPCPacket(5, rpcTradeSetOffer, []Variant{offer, NewInt(10)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	updA = expectPush(t, a, rpcTradeUpdate)
	updB = expectPush(t, b, rpcTradeUpdate)
	if updA.Args[6].Int != 1 || updB.Args[6].Int != 1 {
		t.Fatalf("offer TradeUpdate revision: A=%d B=%d", updA.Args[6].Int, updB.Args[6].Int)
	}
	if updA.Args[1].Int != 10 {
		t.Fatalf("own GP: want 10, got %s", formatArgs(updA.Args))
	}

	// A locks (revision 1) -> TradeUpdate: A ownLocked=true, B otherLocked=true
	// (B sees A's lock as its "other" side), B ownLocked=false.
	pkt, _ = BuildRPCPacket(5, rpcTradeLock, []Variant{NewInt(1), NewBool(true)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	updA = expectPush(t, a, rpcTradeUpdate)
	updB = expectPush(t, b, rpcTradeUpdate)
	if !updA.Args[4].Bool || updA.Args[5].Bool {
		t.Fatalf("A lock: A ownLocked=%v otherLocked=%v", updA.Args[4].Bool, updA.Args[5].Bool)
	}
	if updB.Args[4].Bool || !updB.Args[5].Bool {
		t.Fatalf("B view: B ownLocked=%v otherLocked=%v", updB.Args[4].Bool, updB.Args[5].Bool)
	}

	// B locks (revision 1) -> both locked -> commit -> TradeClosed to both.
	pkt, _ = BuildRPCPacket(5, rpcTradeLock, []Variant{NewInt(1), NewBool(true)})
	b.conn.WriteMessage(websocket.BinaryMessage, pkt)
	closedA := expectPush(t, a, rpcTradeClosed)
	closedB := expectPush(t, b, rpcTradeClosed)
	if closedA.Args[0].Str != "Trade completed." || closedB.Args[0].Str != "Trade completed." {
		t.Fatalf("TradeClosed: A=%s B=%s", formatArgs(closedA.Args), formatArgs(closedB.Args))
	}
}

// TestTradeDecline verifies the decline path: request -> decline -> no session.
func TestTradeDecline(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")

	enterPartyWorld(t, a, "tuser3", "Telia2")
	enterPartyWorld(t, b, "tuser4", "Tpop2")

	pkt, _ := BuildRPCPacket(5, rpcTriggerCommand, []Variant{NewString("trade \"Tpop2\"")})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	_ = expectPush(t, b, rpcTradeRequested)
	_ = expectPush(t, a, rpcPushNotification)

	// B declines -> A gets a PushNotification; no TradeOpened is sent.
	pkt, _ = BuildRPCPacket(5, rpcTradeRespond, []Variant{NewBool(false)})
	b.conn.WriteMessage(websocket.BinaryMessage, pkt)
	note := expectPush(t, a, rpcPushNotification)
	if note.Args[0].Str == "" {
		t.Fatalf("decline notification empty: %s", formatArgs(note.Args))
	}
	log.Printf("decline: %s", note.Args[0].Str)
}
