package main

// Arena module tests: episode lifecycle (start/status/stop), scoring through
// the real combat path, time-limit expiry, and the EPISODE_ACTIVE rejection.
//
// Each test walks a fake Godot client into the world, then drives the bridge
// NDJSON ops (arena_start / arena_status / arena_stop / apply_action) exactly
// like an AI agent would.

import (
	"testing"
	"time"
)

// arenaPeerID is the fake client's peerID (first non-server peer).
const arenaPeerID = 2

// arenaPlayerNick is the character name used by the arena tests.
const arenaPlayerNick = "Areena"

// startArenaBridge brings up a server + bridge + one in-world player and
// returns the bridge connection plus the server (for world inspection).
func startArenaBridge(t *testing.T) (*bridgeConn, *Server, func()) {
	t.Helper()
	url, stopWS := startTestServer(t)
	addr, stopBridge := startBridgeOn(t, lastTestServer)
	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "arenauser1", arenaPlayerNick)
	return dialBridge(t, addr), lastTestServer, func() {
		stopBridge()
		stopWS()
	}
}

// aliveMaggotIDs returns the world CharIDs of living arena monsters.
func aliveMaggotIDs(s *Server) []int64 {
	var ids []int64
	for _, e := range s.world.snapshot() {
		if e.Name == "Maggot" && e.Health > 0 {
			ids = append(ids, e.CharID)
		}
	}
	return ids
}

// attackMonster fires one bridge attack at the given monster RID.
func attackMonster(t *testing.T, bc *bridgeConn, rid int64) map[string]interface{} {
	t.Helper()
	return bc.request(map[string]interface{}{
		"op": "apply_action", "peerID": arenaPeerID,
		"name": "attack", "args": map[string]interface{}{"rid": rid, "skill": flarSkillID},
	})
}

// arenaStatus calls arena_status and returns the episode object (or nil).
func arenaStatus(t *testing.T, bc *bridgeConn) map[string]interface{} {
	t.Helper()
	reply := bc.request(map[string]interface{}{"op": "arena_status", "peerID": arenaPeerID})
	if reply["ok"] != true {
		t.Fatalf("arena_status: %v", reply)
	}
	ep, _ := reply["episode"].(map[string]interface{})
	return ep
}

// eventKinds drains the peer's event buffer and returns the kinds seen.
func eventKinds(t *testing.T, bc *bridgeConn) map[string]int {
	t.Helper()
	reply := bc.request(map[string]interface{}{"op": "drain_events", "peerID": arenaPeerID})
	if reply["ok"] != true {
		t.Fatalf("drain_events: %v", reply)
	}
	kinds := map[string]int{}
	if events, ok := reply["events"].([]interface{}); ok {
		for _, ev := range events {
			em := ev.(map[string]interface{})
			kinds[em["kind"].(string)]++
		}
	}
	return kinds
}

// TestArenaStartAndStatus verifies the start lifecycle: the episode is active,
// the player is teleported to the template spawn, maxActive monsters spawn,
// and the arena_start event lands in the buffer.
func TestArenaStartAndStatus(t *testing.T) {
	bc, srv, stop := startArenaBridge(t)
	defer stop()

	// No episode yet -> null.
	if ep := arenaStatus(t, bc); ep != nil {
		t.Fatalf("status before start: want null, got %v", ep)
	}

	reply := bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "tutorial_maggot",
	})
	if reply["ok"] != true {
		t.Fatalf("arena_start: %v", reply)
	}
	if reply["template"] != "tutorial_maggot" {
		t.Fatalf("arena_start template: want tutorial_maggot, got %v", reply["template"])
	}
	episodeID, _ := reply["episodeId"].(string)
	if episodeID == "" {
		t.Fatalf("arena_start: missing episodeId: %v", reply)
	}
	if brief, _ := reply["brief"].(string); brief == "" {
		t.Fatalf("arena_start: missing brief: %v", reply)
	}

	// Status: active with the template's scoring targets.
	ep := arenaStatus(t, bc)
	if ep == nil {
		t.Fatal("status after start: episode is null")
	}
	if ep["status"] != "active" {
		t.Fatalf("episode status: want active, got %v", ep["status"])
	}
	if ep["id"] != episodeID {
		t.Fatalf("episode id: want %s, got %v", episodeID, ep["id"])
	}
	score := ep["score"].(map[string]interface{})
	if score["kills"].(float64) != 0 || score["deaths"].(float64) != 0 || score["xpGained"].(float64) != 0 {
		t.Fatalf("fresh episode score should be zero: %v", score)
	}
	tmpl := ep["template"].(map[string]interface{})
	if tmpl["killsRequired"].(float64) != 3 {
		t.Fatalf("killsRequired: want 3, got %v", tmpl["killsRequired"])
	}
	if tmpl["timeLimit"].(float64) != 60 {
		t.Fatalf("timeLimit: want 60, got %v", tmpl["timeLimit"])
	}

	// The player was teleported to the template spawn.
	player := srv.world.get(1) // first created character
	if player == nil {
		t.Fatal("player entity not found")
	}
	if player.Pos.X != 2230 || player.Pos.Y != 1250 {
		t.Fatalf("player pos: want (2230,1250), got (%.0f,%.0f)", player.Pos.X, player.Pos.Y)
	}

	// maxActive (4) monsters spawned around the player.
	if ids := aliveMaggotIDs(srv); len(ids) != 4 {
		t.Fatalf("alive monsters: want 4, got %d", len(ids))
	}
	monsters := ep["monsters"].([]interface{})
	if len(monsters) != 4 {
		t.Fatalf("episode monsters: want 4, got %d", len(monsters))
	}

	// The arena_start event was buffered.
	if kinds := eventKinds(t, bc); kinds["arena_start"] != 1 {
		t.Fatalf("events: want 1 arena_start, got %v", kinds)
	}
}

// TestArenaKillToFinish attacks monsters through the real combat path until
// killsRequired is met, verifying the episode finishes with the right events.
func TestArenaKillToFinish(t *testing.T) {
	bc, srv, stop := startArenaBridge(t)
	defer stop()

	reply := bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "tutorial_maggot",
	})
	if reply["ok"] != true {
		t.Fatalf("arena_start: %v", reply)
	}

	// Kill three distinct monsters (Flar one-shots a 30 HP Maggot).
	ids := aliveMaggotIDs(srv)
	if len(ids) < 3 {
		t.Fatalf("want at least 3 monsters, got %d", len(ids))
	}
	for i := 0; i < 3; i++ {
		atk := attackMonster(t, bc, ids[i])
		if atk["ok"] != true {
			t.Fatalf("attack %d: %v", i, atk)
		}
		ep := arenaStatus(t, bc)
		if ep == nil {
			t.Fatalf("attack %d: episode disappeared", i)
		}
		wantKills := float64(i + 1)
		if got := ep["score"].(map[string]interface{})["kills"].(float64); got != wantKills {
			t.Fatalf("attack %d: want kills=%.0f, got %.0f", i, wantKills, got)
		}
		if i < 2 && ep["status"] != "active" {
			t.Fatalf("attack %d: episode should still be active, got %v", i, ep["status"])
		}
	}

	// The third kill finishes the episode.
	ep := arenaStatus(t, bc)
	if ep["status"] != "finished" {
		t.Fatalf("episode status: want finished, got %v", ep["status"])
	}
	if ep["endReason"] != "kills_target_met" {
		t.Fatalf("endReason: want kills_target_met, got %v", ep["endReason"])
	}
	score := ep["score"].(map[string]interface{})
	if score["kills"].(float64) != 3 {
		t.Fatalf("final kills: want 3, got %v", score["kills"])
	}
	if score["xpGained"].(float64) <= 0 {
		t.Fatalf("xpGained should be positive after kills, got %v", score["xpGained"])
	}

	// All monsters were removed when the episode ended.
	if ids := aliveMaggotIDs(srv); len(ids) != 0 {
		t.Fatalf("monsters after finish: want 0, got %d", len(ids))
	}

	// Events: start, 3 kills, 3 arena scoring events, end.
	kinds := eventKinds(t, bc)
	if kinds["arena_start"] != 1 {
		t.Fatalf("events: want 1 arena_start, got %v", kinds)
	}
	if kinds["kill"] != 3 {
		t.Fatalf("events: want 3 kill, got %v", kinds)
	}
	if kinds["arena"] != 3 {
		t.Fatalf("events: want 3 arena scoring events, got %v", kinds)
	}
	if kinds["arena_end"] != 1 {
		t.Fatalf("events: want 1 arena_end, got %v", kinds)
	}

	// A finished episode can be restarted.
	reply = bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "tutorial_maggot",
	})
	if reply["ok"] != true {
		t.Fatalf("restart after finish: %v", reply)
	}
}

// TestArenaStop verifies the explicit stop path: status stopped, monsters
// removed, arena_end event, and NO_ACTIVE_EPISODE on a second stop.
func TestArenaStop(t *testing.T) {
	bc, srv, stop := startArenaBridge(t)
	defer stop()

	// Stopping with no episode at all.
	reply := bc.request(map[string]interface{}{"op": "arena_stop", "peerID": arenaPeerID})
	if reply["ok"] != false || reply["code"] != "NO_ACTIVE_EPISODE" {
		t.Fatalf("stop without episode: %v", reply)
	}

	reply = bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "grind_maggot",
	})
	if reply["ok"] != true {
		t.Fatalf("arena_start: %v", reply)
	}
	if ids := aliveMaggotIDs(srv); len(ids) != 6 { // grind maxActive = 6
		t.Fatalf("grind monsters: want 6, got %d", len(ids))
	}

	reply = bc.request(map[string]interface{}{"op": "arena_stop", "peerID": arenaPeerID})
	if reply["ok"] != true {
		t.Fatalf("arena_stop: %v", reply)
	}
	if reply["status"] != "stopped" {
		t.Fatalf("stop status: want stopped, got %v", reply["status"])
	}
	episodeID, _ := reply["episodeId"].(string)
	if episodeID == "" {
		t.Fatalf("arena_stop: missing episodeId: %v", reply)
	}

	ep := arenaStatus(t, bc)
	if ep == nil || ep["status"] != "stopped" {
		t.Fatalf("status after stop: want stopped, got %v", ep)
	}
	if ep["endReason"] != "stopped" {
		t.Fatalf("endReason: want stopped, got %v", ep["endReason"])
	}
	if ids := aliveMaggotIDs(srv); len(ids) != 0 {
		t.Fatalf("monsters after stop: want 0, got %d", len(ids))
	}

	// Second stop is rejected.
	reply = bc.request(map[string]interface{}{"op": "arena_stop", "peerID": arenaPeerID})
	if reply["ok"] != false || reply["code"] != "NO_ACTIVE_EPISODE" {
		t.Fatalf("second stop: %v", reply)
	}

	kinds := eventKinds(t, bc)
	if kinds["arena_start"] != 1 || kinds["arena_end"] != 1 {
		t.Fatalf("events: want arena_start+arena_end, got %v", kinds)
	}
}

// TestArenaTimeLimitExpiry starts a short-limit episode and verifies the
// background ticker expires it and removes the monsters.
func TestArenaTimeLimitExpiry(t *testing.T) {
	bc, srv, stop := startArenaBridge(t)
	defer stop()

	// Install a short time-limit template and speed the ticker up.
	srv.bridge.arena.registerTemplate(&arenaTemplate{
		Key: "test_quick", Label: "test", Brief: "quick expiry test",
		SpawnPos: Vec2{X: 2230, Y: 1250}, MonsterNick: "Maggot",
		KillsRequired: 99, MaxActive: 2, TimeLimitSecs: 1,
		MonsterHealth: 30, MonsterLevel: 1,
	})
	srv.bridge.arena.tickEvery = 100 * time.Millisecond

	reply := bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "test_quick",
	})
	if reply["ok"] != true {
		t.Fatalf("arena_start: %v", reply)
	}
	if ids := aliveMaggotIDs(srv); len(ids) != 2 {
		t.Fatalf("quick monsters: want 2, got %d", len(ids))
	}

	// The ticker (every 100ms) should expire the 1s episode quickly.
	deadline := time.Now().Add(5 * time.Second)
	var ep map[string]interface{}
	for time.Now().Before(deadline) {
		ep = arenaStatus(t, bc)
		if ep != nil && ep["status"] == "expired" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if ep == nil || ep["status"] != "expired" {
		t.Fatalf("episode should have expired, got %v", ep)
	}
	if ep["endReason"] != "time_limit" {
		t.Fatalf("endReason: want time_limit, got %v", ep["endReason"])
	}
	if ids := aliveMaggotIDs(srv); len(ids) != 0 {
		t.Fatalf("monsters after expiry: want 0, got %d", len(ids))
	}
	kinds := eventKinds(t, bc)
	if kinds["arena_end"] != 1 {
		t.Fatalf("events: want 1 arena_end, got %v", kinds)
	}
}

// TestArenaStartTwiceRejected verifies the EPISODE_ACTIVE guard and the
// UNKNOWN_TEMPLATE validation.
func TestArenaStartTwiceRejected(t *testing.T) {
	bc, _, stop := startArenaBridge(t)
	defer stop()

	reply := bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "tutorial_maggot",
	})
	if reply["ok"] != true {
		t.Fatalf("first arena_start: %v", reply)
	}

	// Second start while the first is active.
	reply = bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "tutorial_maggot",
	})
	if reply["ok"] != false || reply["code"] != "EPISODE_ACTIVE" {
		t.Fatalf("second arena_start: %v", reply)
	}

	// Stop the active episode, then an unknown template is rejected.
	bc.request(map[string]interface{}{"op": "arena_stop", "peerID": arenaPeerID})
	reply = bc.request(map[string]interface{}{
		"op": "arena_start", "peerID": arenaPeerID,
		"nick": arenaPlayerNick, "template": "does_not_exist",
	})
	if reply["ok"] != false || reply["code"] != "UNKNOWN_TEMPLATE" {
		t.Fatalf("unknown template: %v", reply)
	}

	// Missing parameters.
	reply = bc.request(map[string]interface{}{"op": "arena_start", "peerID": arenaPeerID})
	if reply["ok"] != false || reply["code"] != "INVALID_PARAMS" {
		t.Fatalf("missing nick/template: %v", reply)
	}
	reply = bc.request(map[string]interface{}{"op": "arena_status"})
	if reply["ok"] != false || reply["code"] != "INVALID_PARAMS" {
		t.Fatalf("missing peerID: %v", reply)
	}
}
