package main

// Loopback integration test for the phase-1 login chain: a fake Godot client
// speaks the real wire protocol (SIMPLIFY_PATH + RPC packets) against a live
// WebSocket server, then walks the full login flow:
//
//	LoginWithPassword -> AuthError(ERR_OK) + AuthTokenResult
//	CharacterListing  -> CharacterError(ERR_EMPTY_ACCOUNT)
//	CreateCharacter   -> CharacterInfo + CharacterError(ERR_OK)
//	ConnectCharacter  -> CharacterError(ERR_OK)  [peer is IN_GAME]

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startTestServer spins up the real server on an ephemeral port and returns
// the WebSocket URL plus a shutdown func. It uses an isolated in-memory store
// (no disk persistence) so tests do not leak state into each other.
func startTestServer(t *testing.T) (string, func()) {
	t.Helper()
	s := NewServer()
	s.store = NewAccountStore() // override loadStore() with a clean in-memory DB
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: s.routes()}
	go srv.Serve(ln)
	lastTestServer = s
	return "ws://" + ln.Addr().String() + "/ws", func() { srv.Close() }
}

// lastTestServer holds the most recently started test server so tests can
// inspect its store.
var lastTestServer *Server

// fakeClient is a minimal Godot-protocol client for loopback tests.
type fakeClient struct {
	conn   *websocket.Conn
	peerID uint32
}

func dialFake(t *testing.T, url string) *fakeClient {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var idBuf [4]byte
	// Server pushes the 4-byte LE unique_id right after the handshake.
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read unique_id: %v", err)
	}
	copy(idBuf[:], data)
	fc := &fakeClient{conn: conn, peerID: binary.LittleEndian.Uint32(idBuf[:])}
	if fc.peerID < 2 {
		t.Fatalf("bad unique_id %d (must be >= 2)", fc.peerID)
	}
	return fc
}

// announce tells the server about our Network node (client -> server cache).
func (fc *fakeClient) announce(t *testing.T, cacheID uint32, path string) {
	t.Helper()
	fc.conn.WriteMessage(websocket.BinaryMessage, BuildSimplifyPathPacket(cacheID, "00000000000000000000000000000000", path))
}

// call sends an RPC to the server node and returns the reply packets.
// serverPushNameIDs are nameIDs the server pushes spontaneously (phase-2
// world events). Tests waiting for a specific reply skip these.
var serverPushNameIDs = map[uint16]bool{
	rpcAddOnlinePlayer:    true,
	rpcFullUpdateEntity:   true,
	rpcPreloadEntity:      true,
	rpcPreloadPlayer:      true,
	rpcRefreshOnlineList:  true,
	rpcRemoveOnlinePlayer: true,
	rpcUpdateEntity:       true,
	rpcWarpPlayer:         true,
	// phase-3 party pushes
	rpcPartyInvite: true,
	rpcPartyUpdate: true,
	// phase-3 trade pushes
	rpcTradeRequested:   true,
	rpcTradeOpened:      true,
	rpcTradeUpdate:      true,
	rpcTradeClosed:      true,
	rpcPushNotification: true,
	// phase-3 NPC/quest pushes
	rpcContextChoice:     true,
	rpcContextClose:      true,
	rpcContextContinue:   true,
	rpcContextText:       true,
	rpcContextThink:      true,
	rpcCommandFeedback:   true,
	rpcRefreshProgress:   true,
	rpcUpdatePublicStats: true,
	rpcUpdateQuest:       true,
	// phase-3 inventory/equipment pushes
	rpcRefreshInventory: true,
	rpcRefreshEquipment: true,
	// phase-6 pushes
	rpcUpdatePrivateStats: true,
	rpcRemoveEntity:       true,
	rpcDropAdded:          true,
	rpcDropRemoved:        true,
	// phase-8 pushes
	rpcUpdateBestiary:   true,
	rpcUpdateSkill:      true,
	rpcLevelUp:          true,
	rpcItemAdded:        true,
	rpcItemRemoved:      true,
	rpcUpdateAttributes: true,
}

func (fc *fakeClient) call(t *testing.T, cacheID uint32, nameID uint16, args []Variant) []byte {
	t.Helper()
	pkt, err := BuildRPCPacket(cacheID, nameID, args)
	if err != nil {
		t.Fatalf("build rpc: %v", err)
	}
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write rpc: %v", err)
	}
	// Read packets until we get a non-push RPC reply (skip SIMPLIFY_PATH/
	// CONFIRM_PATH and server-initiated world pushes like WarpPlayer).
	for {
		data := fc.readPacket(t)
		cmd, rpc, _, _, err := ParsePacket(data)
		if err != nil {
			continue // sys packets etc.
		}
		if cmd == CmdRemoteCall && rpc != nil && !serverPushNameIDs[rpc.NameID] {
			return data
		}
	}
}

// callCollect sends an RPC and reads exactly n RPC replies.
func (fc *fakeClient) callCollect(t *testing.T, cacheID uint32, nameID uint16, args []Variant, n int) []*ParsedRPC {
	t.Helper()
	pkt, err := BuildRPCPacket(cacheID, nameID, args)
	if err != nil {
		t.Fatalf("build rpc: %v", err)
	}
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write rpc: %v", err)
	}
	var out []*ParsedRPC
	for len(out) < n {
		data := fc.readPacket(t)
		cmd, rpc, _, _, err := ParsePacket(data)
		if err != nil {
			continue
		}
		if cmd == CmdRemoteCall && rpc != nil {
			out = append(out, rpc)
		}
	}
	return out
}

func (fc *fakeClient) readPacket(t *testing.T) []byte {
	t.Helper()
	fc.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := fc.conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return data
}

func parseReply(t *testing.T, data []byte) *ParsedRPC {
	t.Helper()
	_, rpc, _, _, err := ParsePacket(data)
	if err != nil || rpc == nil {
		t.Fatalf("parse reply %x: %v", data, err)
	}
	return rpc
}

func TestLoginChainLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	const cacheID = uint32(5)
	const path = "/root/Network"

	fc := dialFake(t, url)
	fc.announce(t, cacheID, path)

	// 1. LoginWithPassword with an unknown account -> ERR_AUTH (5)
	rpc := parseReply(t, fc.call(t, cacheID, rpcLoginWithPassword, []Variant{
		NewString("alice"), NewString("secret1"), NewBool(true), NewInt(0)}))
	if rpc.NameID != rpcAuthError || len(rpc.Args) != 1 || rpc.Args[0].Int != authErrAuth {
		t.Fatalf("login unknown: want AuthError(ERR_AUTH=%d), got nameID=%d args=%s",
			authErrAuth, rpc.NameID, formatArgs(rpc.Args))
	}

	// 2. CreateAccount (rememberMe=true) -> AuthTokenResult + AuthError(ERR_OK)
	// Godot's Server.gd order: FinalizeLogin (issues token) runs before the
	// final AuthError, so AuthTokenResult arrives first.
	replies := fc.callCollect(t, cacheID, rpcCreateAccount, []Variant{
		NewString("alice"), NewString("secret1"), NewString("a@b.c"), NewBool(true), NewInt(0)}, 2)
	if replies[0].NameID != rpcAuthTokenResult || replies[0].Args[0].Str != "alice" {
		t.Fatalf("create account token: want AuthTokenResult(alice), got nameID=%d args=%s",
			replies[0].NameID, formatArgs(replies[0].Args))
	}
	if replies[1].NameID != rpcAuthError || replies[1].Args[0].Int != authErrOK {
		t.Fatalf("create account: want AuthError(ERR_OK=0), got nameID=%d args=%s",
			replies[1].NameID, formatArgs(replies[1].Args))
	}

	// 3. CharacterListing on empty account -> ERR_EMPTY_ACCOUNT (11)
	rpc = parseReply(t, fc.call(t, cacheID, rpcCharacterListing, nil))
	if rpc.NameID != rpcCharacterError || rpc.Args[0].Int != charErrEmptyAccount {
		t.Fatalf("empty listing: want CharacterError(ERR_EMPTY_ACCOUNT=%d), got nameID=%d args=%s",
			charErrEmptyAccount, rpc.NameID, formatArgs(rpc.Args))
	}

	// 4. CreateCharacter -> CharacterInfo + CharacterError(ERR_OK)
	replies = fc.callCollect(t, cacheID, rpcCreateCharacter, []Variant{
		NewString("Alica"), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}), NewDict(map[string]Variant{})}, 2)
	if replies[0].NameID != rpcCharacterInfo {
		t.Fatalf("create character: want CharacterInfo first, got nameID=%d args=%s",
			replies[0].NameID, formatArgs(replies[0].Args))
	}
	if replies[1].NameID != rpcCharacterError || replies[1].Args[0].Int != charErrOK {
		t.Fatalf("create character: want CharacterError(ERR_OK=0), got nameID=%d args=%s",
			replies[1].NameID, formatArgs(replies[1].Args))
	}

	// 5. CharacterListing now returns the character -> CharacterInfo then ERR_OK
	replies = fc.callCollect(t, cacheID, rpcCharacterListing, nil, 2)
	if replies[0].NameID != rpcCharacterInfo {
		t.Fatalf("listing: want CharacterInfo first, got nameID=%d", replies[0].NameID)
	}
	info := replies[0].Args[0].Dict
	if info == nil || info["nickname"].Str != "Alica" {
		t.Fatalf("listing info: want nickname=Alica, got %s", formatArgs(replies[0].Args))
	}
	if replies[1].NameID != rpcCharacterError || replies[1].Args[0].Int != charErrOK {
		t.Fatalf("listing: want CharacterError(ERR_OK=0), got nameID=%d args=%s",
			replies[1].NameID, formatArgs(replies[1].Args))
	}

	// 6. ConnectCharacter("Alica") -> CharacterError(ERR_OK): peer IN_GAME
	rpc = parseReply(t, fc.call(t, cacheID, rpcConnectCharacter, []Variant{NewString("Alica")}))
	if rpc.NameID != rpcCharacterError || rpc.Args[0].Int != charErrOK {
		t.Fatalf("connect character: want CharacterError(ERR_OK=0), got nameID=%d args=%s",
			rpc.NameID, formatArgs(rpc.Args))
	}

	// 7. Duplicate account creation -> ERR_NAME_AVAILABLE (8)
	rpc = parseReply(t, fc.call(t, cacheID, rpcCreateAccount, []Variant{
		NewString("alice"), NewString("secret1"), NewString("a@b.c"), NewBool(false), NewInt(0)}))
	if rpc.NameID != rpcAuthError || rpc.Args[0].Int != authErrNameAvailable {
		t.Fatalf("duplicate account: want AuthError(ERR_NAME_AVAILABLE=%d), got nameID=%d args=%s",
			authErrNameAvailable, rpc.NameID, formatArgs(rpc.Args))
	}

	_ = info
}

// expectPush reads packets until the given server-pushed RPC arrives.
func expectPush(t *testing.T, fc *fakeClient, nameID uint16) *ParsedRPC {
	t.Helper()
	for {
		data := fc.readPacket(t)
		cmd, rpc, _, _, err := ParsePacket(data)
		if err != nil || cmd != CmdRemoteCall || rpc == nil {
			continue
		}
		if rpc.NameID == nameID {
			return rpc
		}
	}
}

// TestWorldSyncLoopback verifies the phase-2 AOI/movement flow with two
// simulated clients: enter-world pushes, online-list exchange, and movement
// broadcasts.
func TestWorldSyncLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	// setupPlayer walks one fake client through account+char+connect.
	// pushes collects any server-initiated world pushes that arrive before
	// the connect reply (they are part of the enter-world sequence).
	setupPlayer := func(fc *fakeClient, accName, charName string, pushes *[]*ParsedRPC) {
		t.Helper()
		rpc := parseReply(t, fc.call(t, 5, rpcCreateAccount, []Variant{
			NewString(accName), NewString("secret1"), NewString("a@b.c"), NewBool(false), NewInt(0)}))
		if rpc.NameID != rpcAuthError || rpc.Args[0].Int != authErrOK {
			t.Fatalf("%s create account: want AuthError(ERR_OK), got nameID=%d args=%s",
				accName, rpc.NameID, formatArgs(rpc.Args))
		}
		replies := fc.callCollect(t, 5, rpcCreateCharacter, []Variant{
			NewString(charName), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}), NewDict(map[string]Variant{})}, 2)
		if replies[1].NameID != rpcCharacterError || replies[1].Args[0].Int != charErrOK {
			t.Fatalf("%s create character: want CharacterError(ERR_OK), got nameID=%d args=%s",
				charName, replies[1].NameID, formatArgs(replies[1].Args))
		}
		pkt, err := BuildRPCPacket(5, rpcConnectCharacter, []Variant{NewString(charName)})
		if err != nil {
			t.Fatalf("build connect: %v", err)
		}
		if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
			t.Fatalf("write connect: %v", err)
		}
		var connReply *ParsedRPC
		for connReply == nil {
			data := fc.readPacket(t)
			cmd, pr, _, _, err := ParsePacket(data)
			if err != nil || cmd != CmdRemoteCall || pr == nil {
				continue
			}
			if serverPushNameIDs[pr.NameID] {
				if pushes != nil {
					*pushes = append(*pushes, pr)
				}
				continue
			}
			connReply = pr
		}
		if connReply.NameID != rpcCharacterError || connReply.Args[0].Int != charErrOK {
			t.Fatalf("%s connect: want CharacterError(ERR_OK), got nameID=%d args=%s",
				charName, connReply.NameID, formatArgs(connReply.Args))
		}
	}

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	setupPlayer(a, "wuser1", "Wone", nil)

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	var bPushes []*ParsedRPC
	setupPlayer(b, "wuser2", "Wtwo", &bPushes)

	// B's enter-world sequence (captured during connect): WarpPlayer,
	// PreloadPlayer(self), FullUpdateEntity(self),
	// PreloadPlayer+UpdateEntity for each of the other world entities
	// (A + 2 villagers + 5 field monsters = 8),
	// PreloadEntity for each seeded NPC/monster (7),
	// RefreshProgress, RefreshInventory, RefreshEquipment,
	// UpdatePublicStats, UpdatePrivateStats,
	// 3 starter DropAdded pushes (potions near spawn),
	// RefreshOnlineList.
	wantOrder := []uint16{
		rpcWarpPlayer, rpcPreloadPlayer, rpcFullUpdateEntity,
	}
	for i := 0; i < 8; i++ {
		wantOrder = append(wantOrder, rpcPreloadPlayer, rpcUpdateEntity)
	}
	for i := 0; i < 7; i++ {
		wantOrder = append(wantOrder, rpcPreloadEntity)
	}
	wantOrder = append(wantOrder,
		rpcRefreshProgress, rpcRefreshInventory,
		rpcRefreshEquipment, rpcUpdatePublicStats, rpcUpdatePrivateStats,
		rpcDropAdded, rpcDropAdded, rpcDropAdded,
		rpcRefreshOnlineList,
	)
	if len(bPushes) != len(wantOrder) {
		t.Fatalf("B enter pushes: want %d, got %d: %s", len(wantOrder), len(bPushes), pushNames(bPushes))
	}
	for i, pr := range bPushes {
		if pr.NameID != wantOrder[i] {
			t.Fatalf("B enter push %d: want nameID=%d, got %d (%s)", i, wantOrder[i], pr.NameID, pushNames(bPushes))
		}
	}
	// Find the PreloadPlayer push for A (Wone) among the three.
	var pre *ParsedRPC
	for _, pr := range bPushes {
		if pr.NameID == rpcPreloadPlayer && pr.Args[3].Str == "Wone" {
			pre = pr
			break
		}
	}
	if pre == nil {
		t.Fatalf("B preload: no PreloadPlayer for Wone, got %s", pushNames(bPushes))
	}

	// A must learn about B: PreloadPlayer + AddOnlinePlayer.
	preA := expectPush(t, a, rpcPreloadPlayer)
	if preA.Args[3].Str != "Wtwo" {
		t.Fatalf("A preload: want nick=Wtwo, got %s", formatArgs(preA.Args))
	}
	expectPush(t, a, rpcAddOnlinePlayer)

	// A walks right; B must observe the movement via UpdateEntity.
	pkt, err := BuildRPCPacket(5, rpcSetMovePos, []Variant{NewVec2(1, 0)})
	if err != nil {
		t.Fatalf("build setmovepos: %v", err)
	}
	if err := a.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write setmovepos: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	movedX := float32(0)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for movement broadcast (last x=%v)", movedX)
		}
		rpc := expectPush(t, b, rpcUpdateEntity)
		if rpc.Args[0].Int != 1 { // agentRID = char 1 (Wone)
			continue // some other entity's update
		}
		movedX = rpc.Args[2].Vec2[0]
		if movedX > 160.0 { // spawn x was 160
			break
		}
	}
	t.Logf("movement observed: Wone x=%.1f", movedX)

	// Stop A's movement so the server state settles.
	stopPkt, _ := BuildRPCPacket(5, rpcSetMovePos, []Variant{NewVec2(0, 0)})
	a.conn.WriteMessage(websocket.BinaryMessage, stopPkt)
}

// pushNames renders a push list for diagnostics.
func pushNames(pushes []*ParsedRPC) string {
	out := "["
	for i, p := range pushes {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%d", p.NameID)
	}
	return out + "]"
}

// TestGameplayLoopback verifies phase-3 chat/emote/cast notifications between
// two simulated clients in AOI.
func TestGameplayLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	// enterWorld walks a fake client through account+char+connect, collecting
	// the enter-world pushes that precede the CharacterError reply.
	enterWorld := func(fc *fakeClient, acc, chr string) {
		t.Helper()
		parseReply(t, fc.call(t, 5, rpcCreateAccount, []Variant{
			NewString(acc), NewString("secret1"), NewString("a@b.c"), NewBool(false), NewInt(0)}))
		fc.callCollect(t, 5, rpcCreateCharacter, []Variant{
			NewString(chr), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}), NewDict(map[string]Variant{})}, 2)
		pkt, err := BuildRPCPacket(5, rpcConnectCharacter, []Variant{NewString(chr)})
		if err != nil {
			t.Fatalf("build connect: %v", err)
		}
		if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
			t.Fatalf("write connect: %v", err)
		}
		var rpc *ParsedRPC
		for rpc == nil {
			data := fc.readPacket(t)
			cmd, pr, _, _, err := ParsePacket(data)
			if err != nil || cmd != CmdRemoteCall || pr == nil {
				continue
			}
			if serverPushNameIDs[pr.NameID] {
				continue
			}
			rpc = pr
		}
		if rpc.NameID != rpcCharacterError || rpc.Args[0].Int != charErrOK {
			t.Fatalf("%s connect: want CharacterError(ERR_OK), got nameID=%d args=%s",
				chr, rpc.NameID, formatArgs(rpc.Args))
		}
	}

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterWorld(a, "guser1", "Gone")

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterWorld(b, "guser2", "Gtwo")

	// A sends LOCAL chat -> both A and B receive ChatPlayer.
	pkt, _ := BuildRPCPacket(5, rpcTriggerChat, []Variant{NewString("LOCAL"), NewString("hello world")})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	chatA := expectPush(t, a, rpcChatPlayer)
	if chatA.Args[0].Str != "LOCAL" || chatA.Args[1].Str != "Gone" || chatA.Args[2].Str != "hello world" {
		t.Fatalf("A chat: got %s", formatArgs(chatA.Args))
	}
	chatB := expectPush(t, b, rpcChatPlayer)
	if chatB.Args[1].Str != "Gone" || chatB.Args[2].Str != "hello world" {
		t.Fatalf("B chat: got %s", formatArgs(chatB.Args))
	}

	// A triggers an emote -> both receive Emote.
	pkt, _ = BuildRPCPacket(5, rpcTriggerEmote, []Variant{NewInt(42)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	em := expectPush(t, a, rpcEmote)
	if em.Args[1].Int != 42 {
		t.Fatalf("A emote: got %s", formatArgs(em.Args))
	}
	emB := expectPush(t, b, rpcEmote)
	if emB.Args[1].Int != 42 {
		t.Fatalf("B emote: got %s", formatArgs(emB.Args))
	}

	// A casts a skill -> both receive Casted.
	pkt, _ = BuildRPCPacket(5, rpcCasted, []Variant{NewInt(1), NewInt(7), NewFloat(2.5)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	ca := expectPush(t, a, rpcCasted)
	if ca.Args[1].Int != 7 {
		t.Fatalf("A cast: got %s", formatArgs(ca.Args))
	}
	caB := expectPush(t, b, rpcCasted)
	if caB.Args[1].Int != 7 {
		t.Fatalf("B cast: got %s", formatArgs(caB.Args))
	}

	// B sends GLOBAL chat -> both receive ChatPlayer with GLOBAL channel.
	pkt, _ = BuildRPCPacket(5, rpcTriggerChat, []Variant{NewString("GLOBAL"), NewString("broadcast")})
	b.conn.WriteMessage(websocket.BinaryMessage, pkt)
	gb := expectPush(t, a, rpcChatPlayer)
	if gb.Args[0].Str != "GLOBAL" {
		t.Fatalf("GLOBAL chat: got %s", formatArgs(gb.Args))
	}
}

// TestNpcQuestLoopback verifies the NPC interaction flow: TriggerInteract opens
// a dialogue (ContextText + ContextChoice), TriggerChoice applies the quest
// state and pushes UpdateQuest, and TriggerNextContext advances the dialogue.
func TestNpcQuestLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")

	// enterWorld (same helper pattern as TestGameplayLoopback).
	parseReply(t, a.call(t, 5, rpcCreateAccount, []Variant{
		NewString("nuser1"), NewString("secret1"), NewString("a@b.c"), NewBool(false), NewInt(0)}))
	a.callCollect(t, 5, rpcCreateCharacter, []Variant{
		NewString("Necia"), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}), NewDict(map[string]Variant{})}, 2)
	pkt, err := BuildRPCPacket(5, rpcConnectCharacter, []Variant{NewString("Necia")})
	if err != nil {
		t.Fatalf("build connect: %v", err)
	}
	if err := a.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write connect: %v", err)
	}
	// Drain until the CharacterError reply.
	for {
		data := a.readPacket(t)
		cmd, pr, _, _, err := ParsePacket(data)
		if err != nil || cmd != CmdRemoteCall || pr == nil || serverPushNameIDs[pr.NameID] {
			continue
		}
		if pr.NameID != rpcCharacterError || pr.Args[0].Int != charErrOK {
			t.Fatalf("connect: want CharacterError(ERR_OK), got nameID=%d args=%s",
				pr.NameID, formatArgs(pr.Args))
		}
		break
	}

	// Interact with the Elder NPC (RID 1) -> ContextText greeting + ContextContinue
	// (the Elder's first step is a greeting; the choice comes on the next step).
	pkt, _ = BuildRPCPacket(5, rpcTriggerInteract, []Variant{NewInt(1)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	text := expectPush(t, a, rpcContextText)
	if text.Args[0].Str != "Elder" || text.Args[1].Str == "" {
		t.Fatalf("interact greeting: got %s", formatArgs(text.Args))
	}
	_ = expectPush(t, a, rpcContextContinue)

	// Advance to the choice step -> ContextText + ContextChoice.
	pkt, _ = BuildRPCPacket(5, rpcTriggerNextContext, nil)
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	text2 := expectPush(t, a, rpcContextText)
	if text2.Args[0].Str != "Elder" {
		t.Fatalf("choice step text: got %s", formatArgs(text2.Args))
	}
	choice := expectPush(t, a, rpcContextChoice)
	if len(choice.Args) == 0 {
		t.Fatalf("expected ContextChoice, got %s", formatArgs(choice.Args))
	}

	// Choose option 0 ("I will help.") -> ContextText echo + UpdateQuest(active).
	pkt, _ = BuildRPCPacket(5, rpcTriggerChoice, []Variant{NewInt(0)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	echo := expectPush(t, a, rpcContextText)
	if echo.Args[0].Str != "Elder" {
		t.Fatalf("choice echo: got %s", formatArgs(echo.Args))
	}
	uq := expectPush(t, a, rpcUpdateQuest)
	if uq.Args[0].Int != 1 || uq.Args[1].Int != questStateActive {
		t.Fatalf("UpdateQuest: want quest 1 active, got %s", formatArgs(uq.Args))
	}

	// TriggerSelect on the NPC -> UpdatePublicStats for the Elder (RID 1).
	// Payload: agentRID, level, health, maxHealth, hairstyle, haircolor,
	// gender, race, skintone, currentShape.
	pkt, _ = BuildRPCPacket(5, rpcTriggerSelect, []Variant{NewInt(1)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	stats := expectPush(t, a, rpcUpdatePublicStats)
	if stats.Args[0].Int != 1 || stats.Args[1].Int != 5 || stats.Args[3].Int != 100 {
		t.Fatalf("UpdatePublicStats: want Elder rid=1 level=5 maxHealth=100, got %s", formatArgs(stats.Args))
	}

	// TriggerCommand -> CommandFeedback.
	pkt, _ = BuildRPCPacket(5, rpcTriggerCommand, []Variant{NewString("dance")})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	cf := expectPush(t, a, rpcCommandFeedback)
	if cf.Args[0].Str == "" {
		t.Fatalf("CommandFeedback empty: %s", formatArgs(cf.Args))
	}
}

// TestValidateNameCJK is a unit test for the CJK-aware validateName. It checks
// that Chinese, Japanese and Korean names pass, that ASCII still passes, and
// that size/character constraints are still enforced (now counting runes).
func TestValidateNameCJK(t *testing.T) {
	cases := []struct {
		name       string
		min, max   int
		wantErr    int // 0 = valid, 1 = too small, 2 = too large, 3 = invalid char
	}{
		// Chinese (CJK Unified Ideographs U+4E00–U+9FFF).
		{"张三", charNameMinSize, charNameMaxSize, 0},
		{"玩家测试", accountNameMinSize, accountNameMaxSize, 0},
		{"李小龙", charNameMinSize, charNameMaxSize, 0},
		// Boundary: exactly min / max runes.
		{"一二", charNameMinSize, charNameMaxSize, 0},
		{"一二三四五六七八", 8, 8, 0},
		// Japanese Hiragana / Katakana.
		{"ひらがな", charNameMinSize, charNameMaxSize, 0},
		{"カタカナ", charNameMinSize, charNameMaxSize, 0},
		// Korean Hangul.
		{"김철수", charNameMinSize, charNameMaxSize, 0},
		// ASCII still valid.
		{"TestPlayer", accountNameMinSize, accountNameMaxSize, 0},
		{"Test_1", accountNameMinSize, accountNameMaxSize, 0},
		// Account name constraints (min 4).
		{"abc", accountNameMinSize, accountNameMaxSize, 1},  // too small (3 runes)
		{"ab", accountNameMinSize, accountNameMaxSize, 1},   // too small (2 runes)
		{"a", accountNameMinSize, accountNameMaxSize, 1},    // too small
		// Too large: 17 CJK runes > accountNameMaxSize (16).
		{"一二三四五六七八九十一二三四五六七", accountNameMinSize, accountNameMaxSize, 2},
		// Invalid characters: space, @, #, $, emoji, combining mark.
		{"Test@", accountNameMinSize, accountNameMaxSize, 3},
		{"Test@#$", accountNameMinSize, accountNameMaxSize, 3},
		{"Test Player", accountNameMinSize, accountNameMaxSize, 3},
		{"玩家🎮", charNameMinSize, charNameMaxSize, 3}, // emoji outside CJK ranges
	}
	for _, tc := range cases {
		got := validateName(tc.name, tc.min, tc.max)
		if got != tc.wantErr {
			t.Errorf("validateName(%q, %d, %d) = %d, want %d",
				tc.name, tc.min, tc.max, got, tc.wantErr)
		}
	}
}

// TestChineseNamesLoopback is an end-to-end loopback test over the real
// WebSocket protocol: it creates an account with a Chinese name, logs in with
// it, creates a character with a Chinese name, and verifies no ERR_NAME_VALID
// / ERR_NAME_SIZE errors occur and that names round-trip through the wire
// codec intact.
func TestChineseNamesLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	const cacheID = uint32(5)
	const path = "/root/Network"

	fc := dialFake(t, url)
	fc.announce(t, cacheID, path)

	const (
		accountName = "玩家测试" // 4 CJK runes, within [4,16]
		charName    = "张三"   // 2 CJK runes, within [2,16]
		password    = "密码1234" // 4 CJK runes + 4 digits, within [4,32]
	)

	// 1. CreateAccount with a Chinese account name -> AuthError(ERR_OK).
	// rememberMe=true, so AuthTokenResult is pushed first.
	replies := fc.callCollect(t, cacheID, rpcCreateAccount, []Variant{
		NewString(accountName), NewString(password), NewString("zh@test.com"), NewBool(true), NewInt(0)}, 2)
	if replies[0].NameID != rpcAuthTokenResult || replies[0].Args[0].Str != accountName {
		t.Fatalf("create account token: want AuthTokenResult(%q), got nameID=%d args=%s",
			accountName, replies[0].NameID, formatArgs(replies[0].Args))
	}
	if replies[1].NameID != rpcAuthError || replies[1].Args[0].Int != authErrOK {
		t.Fatalf("create account: want AuthError(ERR_OK=0), got nameID=%d args=%s",
			replies[1].NameID, formatArgs(replies[1].Args))
	}

	// 2. LoginWithPassword with the Chinese account name -> AuthError(ERR_OK).
	rpc := parseReply(t, fc.call(t, cacheID, rpcLoginWithPassword, []Variant{
		NewString(accountName), NewString(password), NewBool(false), NewInt(0)}))
	if rpc.NameID != rpcAuthError || rpc.Args[0].Int != authErrOK {
		t.Fatalf("login: want AuthError(ERR_OK=0), got nameID=%d args=%s",
			rpc.NameID, formatArgs(rpc.Args))
	}

	// 3. CreateCharacter with a Chinese character name -> CharacterInfo + CharacterError(ERR_OK).
	replies = fc.callCollect(t, cacheID, rpcCreateCharacter, []Variant{
		NewString(charName), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}), NewDict(map[string]Variant{})}, 2)
	if replies[0].NameID != rpcCharacterInfo {
		t.Fatalf("create character: want CharacterInfo first, got nameID=%d args=%s",
			replies[0].NameID, formatArgs(replies[0].Args))
	}
	info := replies[0].Args[0].Dict
	if info == nil || info["nickname"].Str != charName {
		t.Fatalf("create character info: want nickname=%q, got %s", charName, formatArgs(replies[0].Args))
	}
	if replies[1].NameID != rpcCharacterError || replies[1].Args[0].Int != charErrOK {
		t.Fatalf("create character: want CharacterError(ERR_OK=0), got nameID=%d args=%s",
			replies[1].NameID, formatArgs(replies[1].Args))
	}

	// 4. CharacterListing returns the Chinese-named character.
	replies = fc.callCollect(t, cacheID, rpcCharacterListing, nil, 2)
	if replies[0].NameID != rpcCharacterInfo || replies[0].Args[0].Dict["nickname"].Str != charName {
		t.Fatalf("listing: want CharacterInfo nickname=%q, got %s", charName, formatArgs(replies[0].Args))
	}

	// 5. ConnectCharacter with the Chinese name -> CharacterError(ERR_OK).
	rpc = parseReply(t, fc.call(t, cacheID, rpcConnectCharacter, []Variant{NewString(charName)}))
	if rpc.NameID != rpcCharacterError || rpc.Args[0].Int != charErrOK {
		t.Fatalf("connect character: want CharacterError(ERR_OK=0), got nameID=%d args=%s",
			rpc.NameID, formatArgs(rpc.Args))
	}

	// 6. Negative control: a name with invalid characters still fails.
	rpc = parseReply(t, fc.call(t, cacheID, rpcCreateCharacter, []Variant{
		NewString("玩家@测试"), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}), NewDict(map[string]Variant{})}))
	// The invalid-char name may be rejected as NAME_SIZE (validateName returns
	// non-zero) or NAME_AVAILABLE (duplicate), never OK.
	if rpc.NameID != rpcCharacterError || rpc.Args[0].Int == charErrOK {
		t.Fatalf("invalid char name: want non-OK CharacterError, got nameID=%d args=%s",
			rpc.NameID, formatArgs(rpc.Args))
	}
}
