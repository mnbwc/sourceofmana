package main

// Phase-5 progression tests: experience/level, sit, camera, password,
// and online list request.

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestExperienceTable verifies the level-up formula against the SoM table.
// SoM levels are 0-based: Stats.level starts at 0, and
// Experience.GetNeededExperienceForNextLevel(level) indexes _experienceTable[level].
func TestExperienceTable(t *testing.T) {
	// Level 0 needs 9 XP (first entry).
	if got := neededExperience(0); got != 9 {
		t.Fatalf("neededExperience(0)=%d want 9", got)
	}
	// Level 1 needs 16 XP (second entry).
	if got := neededExperience(1); got != 16 {
		t.Fatalf("neededExperience(1)=%d want 16", got)
	}
	// Level 2 needs 25 XP (third entry).
	if got := neededExperience(2); got != 25 {
		t.Fatalf("neededExperience(2)=%d want 25", got)
	}
	// Out of range returns 0 (MAX_LEVEL_REACHED).
	if got := neededExperience(-1); got != maxLevelReached {
		t.Fatalf("neededExperience(-1)=%d want 0", got)
	}
	if got := neededExperience(999); got != maxLevelReached {
		t.Fatalf("neededExperience(999)=%d want 0", got)
	}
}

// TestGrantExperience verifies level-ups accumulate correctly.
func TestGrantExperience(t *testing.T) {
	chr := &Character{Level: 0, Experience: 0}
	// Level 0->1 needs 9 XP.
	levels := grantExperience(chr, 9)
	if levels != 1 {
		t.Fatalf("grant 9 XP: levels=%d want 1", levels)
	}
	if chr.Level != 1 || chr.Experience != 0 {
		t.Fatalf("after level up: level=%d exp=%d want 1/0", chr.Level, chr.Experience)
	}
	// Grant 20 XP: level 1 needs 16, so 20-16=4 carries over to level 2.
	levels = grantExperience(chr, 20)
	if levels != 1 {
		t.Fatalf("grant 20 XP: levels=%d want 1", levels)
	}
	if chr.Level != 2 || chr.Experience != 4 {
		t.Fatalf("after level up: level=%d exp=%d want 2/4", chr.Level, chr.Experience)
	}
	// Grant 0 XP: no change.
	levels = grantExperience(chr, 0)
	if levels != 0 || chr.Level != 2 || chr.Experience != 4 {
		t.Fatalf("grant 0 XP: levels=%d level=%d exp=%d want 0/2/4", levels, chr.Level, chr.Experience)
	}
	// Grant a huge amount: multiple level-ups.
	chr2 := &Character{Level: 0, Experience: 0}
	levels = grantExperience(chr2, 100)
	if levels != 4 { // 9+16+25+36=86, 100-86=14 < 77 (level 4)
		t.Fatalf("grant 100 XP: levels=%d want 4", levels)
	}
	if chr2.Level != 4 || chr2.Experience != 14 {
		t.Fatalf("after big grant: level=%d exp=%d want 4/14", chr2.Level, chr2.Experience)
	}
}

// TestProgressionLoopback verifies sit, camera, online list, and password
// handlers end to end.
func TestProgressionLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	fc := dialFake(t, url)
	fc.announce(t, 5, "/root/Network")
	enterPartyWorld(t, fc, "puser", "POne")

	col := startCollector(fc)

	// --- TriggerSit: IDLE -> SIT ---
	pkt, _ := BuildRPCPacket(5, rpcTriggerSit, []Variant{})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("sit: %v", err)
	}
	// No push expected (sit is a local state change); just verify no error.
	time.Sleep(100 * time.Millisecond)

	// --- RequestOnlineList ---
	pkt, _ = BuildRPCPacket(5, rpcRequestOnlineList, []Variant{})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("online list: %v", err)
	}
	online := col.recv(rpcRefreshOnlineList, 2*time.Second)
	if online == nil {
		t.Fatal("RequestOnlineList: no RefreshOnlineList push")
	}

	// --- SetViewportSize ---
	pkt, _ = BuildRPCPacket(5, rpcSetViewportSize, []Variant{NewFloat(640), NewFloat(360)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("viewport: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// --- CameraLookAt ---
	pkt, _ = BuildRPCPacket(5, rpcCameraLookAt, []Variant{NewVec2(100, 200)})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("camera look: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// --- CameraReset ---
	pkt, _ = BuildRPCPacket(5, rpcCameraReset, []Variant{})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("camera reset: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// --- ChangePassword (wrong current) ---
	pkt, _ = BuildRPCPacket(5, rpcChangePassword, []Variant{NewString("wrong"), NewString("newpass")})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("change pw: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// --- ChangePassword (correct) ---
	pkt, _ = BuildRPCPacket(5, rpcChangePassword, []Variant{NewString("secret1"), NewString("newpass")})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("change pw 2: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// --- RequestPasswordReset ---
	pkt, _ = BuildRPCPacket(5, rpcRequestPasswordReset, []Variant{NewString("puser")})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("reset req: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Retrieve the generated verification code from the server.
	lastTestServer.mu.Lock()
	rc, ok := lastTestServer.resetCodes["puser"]
	lastTestServer.mu.Unlock()
	if !ok {
		t.Fatal("no reset code generated for puser")
	}

	// --- ConfirmPasswordReset with the correct code ---
	pkt, _ = BuildRPCPacket(5, rpcConfirmPasswordReset, []Variant{NewString("puser"), NewString(rc.Code), NewString("finalpass")})
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("reset confirm: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Verify the password was changed by checking the store.
	acc := lastTestServer.store.Get("puser")
	if acc == nil {
		t.Fatal("account puser not found")
	}
	if acc.Password != "finalpass" {
		t.Fatalf("password=%q want finalpass", acc.Password)
	}
}
