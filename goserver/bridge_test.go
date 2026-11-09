package main

// AgentBridge loopback test: connects a raw TCP client to the bridge, walks a
// fake Godot client through login/enter-world, then exercises the NDJSON ops
// (ping, list_players, snapshot, apply_action, drain_events, latest_verdict,
// spectate, close_session) and verifies the JSON replies.

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// bridgeConn is a minimal NDJSON TCP client for the bridge.
type bridgeConn struct {
	t    *testing.T
	conn net.Conn
	rd   *bufio.Reader
}

// dialBridge connects to the bridge at addr.
func dialBridge(t *testing.T, addr string) *bridgeConn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial bridge %s: %v", addr, err)
	}
	return &bridgeConn{t: t, conn: conn, rd: bufio.NewReader(conn)}
}

// request sends one JSON request line and reads one JSON reply line.
func (bc *bridgeConn) request(req map[string]interface{}) map[string]interface{} {
	bc.t.Helper()
	buf, err := json.Marshal(req)
	if err != nil {
		bc.t.Fatalf("marshal request: %v", err)
	}
	if _, err := bc.conn.Write([]byte(string(buf) + "\n")); err != nil {
		bc.t.Fatalf("write request: %v", err)
	}
	line, err := bc.rd.ReadString('\n')
	if err != nil {
		bc.t.Fatalf("read reply: %v", err)
	}
	var reply map[string]interface{}
	if err := json.Unmarshal([]byte(line), &reply); err != nil {
		bc.t.Fatalf("unmarshal reply %q: %v", line, err)
	}
	return reply
}

// requestRaw sends a raw JSON string (for malformed-input tests).
func (bc *bridgeConn) requestRaw(line string) map[string]interface{} {
	bc.t.Helper()
	if _, err := bc.conn.Write([]byte(line + "\n")); err != nil {
		bc.t.Fatalf("write raw request: %v", err)
	}
	reply, err := bc.rd.ReadString('\n')
	if err != nil {
		bc.t.Fatalf("read reply: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(reply), &out); err != nil {
		bc.t.Fatalf("unmarshal reply %q: %v", reply, err)
	}
	return out
}

func (bc *bridgeConn) close() { bc.conn.Close() }

// startBridgeOn starts the bridge on an ephemeral port for an existing
// server and returns the bridge addr + cleanup. The bridge must wrap the same
// Server the game clients are connected to.
func startBridgeOn(t *testing.T, s *Server) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	// Reserve the port, then close so StartBridge can bind it. This avoids
	// racing two listeners on the same ephemeral port.
	addr := ln.Addr().String()
	ln.Close()
	// Give the kernel a moment to release the port.
	time.Sleep(20 * time.Millisecond)
	b, err := StartBridge(s, addr)
	if err != nil {
		t.Fatalf("StartBridge %s: %v", addr, err)
	}
	return addr, func() { b.Stop() }
}

// TestBridgePing verifies the ping op returns a frame counter and the SoM
// protocol version.
func TestBridgePing(t *testing.T) {
	_, stopWS := startTestServer(t)
	defer stopWS()
	addr, stop := startBridgeOn(t, lastTestServer)
	defer stop()

	bc := dialBridge(t, addr)
	defer bc.close()

	reply := bc.request(map[string]interface{}{"op": "ping"})
	if reply["ok"] != true {
		t.Fatalf("ping: want ok=true, got %v", reply)
	}
	if _, ok := reply["frame"].(float64); !ok {
		t.Fatalf("ping: frame missing/not a number: %v", reply)
	}
	if pv, ok := reply["protocolVersion"].(float64); !ok || int64(pv) != int64(bridgeProtocolVersion) {
		t.Fatalf("ping: protocolVersion=%v want %d", reply["protocolVersion"], bridgeProtocolVersion)
	}
}

// TestBridgeUnknownOp verifies the INVALID_PARAMS reply for an unknown op.
func TestBridgeUnknownOp(t *testing.T) {
	_, stopWS := startTestServer(t)
	defer stopWS()
	addr, stop := startBridgeOn(t, lastTestServer)
	defer stop()

	bc := dialBridge(t, addr)
	defer bc.close()

	reply := bc.request(map[string]interface{}{"op": "bogus_op"})
	if reply["ok"] != false {
		t.Fatalf("unknown op: want ok=false, got %v", reply)
	}
	if reply["code"] != "INVALID_PARAMS" {
		t.Fatalf("unknown op: want code=INVALID_PARAMS, got %v", reply)
	}
}

// TestBridgeMissingPeer verifies peerID validation on snapshot.
func TestBridgeMissingPeer(t *testing.T) {
	_, stopWS := startTestServer(t)
	defer stopWS()
	addr, stop := startBridgeOn(t, lastTestServer)
	defer stop()

	bc := dialBridge(t, addr)
	defer bc.close()

	reply := bc.request(map[string]interface{}{"op": "snapshot"})
	if reply["ok"] != false || reply["code"] != "INVALID_PARAMS" {
		t.Fatalf("snapshot without peerID: got %v", reply)
	}
}

// TestBridgeFullFlow is the main integration test: it walks a fake Godot
// client through login/enter-world, then exercises the bridge ops against the
// resulting in-game entity.
func TestBridgeFullFlow(t *testing.T) {
	url, stopWS := startTestServer(t)
	defer stopWS()
	addr, stopBridge := startBridgeOn(t, lastTestServer)
	defer stopBridge()

	// Bring one fake client into the world.
	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "buser1", "Bridger")

	// The fake client's peerID is 2 (first non-server peer).
	peerID := 2

	bc := dialBridge(t, addr)
	defer bc.close()

	// --- ping ---
	ping := bc.request(map[string]interface{}{"op": "ping"})
	if ping["ok"] != true {
		t.Fatalf("ping: %v", ping)
	}

	// --- list_players ---
	lp := bc.request(map[string]interface{}{"op": "list_players"})
	if lp["ok"] != true {
		t.Fatalf("list_players: %v", lp)
	}
	players, ok := lp["players"].([]interface{})
	if !ok || len(players) == 0 {
		t.Fatalf("list_players: no players: %v", lp)
	}
	found := false
	for _, p := range players {
		pm := p.(map[string]interface{})
		if int(pm["peerID"].(float64)) == peerID {
			found = true
			if pm["nick"] != "Bridger" {
				t.Fatalf("list_players: want nick Bridger, got %v", pm["nick"])
			}
		}
	}
	if !found {
		t.Fatalf("list_players: peer %d not listed: %v", peerID, players)
	}

	// --- snapshot ---
	snapReply := bc.request(map[string]interface{}{"op": "snapshot", "peerID": peerID})
	if snapReply["ok"] != true {
		t.Fatalf("snapshot: %v", snapReply)
	}
	snap := snapReply["snapshot"].(map[string]interface{})
	player := snap["player"].(map[string]interface{})
	if int(player["peerID"].(float64)) != peerID {
		t.Fatalf("snapshot player peerID: %v", player["peerID"])
	}
	if player["nick"] != "Bridger" {
		t.Fatalf("snapshot player nick: %v", player["nick"])
	}
	if player["alive"] != true {
		t.Fatalf("snapshot player should be alive: %v", player)
	}
	health := player["health"].(map[string]interface{})
	if health["current"].(float64) <= 0 || health["max"].(float64) <= 0 {
		t.Fatalf("snapshot health: %v", health)
	}
	if _, ok := snap["nearby"].([]interface{}); !ok {
		t.Fatalf("snapshot nearby missing: %v", snap)
	}

	// --- apply_action: navigate ---
	nav := bc.request(map[string]interface{}{
		"op": "apply_action", "peerID": peerID,
		"name": "navigate", "args": map[string]interface{}{"x": 200, "y": 200},
	})
	if nav["ok"] != true || nav["verdict"] != "APPLIED" || nav["code"] != "APPLIED" {
		t.Fatalf("navigate: %v", nav)
	}

	// --- apply_action: say ---
	say := bc.request(map[string]interface{}{
		"op": "apply_action", "peerID": peerID,
		"name": "say", "args": map[string]interface{}{"text": "hello bridge"},
	})
	if say["ok"] != true || say["verdict"] != "APPLIED" {
		t.Fatalf("say: %v", say)
	}

	// --- apply_action: unknown action ---
	bad := bc.request(map[string]interface{}{
		"op": "apply_action", "peerID": peerID,
		"name": "teleport", "args": map[string]interface{}{},
	})
	if bad["ok"] != false || bad["code"] != "INVALID_PARAMS" {
		t.Fatalf("unknown action: %v", bad)
	}

	// --- drain_events: should contain the chat event from say ---
	drain := bc.request(map[string]interface{}{"op": "drain_events", "peerID": peerID})
	if drain["ok"] != true {
		t.Fatalf("drain_events: %v", drain)
	}
	events, ok := drain["events"].([]interface{})
	if !ok {
		t.Fatalf("drain_events: events missing: %v", drain)
	}
	kindSeen := map[string]bool{}
	for _, ev := range events {
		em := ev.(map[string]interface{})
		kindSeen[em["kind"].(string)] = true
		if _, ok := em["seq"].(float64); !ok {
			t.Fatalf("drain_events: event missing seq: %v", em)
		}
	}
	if !kindSeen["chat"] {
		t.Fatalf("drain_events: expected a chat event, got %v", kindSeen)
	}

	// --- drain_events again: buffer should be empty ---
	drain2 := bc.request(map[string]interface{}{"op": "drain_events", "peerID": peerID})
	events2 := drain2["events"].([]interface{})
	if len(events2) != 0 {
		t.Fatalf("second drain: want empty, got %v", events2)
	}

	// --- latest_verdict: navigate should resolve to APPLIED once the entity
	// has moved (the world tick processes Target at 10 Hz).
	deadline := time.Now().Add(2 * time.Second)
	var verdict string
	for time.Now().Before(deadline) {
		lv := bc.request(map[string]interface{}{"op": "latest_verdict", "peerID": peerID})
		verdict = lv["verdict"].(string)
		if verdict == "APPLIED" || verdict == "REJECTED" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if verdict != "APPLIED" {
		t.Fatalf("latest_verdict after navigate: want APPLIED, got %q", verdict)
	}

	// --- spectate ---
	spec := bc.request(map[string]interface{}{"op": "spectate", "peerID": peerID})
	if spec["ok"] != true || spec["readOnly"] != true {
		t.Fatalf("spectate: %v", spec)
	}
	if int(spec["spectators"].(float64)) != 1 {
		t.Fatalf("spectate: want 1 spectator, got %v", spec["spectators"])
	}
	if _, ok := spec["snapshot"]; !ok {
		t.Fatalf("spectate: snapshot missing: %v", spec)
	}

	// --- spectate_feed ---
	feed := bc.request(map[string]interface{}{"op": "spectate_feed", "peerID": peerID})
	if feed["ok"] != true || feed["readOnly"] != true {
		t.Fatalf("spectate_feed: %v", feed)
	}
	if _, ok := feed["events"].([]interface{}); !ok {
		t.Fatalf("spectate_feed: events missing: %v", feed)
	}
	if _, ok := feed["snapshot"]; !ok {
		t.Fatalf("spectate_feed: snapshot missing: %v", feed)
	}

	// --- spectate_stop ---
	stopSpec := bc.request(map[string]interface{}{"op": "spectate_stop", "peerID": peerID})
	if stopSpec["ok"] != true || stopSpec["code"] != "SPECTATE_LEFT" {
		t.Fatalf("spectate_stop: %v", stopSpec)
	}
	if int(stopSpec["spectators"].(float64)) != 0 {
		t.Fatalf("spectate_stop: want 0 spectators, got %v", stopSpec["spectators"])
	}

	// --- close_session ---
	closeReply := bc.request(map[string]interface{}{"op": "close_session", "peerID": peerID})
	if closeReply["ok"] != true || closeReply["code"] != "CLOSED" {
		t.Fatalf("close_session: %v", closeReply)
	}
}

// TestBridgeAttackFlow verifies the attack action damages the target and
// produces damaged/kill events plus an APPLIED verdict.
func TestBridgeAttackFlow(t *testing.T) {
	url, stopWS := startTestServer(t)
	defer stopWS()
	addr, stopBridge := startBridgeOn(t, lastTestServer)
	defer stopBridge()

	// Two clients: attacker (peer 2, char 1) and target (peer 3, char 2).
	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPartyWorld(t, a, "auser1", "Attacker")
	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterPartyWorld(t, b, "auser2", "Targeter")

	attackerPeer := 2
	targetRID := int64(2)

	bc := dialBridge(t, addr)
	defer bc.close()

	// Snapshot the target's starting health.
	snapReply := bc.request(map[string]interface{}{"op": "snapshot", "peerID": attackerPeer})
	snap := snapReply["snapshot"].(map[string]interface{})
	nearby := snap["nearby"].([]interface{})
	var targetHealth float64 = -1
	for _, n := range nearby {
		nm := n.(map[string]interface{})
		if int64(nm["rid"].(float64)) == targetRID {
			targetHealth = nm["health"].(map[string]interface{})["current"].(float64)
		}
	}
	if targetHealth < 0 {
		t.Fatalf("target rid %d not in attacker's nearby: %v", targetRID, nearby)
	}

	// Attack the target with a known skill (Flar).
	atk := bc.request(map[string]interface{}{
		"op": "apply_action", "peerID": attackerPeer,
		"name": "attack", "args": map[string]interface{}{"rid": targetRID, "skill": flarSkillID},
	})
	if atk["ok"] != true {
		t.Fatalf("attack: %v", atk)
	}

	// The target's health should have dropped.
	time.Sleep(150 * time.Millisecond)
	snap2 := bc.request(map[string]interface{}{"op": "snapshot", "peerID": attackerPeer})
	nearby2 := snap2["snapshot"].(map[string]interface{})["nearby"].([]interface{})
	var newHealth float64 = -1
	for _, n := range nearby2 {
		nm := n.(map[string]interface{})
		if int64(nm["rid"].(float64)) == targetRID {
			newHealth = nm["health"].(map[string]interface{})["current"].(float64)
		}
	}
	if newHealth < 0 {
		t.Fatalf("target rid %d disappeared from nearby after attack", targetRID)
	}
	if newHealth >= targetHealth {
		t.Fatalf("attack did not reduce target health: before=%.0f after=%.0f", targetHealth, newHealth)
	}

	// drain_events should carry a damaged event.
	drain := bc.request(map[string]interface{}{"op": "drain_events", "peerID": attackerPeer})
	events := drain["events"].([]interface{})
	hasDamaged := false
	for _, ev := range events {
		em := ev.(map[string]interface{})
		if em["kind"] == "damaged" {
			hasDamaged = true
		}
	}
	if !hasDamaged {
		t.Fatalf("drain_events: want a damaged event, got %v", events)
	}

	// latest_verdict should resolve to APPLIED (target health decreased).
	lv := bc.request(map[string]interface{}{"op": "latest_verdict", "peerID": attackerPeer})
	if lv["verdict"] != "APPLIED" {
		t.Fatalf("latest_verdict after attack: want APPLIED, got %v", lv)
	}
}

// TestBridgeMalformedJSON verifies the bridge replies cleanly to bad input.
func TestBridgeMalformedJSON(t *testing.T) {
	_, stopWS := startTestServer(t)
	defer stopWS()
	addr, stop := startBridgeOn(t, lastTestServer)
	defer stop()

	bc := dialBridge(t, addr)
	defer bc.close()

	reply := bc.requestRaw("this is not json")
	if reply["ok"] != false || reply["code"] != "INVALID_PARAMS" {
		t.Fatalf("malformed JSON: %v", reply)
	}

	reply = bc.requestRaw(`{"op":""}`)
	if reply["ok"] != false || reply["code"] != "INVALID_PARAMS" {
		t.Fatalf("empty op: %v", reply)
	}
}

// TestBridgeMultipleClients verifies that two TCP clients can talk to the
// bridge concurrently without interleaved replies.
func TestBridgeMultipleClients(t *testing.T) {
	_, stopWS := startTestServer(t)
	defer stopWS()
	addr, stop := startBridgeOn(t, lastTestServer)
	defer stop()

	bc1 := dialBridge(t, addr)
	defer bc1.close()
	bc2 := dialBridge(t, addr)
	defer bc2.close()

	r1 := bc1.request(map[string]interface{}{"op": "ping"})
	r2 := bc2.request(map[string]interface{}{"op": "ping"})
	if r1["ok"] != true || r2["ok"] != true {
		t.Fatalf("concurrent ping: r1=%v r2=%v", r1, r2)
	}
	// Both replies must be valid single-line JSON (no interleaving).
	if _, ok := r1["protocolVersion"]; !ok {
		t.Fatalf("r1 missing protocolVersion: %v", r1)
	}
	if _, ok := r2["protocolVersion"]; !ok {
		t.Fatalf("r2 missing protocolVersion: %v", r2)
	}
}

// TestBridgeFrameAdvances verifies the frame counter increments over time.
func TestBridgeFrameAdvances(t *testing.T) {
	_, stopWS := startTestServer(t)
	defer stopWS()
	addr, stop := startBridgeOn(t, lastTestServer)
	defer stop()

	bc := dialBridge(t, addr)
	defer bc.close()

	f1 := int64(bc.request(map[string]interface{}{"op": "ping"})["frame"].(float64))
	time.Sleep(350 * time.Millisecond)
	f2 := int64(bc.request(map[string]interface{}{"op": "ping"})["frame"].(float64))
	if f2 <= f1 {
		t.Fatalf("frame did not advance: f1=%d f2=%d", f1, f2)
	}
}
