package main

// Phase-8 loopback tests: bestiary, skills, attributes, GM commands,
// and RetrieveCharacterInformation.

import (
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// enterPhase8World walks one fake client through account+char+connect and
// returns once the CharacterError reply arrives. The account is granted
// ADMIN permission so GM commands can be tested.
func enterPhase8World(t *testing.T, fc *fakeClient, acc, chr string) {
	t.Helper()
	parseReply(t, fc.call(t, 5, rpcCreateAccount, []Variant{
		NewString(acc), NewString("secret1"), NewString("a@b.c"), NewBool(false), NewInt(0)}))
	// Grant ADMIN permission to the test account.
	lastTestServer.store.mu.Lock()
	if a := lastTestServer.store.accounts[acc]; a != nil {
		a.Permission = 3 // ADMIN
	}
	lastTestServer.store.mu.Unlock()
	fc.callCollect(t, 5, rpcCreateCharacter, []Variant{
		NewString(chr), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}), NewDict(map[string]Variant{})}, 2)
	pkt, err := BuildRPCPacket(5, rpcConnectCharacter, []Variant{NewString(chr)})
	if err != nil {
		t.Fatalf("build connect: %v", err)
	}
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write connect: %v", err)
	}
	for {
		data := fc.readPacket(t)
		cmd, pr, _, _, err := ParsePacket(data)
		if err != nil || cmd != CmdRemoteCall || pr == nil || serverPushNameIDs[pr.NameID] {
			continue
		}
		if pr.NameID != rpcCharacterError || pr.Args[0].Int != charErrOK {
			t.Fatalf("%s connect: want CharacterError(ERR_OK), got nameID=%d args=%s",
				chr, pr.NameID, formatArgs(pr.Args))
		}
		break
	}
}

// sendGM sends a TriggerCommand RPC to the server.
func sendGM(t *testing.T, fc *fakeClient, command string) {
	t.Helper()
	pkt, err := BuildRPCPacket(5, rpcTriggerCommand, []Variant{NewString(command)})
	if err != nil {
		t.Fatalf("build triggercommand: %v", err)
	}
	if err := fc.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write triggercommand: %v", err)
	}
}

// expectPushAny waits for any push within the timeout and returns it.
func expectPushAny(t *testing.T, fc *fakeClient) *ParsedRPC {
	t.Helper()
	fc.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		data := fc.readPacket(t)
		cmd, pr, _, _, err := ParsePacket(data)
		if err != nil || cmd != CmdRemoteCall || pr == nil {
			continue
		}
		if serverPushNameIDs[pr.NameID] {
			return pr
		}
	}
}

// TestBestiaryTracking verifies that killing an NPC via the GM "kill" command
// triggers an UpdateBestiary push with the correct count.
func TestBestiaryTracking(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "buser1", "Bestia")

	// Kill a Maggot via GM command. The roster has three (rids 1002..1004)
	// and world.byName returns one of them, so accept any of those rids.
	// Order of pushes: FullUpdateEntity (death), UpdateBestiary, CommandFeedback.
	sendGM(t, a, "kill Maggot")
	// Expect UpdateBestiary push first (before CommandFeedback).
	ub := expectPush(t, a, rpcUpdateBestiary)
	maggotRID := ub.Args[0].Int
	if maggotRID < 1002 || maggotRID > 1004 {
		t.Fatalf("UpdateBestiary: want a Maggot rid (1002..1004), got %d", maggotRID)
	}
	if ub.Args[1].Int != 1 { // first kill
		t.Fatalf("UpdateBestiary: want count=1, got %d", ub.Args[1].Int)
	}
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Killed Maggot") {
		t.Fatalf("kill feedback: want 'Killed Maggot', got %q", fb.Args[0].Str)
	}

	// GM bestiary resolves the nick to the same rid namespace, so this
	// increments the entry the kill just created (1 + 5 = 6).
	sendGM(t, a, "bestiary Maggot 5")
	// Order: UpdateBestiary, CommandFeedback.
	ub = expectPush(t, a, rpcUpdateBestiary)
	if ub.Args[0].Int != maggotRID {
		t.Fatalf("UpdateBestiary after GM: want mobID=%d, got %d", maggotRID, ub.Args[0].Int)
	}
	if ub.Args[1].Int != 6 {
		t.Fatalf("UpdateBestiary after GM: want count=6, got %d", ub.Args[1].Int)
	}
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Bestiary Maggot set to 6") {
		t.Fatalf("bestiary feedback: want 'set to 6', got %q", fb.Args[0].Str)
	}
}

// TestSkillProgression verifies that the GM "skill" command triggers an
// UpdateSkill push with the correct level.
func TestSkillProgression(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "suser1", "Skiller")

	// Grant skill via GM command.
	// Order of pushes: UpdateSkill, CommandFeedback.
	sendGM(t, a, "skill Fireball 3")
	us := expectPush(t, a, rpcUpdateSkill)
	wantSkill := int64(djb2String("Fireball"))
	if us.Args[0].Int != wantSkill {
		t.Fatalf("UpdateSkill: want skillID=%d, got %d", wantSkill, us.Args[0].Int)
	}
	if us.Args[1].Int != 3 {
		t.Fatalf("UpdateSkill: want level=3, got %d", us.Args[1].Int)
	}
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Skill Fireball set to level 3") {
		t.Fatalf("skill feedback: want 'set to level 3', got %q", fb.Args[0].Str)
	}

	// Verify persistence: the skill should be in the character progress.
	chr := lastTestServer.store.findCharacterByID(1)
	if chr == nil {
		t.Fatal("character not found")
	}
	if lvl := lastTestServer.progress.GetSkillLevel(1, wantSkill); lvl != 3 {
		t.Fatalf("GetSkillLevel: want 3, got %d", lvl)
	}
}

// TestGMSpawnAndKillAll verifies the GM spawn and killall commands.
func TestGMSpawnAndKillAll(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "guser1", "Gmaster")

	// Spawn 2 Maggots.
	sendGM(t, a, "spawn Maggot 2")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Spawned 2 x Maggot") {
		t.Fatalf("spawn feedback: want 'Spawned 2 x Maggot', got %q", fb.Args[0].Str)
	}

	// Kill every Maggot: 3 seeded + 2 spawned = 5.
	// Order: FullUpdateEntity + UpdateBestiary per kill, then CommandFeedback.
	sendGM(t, a, "killall Maggot")
	// Collect pushes until we see the feedback; one UpdateBestiary per kill.
	deadline := time.Now().Add(3 * time.Second)
	var kills int
	for time.Now().Before(deadline) {
		pr := expectPushAny(t, a)
		if pr.NameID == rpcUpdateBestiary {
			kills++
		}
		if pr.NameID == rpcCommandFeedback && strings.Contains(pr.Args[0].Str, "Killed 5 NPCs") {
			break
		}
	}
	if kills != 5 {
		t.Fatalf("bestiary pushes after killall: want 5, got %d", kills)
	}
}

// TestGMWarpAndJump verifies the GM warp and jump commands.
func TestGMWarpAndJump(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "wuser1", "Warper")

	// Warp to map 1 at (500, 600).
	sendGM(t, a, "warp 1 500 600")
	// Expect WarpPlayer push.
	wp := expectPush(t, a, rpcWarpPlayer)
	if wp.Args[0].Int != 1 || wp.Args[1].Vec2[0] != 500 || wp.Args[1].Vec2[1] != 600 {
		t.Fatalf("WarpPlayer: want map=1 pos=(500,600), got %s", formatArgs(wp.Args))
	}
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Warped to map 1") {
		t.Fatalf("warp feedback: want 'Warped to map 1', got %q", fb.Args[0].Str)
	}

	// Jump (random position).
	sendGM(t, a, "jump")
	wp = expectPush(t, a, rpcWarpPlayer)
	// Just verify it's a valid position.
	if wp.Args[0].Int != 1 {
		t.Fatalf("jump: want same map=1, got map=%d", wp.Args[0].Int)
	}
	t.Logf("jumped to (%.0f, %.0f)", wp.Args[1].Vec2[0], wp.Args[1].Vec2[1])
}

// TestGMGotoAndRecall verifies the GM goto and recall commands.
func TestGMGotoAndRecall(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "guser2", "Gotoer")

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterPhase8World(t, b, "guser3", "Target")

	// A gotos B.
	sendGM(t, a, "goto Target")
	// A should receive WarpPlayer to B's position.
	wp := expectPush(t, a, rpcWarpPlayer)
	t.Logf("goto warped to map=%d pos=(%.0f,%.0f)", wp.Args[0].Int, wp.Args[1].Vec2[0], wp.Args[1].Vec2[1])
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Teleported to Target") {
		t.Fatalf("goto feedback: want 'Teleported to Target', got %q", fb.Args[0].Str)
	}

	// A recalls B to A's position.
	sendGM(t, a, "recall Target")
	// B should receive WarpPlayer.
	wpB := expectPush(t, b, rpcWarpPlayer)
	t.Logf("recall warped B to map=%d pos=(%.0f,%.0f)", wpB.Args[0].Int, wpB.Args[1].Vec2[0], wpB.Args[1].Vec2[1])
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Recalled Target") {
		t.Fatalf("recall feedback: want 'Recalled Target', got %q", fb.Args[0].Str)
	}
}

// TestGMBroadcast verifies the GM broadcast and localbroadcast commands.
func TestGMBroadcast(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "buser2", "Broadcaster")

	// Global broadcast.
	sendGM(t, a, "broadcast Hello World")
	pn := expectPush(t, a, rpcPushNotification)
	if pn.Args[0].Str != "Hello World" {
		t.Fatalf("broadcast: want 'Hello World', got %q", pn.Args[0].Str)
	}
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Global broadcast sent") {
		t.Fatalf("broadcast feedback: want 'Global broadcast sent', got %q", fb.Args[0].Str)
	}

	// Local broadcast.
	sendGM(t, a, "localbroadcast Local Hello")
	pn = expectPush(t, a, rpcPushNotification)
	if pn.Args[0].Str != "Local Hello" {
		t.Fatalf("localbroadcast: want 'Local Hello', got %q", pn.Args[0].Str)
	}
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Local broadcast sent") {
		t.Fatalf("localbroadcast feedback: want 'Local broadcast sent', got %q", fb.Args[0].Str)
	}
}

// TestGMWhisperAndQuery verifies the GM whisper and query commands.
func TestGMWhisperAndQuery(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "wuser2", "Whisperer")

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterPhase8World(t, b, "wuser3", "Listener")

	// A whispers B.
	sendGM(t, a, "whisper Listener Hello there")
	// Both A and B should receive ChatPlayer.
	chatB := expectPush(t, b, rpcChatPlayer)
	if chatB.Args[2].Str != "Hello there" {
		t.Fatalf("whisper to B: want 'Hello there', got %q", chatB.Args[2].Str)
	}
	chatA := expectPush(t, a, rpcChatPlayer)
	if chatA.Args[2].Str != "Hello there" {
		t.Fatalf("whisper to A: want 'Hello there', got %q", chatA.Args[2].Str)
	}

	// A queries B.
	sendGM(t, a, "query Listener")
	// A should receive UpdatePublicStats for B.
	ups := expectPush(t, a, rpcUpdatePublicStats)
	if ups.Args[0].Int != 2 { // B is charID 2
		t.Fatalf("query: want charID=2, got %d", ups.Args[0].Int)
	}
}

// TestGMItemAndQuest verifies the GM item and quest commands.
func TestGMItemAndQuest(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "iuser1", "Itemer")

	// Give item via GM command.
	sendGM(t, a, "item Potion 5")
	ia := expectPush(t, a, rpcItemAdded)
	wantItem := int64(djb2String("Potion"))
	if ia.Args[0].Int != wantItem {
		t.Fatalf("ItemAdded: want itemID=%d, got %d", wantItem, ia.Args[0].Int)
	}
	if ia.Args[2].Int != 5 {
		t.Fatalf("ItemAdded: want count=5, got %d", ia.Args[2].Int)
	}
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Item Potion x5") {
		t.Fatalf("item feedback: want 'Item Potion x5', got %q", fb.Args[0].Str)
	}

	// Set quest via GM command.
	sendGM(t, a, "quest MyQuest 2")
	uq := expectPush(t, a, rpcUpdateQuest)
	wantQuest := int64(djb2String("MyQuest"))
	if uq.Args[0].Int != wantQuest {
		t.Fatalf("UpdateQuest: want questID=%d, got %d", wantQuest, uq.Args[0].Int)
	}
	if uq.Args[1].Int != 2 {
		t.Fatalf("UpdateQuest: want state=2, got %d", uq.Args[1].Int)
	}
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Quest MyQuest set to state 2") {
		t.Fatalf("quest feedback: want 'set to state 2', got %q", fb.Args[0].Str)
	}
}

// TestGMStatCommands verifies the GM stat/setstat/level/health commands.
func TestGMStatCommands(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "suser2", "Statner")

	// Set health to 200.
	// Order: UpdatePublicStats, CommandFeedback.
	sendGM(t, a, "health 200")
	ups := expectPush(t, a, rpcUpdatePublicStats)
	if ups.Args[0].Int != 1 { // charID 1
		t.Fatalf("UpdatePublicStats: want charID=1, got %d", ups.Args[0].Int)
	}
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Set health to 200") {
		t.Fatalf("health feedback: want 'Set health to 200', got %q", fb.Args[0].Str)
	}

	// Set level to 5.
	// Order: LevelUp, CommandFeedback.
	sendGM(t, a, "level 5")
	lu := expectPush(t, a, rpcLevelUp)
	if lu.Args[0].Int != 1 {
		t.Fatalf("LevelUp: want charID=1, got %d", lu.Args[0].Int)
	}
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Set level to 5") {
		t.Fatalf("level feedback: want 'Set level to 5', got %q", fb.Args[0].Str)
	}

	// Set GP.
	// Order: CommandFeedback only (no push for GP change).
	sendGM(t, a, "gp 1000")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Set gp to 1000") {
		t.Fatalf("gp feedback: want 'Set gp to 1000', got %q", fb.Args[0].Str)
	}
}

// TestGMKillAndRevive verifies the GM kill and revive commands.
func TestGMKillAndRevive(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "kuser1", "Killer")

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterPhase8World(t, b, "kuser2", "Victim")

	// A kills B.
	sendGM(t, a, "kill Victim")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Killed Victim") {
		t.Fatalf("kill feedback: want 'Killed Victim', got %q", fb.Args[0].Str)
	}

	// A revives B.
	sendGM(t, a, "revive Victim")
	// B should receive WarpPlayer (revive teleports to spawn).
	wp := expectPush(t, b, rpcWarpPlayer)
	t.Logf("revive warped B to map=%d pos=(%.0f,%.0f)", wp.Args[0].Int, wp.Args[1].Vec2[0], wp.Args[1].Vec2[1])
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Revived Victim") {
		t.Fatalf("revive feedback: want 'Revived Victim', got %q", fb.Args[0].Str)
	}
}

// TestRetrieveCharacterInformation verifies the RetrieveCharacterInformation
// RPC pushes stats, progress, inventory, equipment, and attributes.
func TestRetrieveCharacterInformation(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "ruser1", "Retriever")

	// Send RetrieveCharacterInformation.
	pkt, _ := BuildRPCPacket(5, rpcRetrieveCharacterInformation, nil)
	if err := a.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write retrieve: %v", err)
	}

	// Expect a sequence of pushes: UpdatePublicStats, UpdatePrivateStats,
	// RefreshProgress, RefreshInventory, RefreshEquipment, UpdateAttributes.
	var gotPublic, gotPrivate, gotProgress, gotInventory, gotEquipment, gotAttributes bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !(gotPublic && gotPrivate && gotProgress && gotInventory && gotEquipment && gotAttributes) {
		pr := expectPushAny(t, a)
		switch pr.NameID {
		case rpcUpdatePublicStats:
			gotPublic = true
		case rpcUpdatePrivateStats:
			gotPrivate = true
		case rpcRefreshProgress:
			gotProgress = true
			// Verify the dicts are valid JSON.
			if pr.Args[0].Dict == nil || pr.Args[1].Dict == nil || pr.Args[2].Dict == nil {
				t.Fatalf("RefreshProgress: expected 3 dicts, got %s", formatArgs(pr.Args))
			}
		case rpcRefreshInventory:
			gotInventory = true
		case rpcRefreshEquipment:
			gotEquipment = true
		case rpcUpdateAttributes:
			gotAttributes = true
			if len(pr.Args) != 5 {
				t.Fatalf("UpdateAttributes: want 5 args, got %d", len(pr.Args))
			}
		}
	}
	if !gotPublic {
		t.Fatal("RetrieveCharacterInformation: missing UpdatePublicStats")
	}
	if !gotPrivate {
		t.Fatal("RetrieveCharacterInformation: missing UpdatePrivateStats")
	}
	if !gotProgress {
		t.Fatal("RetrieveCharacterInformation: missing RefreshProgress")
	}
	if !gotInventory {
		t.Fatal("RetrieveCharacterInformation: missing RefreshInventory")
	}
	if !gotEquipment {
		t.Fatal("RetrieveCharacterInformation: missing RefreshEquipment")
	}
	if !gotAttributes {
		t.Fatal("RetrieveCharacterInformation: missing UpdateAttributes")
	}
}

// TestGMCommandFeedback verifies that unknown commands produce feedback.
func TestGMCommandFeedback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "fuser1", "Feedbacker")

	// Unknown command.
	sendGM(t, a, "unknowncommand foo")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Unknown command") {
		t.Fatalf("unknown command feedback: want 'Unknown command', got %q", fb.Args[0].Str)
	}

	// Usage feedback for a known command with missing args.
	sendGM(t, a, "warp")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Usage: warp") {
		t.Fatalf("usage feedback: want 'Usage: warp', got %q", fb.Args[0].Str)
	}
}

// TestGMPermissionAndBan verifies the GM permission and ban commands.
func TestGMPermissionAndBan(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "puser1", "Permitter")

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterPhase8World(t, b, "puser2", "BanTarget")

	// Set permission.
	sendGM(t, a, "permission BanTarget 2")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Set BanTarget permission to 2") {
		t.Fatalf("permission feedback: want 'Set BanTarget permission to 2', got %q", fb.Args[0].Str)
	}

	// Ban B (this will close B's connection).
	sendGM(t, a, "ban BanTarget")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "'BanTarget' banned") {
		t.Fatalf("ban feedback: want \"'BanTarget' banned\", got %q", fb.Args[0].Str)
	}

	// Unban.
	sendGM(t, a, "unban BanTarget")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "'BanTarget' unbanned") {
		t.Fatalf("unban feedback: want \"'BanTarget' unbanned\", got %q", fb.Args[0].Str)
	}
}

// TestGMDisableNpc verifies the GM disablenpc command.
func TestGMDisableNpc(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "duser1", "Disabler")

	// Disable NPC "Nina" (RID 2).
	// Order: RemoveEntity (broadcast), CommandFeedback.
	sendGM(t, a, "disablenpc Nina")
	re := expectPush(t, a, rpcRemoveEntity)
	if re.Args[0].Int != -2 { // Nina RID = 2, charID = -2
		t.Fatalf("RemoveEntity: want charID=-2, got %d", re.Args[0].Int)
	}
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Disabled NPC 'Nina'") {
		t.Fatalf("disablenpc feedback: want 'Disabled NPC', got %q", fb.Args[0].Str)
	}

	// Verify the NPC is gone from the world.
	if lastTestServer.npcByCharID(-2) == nil {
		t.Fatal("Nina NPC should still exist in npcWorld (not deleted)")
	}
	// But it should be removed from the world map.
	if e := lastTestServer.world.get(-2); e != nil {
		t.Fatal("Nina should be removed from world after disablenpc")
	}
}

// TestGMGodmodeAndHide verifies the GM godmode and hide commands.
func TestGMGodmodeAndHide(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "guser4", "Godmoder")

	// Godmode on.
	sendGM(t, a, "godmode on")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Godmode enabled") {
		t.Fatalf("godmode on feedback: want 'Godmode enabled', got %q", fb.Args[0].Str)
	}
	ups := expectPush(t, a, rpcUpdatePublicStats)
	if ups.Args[0].Int != 1 {
		t.Fatalf("godmode: want charID=1, got %d", ups.Args[0].Int)
	}

	// Godmode off.
	sendGM(t, a, "godmode off")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Godmode disabled") {
		t.Fatalf("godmode off feedback: want 'Godmode disabled', got %q", fb.Args[0].Str)
	}

	// Hide on.
	sendGM(t, a, "hide on")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Hidden") {
		t.Fatalf("hide on feedback: want 'Hidden', got %q", fb.Args[0].Str)
	}

	// Hide off.
	sendGM(t, a, "hide off")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Visible") {
		t.Fatalf("hide off feedback: want 'Visible', got %q", fb.Args[0].Str)
	}
}

// TestProgressStoreBestiary verifies the progressStore bestiary methods.
func TestProgressStoreBestiary(t *testing.T) {
	p := newProgressStore()

	// Add bestiary entries.
	if got := p.AddBestiary(1, 100, 1); got != 1 {
		t.Fatalf("AddBestiary(1,100,1)=%d want 1", got)
	}
	if got := p.AddBestiary(1, 100, 1); got != 2 {
		t.Fatalf("AddBestiary(1,100,1) again=%d want 2", got)
	}
	if got := p.AddBestiary(1, 101, 5); got != 5 {
		t.Fatalf("AddBestiary(1,101,5)=%d want 5", got)
	}
	if got := p.GetBestiary(1, 100); got != 2 {
		t.Fatalf("GetBestiary(1,100)=%d want 2", got)
	}
	if got := p.GetBestiary(1, 999); got != 0 {
		t.Fatalf("GetBestiary(1,999)=%d want 0", got)
	}

	// Add skill. AddSkill returns the *previous* level (0 for new skills).
	if got := p.AddSkill(1, 200, 3); got != 0 {
		t.Fatalf("AddSkill(1,200,3) prev=%d want 0", got)
	}
	if got := p.GetSkillLevel(1, 200); got != 3 {
		t.Fatalf("GetSkillLevel(1,200)=%d want 3", got)
	}
	// Update skill. Previous level is 3.
	if got := p.AddSkill(1, 200, 5); got != 3 {
		t.Fatalf("AddSkill(1,200,5) prev=%d want 3", got)
	}
	if got := p.GetSkillLevel(1, 200); got != 5 {
		t.Fatalf("GetSkillLevel(1,200)=%d want 5", got)
	}
	// Remove skill.
	p.RemoveSkill(1, 200)
	if got := p.GetSkillLevel(1, 200); got != 0 {
		t.Fatalf("GetSkillLevel after remove=%d want 0", got)
	}
}

// TestMaxAttributePoints verifies the attribute cap formula.
func TestMaxAttributePoints(t *testing.T) {
	tests := []struct {
		level int64
		want  int64
	}{
		{0, 4},
		{1, 5},
		{10, 14},
		{50, 54},
	}
	for _, tt := range tests {
		if got := maxAttributePoints(tt.level); got != tt.want {
			t.Errorf("maxAttributePoints(%d)=%d want %d", tt.level, got, tt.want)
		}
	}
}

// TestDjb2String pins the id namespace: item/quest/skill ids are Godot
// String.hash() values (djb2), so the client and server must agree on them.
func TestDjb2String(t *testing.T) {
	cases := []struct {
		in   string
		want uint32
	}{
		{"Tutorial", 1180319673},
		{"Elanore's Potions", 3984490417},
		{"Grain in the desert", 2893494389},
		{"Short Sword", 3056796356},
		{"Water Bottle", 3288439794},
	}
	for _, c := range cases {
		if got := djb2String(c.in); got != c.want {
			t.Errorf("djb2String(%q)=%d want %d", c.in, got, c.want)
		}
	}
	// Deterministic and collision-free for our sample.
	if djb2String("Fireball") != djb2String("Fireball") {
		t.Error("djb2String not deterministic")
	}
	if djb2String("Fireball") == djb2String("Icebolt") {
		t.Error("djb2String collision: Fireball == Icebolt")
	}
}

// TestGMInvisible verifies the GM invisible command.
func TestGMInvisible(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "iuser2", "Invisibl")

	// Invisible on.
	sendGM(t, a, "invisible on")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Invisible") {
		t.Fatalf("invisible on feedback: want 'Invisible', got %q", fb.Args[0].Str)
	}

	// Invisible off.
	sendGM(t, a, "invisible off")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Visible") {
		t.Fatalf("invisible off feedback: want 'Visible', got %q", fb.Args[0].Str)
	}
}

// TestGMRecallNpc verifies the GM recallnpc command.
func TestGMRecallNpc(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "ruser2", "Recaller")

	// Recall NPC "Elanore" to A's position.
	// Order: FullUpdateEntity (broadcast), CommandFeedback.
	sendGM(t, a, "recallnpc Elanore")
	fue := expectPush(t, a, rpcFullUpdateEntity)
	t.Logf("recallnpc: NPC charID=%d", fue.Args[0].Int)
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Recalled NPC 'Elanore'") {
		t.Fatalf("recallnpc feedback: want 'Recalled NPC Elanore', got %q", fb.Args[0].Str)
	}
}

// TestGMDeploy verifies the GM deploy command.
func TestGMDeploy(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "duser2", "Deployer")

	sendGM(t, a, "deploy")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Deploy is not available") {
		t.Fatalf("deploy feedback: want 'Deploy is not available', got %q", fb.Args[0].Str)
	}
}

// TestGMQuerySelf verifies that querying yourself is rejected.
func TestGMQuerySelf(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "quser1", "Querier")

	sendGM(t, a, "query Querier")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "You cannot query yourself") {
		t.Fatalf("query self feedback: want 'You cannot query yourself', got %q", fb.Args[0].Str)
	}
}

// TestGMWhisperSelf verifies that whispering to yourself is rejected.
func TestGMWhisperSelf(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "wuser4", "SelfWhisperer")

	sendGM(t, a, "whisper SelfWhisperer Hello")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "You cannot whisper to yourself") {
		t.Fatalf("whisper self feedback: want 'You cannot whisper to yourself', got %q", fb.Args[0].Str)
	}
}

// TestGMKick verifies the GM kick command.
func TestGMKick(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "kuser3", "Kicker")

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterPhase8World(t, b, "kuser4", "KickTarget")

	// A kicks B.
	sendGM(t, a, "kick KickTarget")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "'KickTarget' kicked") {
		t.Fatalf("kick feedback: want \"'KickTarget' kicked\", got %q", fb.Args[0].Str)
	}

	// B's connection should be closed.
	// Try reading from B — should get an error or EOF.
	b.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := b.conn.ReadMessage()
	if err == nil {
		t.Fatal("kick: B's connection should be closed")
	}
}

// TestGMExperience verifies the GM experience command.
func TestGMExperience(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "euser1", "Experiencer")

	// Grant experience.
	// Order: UpdatePrivateStats (if leveled), CommandFeedback.
	sendGM(t, a, "experience 100")
	// Private stats should be pushed (experience may cause level up).
	deadline := time.Now().Add(2 * time.Second)
	var gotFeedback bool
	for time.Now().Before(deadline) && !gotFeedback {
		pr := expectPushAny(t, a)
		if pr.NameID == rpcCommandFeedback && strings.Contains(pr.Args[0].Str, "Set experience to 100") {
			gotFeedback = true
		}
	}
	if !gotFeedback {
		t.Fatal("experience: no CommandFeedback push")
	}
}

// TestGMSpeed verifies the GM speed command.
func TestGMSpeed(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "suser3", "Speeder")

	sendGM(t, a, "speed 200")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "Speed set to 200") {
		t.Fatalf("speed feedback: want 'Speed set to 200', got %q", fb.Args[0].Str)
	}
}

// TestGMIpCheck verifies the GM ipcheck command.
func TestGMIpCheck(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "iuser3", "IpChecker")

	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")
	enterPhase8World(t, b, "iuser4", "IpTarget")

	sendGM(t, a, "ipcheck IpTarget")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "IP for 'IpTarget'") {
		t.Fatalf("ipcheck feedback: want 'IP for IpTarget', got %q", fb.Args[0].Str)
	}
}

// TestGMBanList verifies the GM banlist command.
func TestGMBanList(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "buser3", "BanLister")

	sendGM(t, a, "banlist")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "No active bans found") {
		t.Fatalf("banlist feedback: want 'No active bans found', got %q", fb.Args[0].Str)
	}

	sendGM(t, a, "ipbanlist")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "No IP bans found") {
		t.Fatalf("ipbanlist feedback: want 'No IP bans found', got %q", fb.Args[0].Str)
	}
}

// TestGMIpBan verifies the GM ipban/ipunban commands.
func TestGMIpBan(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	enterPhase8World(t, a, "iuser5", "IpBaner")

	sendGM(t, a, "ipban 192.168.1.1 spamming")
	fb := expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "IP range '192.168.1.1' banned") {
		t.Fatalf("ipban feedback: want 'IP range banned', got %q", fb.Args[0].Str)
	}

	sendGM(t, a, "ipunban 192.168.1.1")
	fb = expectPush(t, a, rpcCommandFeedback)
	if !strings.Contains(fb.Args[0].Str, "IP range '192.168.1.1' unbanned") {
		t.Fatalf("ipunban feedback: want 'IP range unbanned', got %q", fb.Args[0].Str)
	}
}

// TestProgressStoreData verifies the progressStore data model for
// bestiary/skills/quests.
func TestProgressStoreData(t *testing.T) {
	p := newProgressStore()

	// Bestiary.
	if got := p.AddBestiary(1, 100, 1); got != 1 {
		t.Fatalf("AddBestiary(1,100,1)=%d want 1", got)
	}
	if got := p.AddBestiary(1, 100, 1); got != 2 {
		t.Fatalf("AddBestiary(1,100,1) again=%d want 2", got)
	}
	if got := p.GetBestiary(1, 100); got != 2 {
		t.Fatalf("GetBestiary(1,100)=%d want 2", got)
	}
	if got := p.GetBestiary(1, 999); got != 0 {
		t.Fatalf("GetBestiary(1,999)=%d want 0", got)
	}

	// Skills.
	if got := p.AddSkill(1, 200, 3); got != 0 {
		t.Fatalf("AddSkill(1,200,3) prev=%d want 0", got)
	}
	if got := p.GetSkillLevel(1, 200); got != 3 {
		t.Fatalf("GetSkillLevel(1,200)=%d want 3", got)
	}
	if got := p.AddSkill(1, 200, 5); got != 3 {
		t.Fatalf("AddSkill(1,200,5) prev=%d want 3", got)
	}
	if got := p.GetSkillLevel(1, 200); got != 5 {
		t.Fatalf("GetSkillLevel(1,200)=%d want 5", got)
	}
	p.RemoveSkill(1, 200)
	if got := p.GetSkillLevel(1, 200); got != 0 {
		t.Fatalf("GetSkillLevel after remove=%d want 0", got)
	}

	// Quests.
	p.SetQuest(1, 300, 2)
	if got := p.GetQuest(1, 300); got != 2 {
		t.Fatalf("GetQuest(1,300)=%d want 2", got)
	}
	if got := p.GetQuest(1, 999); got != questStateInactive {
		t.Fatalf("GetQuest(1,999)=%d want %d", got, questStateInactive)
	}
}
