package main

// Phase-3 party loopback tests: invite -> accept -> PartyUpdate -> PARTY chat
// -> promote -> leave, plus a unit test of the PartyService data model.

import (
	"encoding/json"
	"testing"

	"github.com/gorilla/websocket"
)

// enterPartyWorld walks one fake client through account+char+connect and
// returns once the CharacterError reply arrives.
func enterPartyWorld(t *testing.T, fc *fakeClient, acc, chr string) {
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

// TestPartyService is a unit test of the PartyService data model (mirrors
// PartyService.gd semantics).
func TestPartyService(t *testing.T) {
	ps := newPartyService()

	// Create a party led by rid 10.
	pid := ps.createParty(10)
	if pid == 0 {
		t.Fatal("createParty failed")
	}
	if ps.createParty(10) != 0 {
		t.Fatal("createParty on an existing leader should fail")
	}
	if ps.partyID(10) != pid {
		t.Fatalf("partyID(10)=%d want %d", ps.partyID(10), pid)
	}
	if ps.leader(pid) != 10 {
		t.Fatalf("leader=%d want 10", ps.leader(pid))
	}

	// Add rid 11 to the party.
	if !ps.join(pid, 11) {
		t.Fatal("join(11) failed")
	}
	if ps.join(pid, 11) {
		t.Fatal("double join should fail")
	}
	members := ps.members(pid)
	if len(members) != 2 {
		t.Fatalf("members=%v want 2", members)
	}

	// Invites: rid 12 invited by 10. Re-invite overwrites (mirrors SoM:
	// invites[targetRID] = inviterRID) and still returns true.
	if !ps.addInvite(12, 10) {
		t.Fatal("addInvite failed")
	}
	if !ps.addInvite(12, 20) {
		t.Fatal("re-invite should overwrite and return true")
	}
	if ps.popInvite(12) != 20 {
		t.Fatalf("popInvite=%d want 20 (the latest inviter)", ps.popInvite(12))
	}
	if ps.popInvite(12) != 0 {
		t.Fatal("popInvite on consumed invite should be 0")
	}
	ps.join(pid, 12)
	if ps.addInvite(12, 10) {
		t.Fatal("invite to a party member should fail")
	}

	// Promote 11 to leader.
	if !ps.promote(pid, 10, 11) {
		t.Fatal("promote failed")
	}
	if ps.leader(pid) != 11 {
		t.Fatalf("leader=%d want 11", ps.leader(pid))
	}
	if ps.promote(pid, 10, 12) {
		t.Fatal("non-leader promote should fail")
	}

	// Kick 12 by the leader.
	if !ps.kick(pid, 11, 12) {
		t.Fatal("kick failed")
	}
	if ps.partyID(12) != 0 {
		t.Fatal("kicked member should have no party")
	}

	// Leader 11 leaves -> leadership transfers to 10.
	if ps.leave(11) != pid {
		t.Fatal("leader leave should return pid")
	}
	if ps.leader(pid) != 10 {
		t.Fatalf("leader after transfer=%d want 10", ps.leader(pid))
	}

	// Last member leaves -> party disbands.
	if ps.leave(10) != pid {
		t.Fatal("last member leave should return pid")
	}
	if len(ps.parties) != 0 {
		t.Fatal("empty party should be disbanded")
	}
	if ps.partyID(10) != 0 {
		t.Fatal("disbanded member should have no party")
	}
}

// TestPartyLoopback verifies the full party flow between two simulated clients:
// invite -> PartyInvite push -> accept -> PartyUpdate to both -> PARTY chat
// delivered to the other member -> promote -> PartyUpdate leader change ->
// leave -> PartyUpdate with the remaining member.
func TestPartyLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")

	enterPartyWorld(t, a, "puser1", "Pelia")
	enterPartyWorld(t, b, "puser2", "Popp")

	// A invites B -> B receives PartyInvite(Pelia).
	pkt, _ := BuildRPCPacket(5, rpcTriggerPartyInvite, []Variant{NewString("Popp")})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	inv := expectPush(t, b, rpcPartyInvite)
	if inv.Args[0].Str != "Pelia" {
		t.Fatalf("PartyInvite: want inviter Pelia, got %s", formatArgs(inv.Args))
	}

	// B accepts -> both receive PartyUpdate with 2 members.
	pkt, _ = BuildRPCPacket(5, rpcTriggerPartyAccept, nil)
	b.conn.WriteMessage(websocket.BinaryMessage, pkt)
	updA := expectPush(t, a, rpcPartyUpdate)
	updB := expectPush(t, b, rpcPartyUpdate)
	// members is a JSON array string; leader is one of the two charIDs.
	if !isJSONArray(updA.Args[0].Str) || !isJSONArray(updB.Args[0].Str) {
		t.Fatalf("PartyUpdate members not JSON arrays: A=%s B=%s",
			updA.Args[0].Str, updB.Args[0].Str)
	}
	if updA.Args[1].Int == 0 || updB.Args[1].Int == 0 {
		t.Fatalf("PartyUpdate leader should be non-zero: A=%s B=%s",
			formatArgs(updA.Args), formatArgs(updB.Args))
	}
	if updA.Args[1].Int != updB.Args[1].Int {
		t.Fatalf("leader mismatch: A=%d B=%d", updA.Args[1].Int, updB.Args[1].Int)
	}

	// A sends PARTY chat -> B receives ChatPlayer(PARTY).
	pkt, _ = BuildRPCPacket(5, rpcTriggerChat, []Variant{NewString("PARTY"), NewString("onward")})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	chat := expectPush(t, b, rpcChatPlayer)
	if chat.Args[0].Str != "PARTY" || chat.Args[1].Str != "Pelia" || chat.Args[2].Str != "onward" {
		t.Fatalf("PARTY chat: got %s", formatArgs(chat.Args))
	}

	// A promotes B -> both receive PartyUpdate with B as leader.
	pkt, _ = BuildRPCPacket(5, rpcTriggerPartyPromote, []Variant{NewString("Popp")})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	updA2 := expectPush(t, a, rpcPartyUpdate)
	updB2 := expectPush(t, b, rpcPartyUpdate)
	if updA2.Args[1].Int != updB2.Args[1].Int {
		t.Fatalf("promote leader mismatch: A=%d B=%d", updA2.Args[1].Int, updB2.Args[1].Int)
	}
	if updA2.Args[1].Int != 2 { // B (Popp) is charID 2
		t.Fatalf("promote: want leader charID 2, got %d", updA2.Args[1].Int)
	}

	// B leaves -> A receives PartyUpdate with only itself.
	pkt, _ = BuildRPCPacket(5, rpcTriggerPartyLeave, nil)
	b.conn.WriteMessage(websocket.BinaryMessage, pkt)
	updA3 := expectPush(t, a, rpcPartyUpdate)
	if updA3.Args[1].Int != 1 { // A (Pelia) is charID 1 and now the leader
		t.Fatalf("after leave: want leader charID 1, got %s", formatArgs(updA3.Args))
	}
	if countJSONArrayMembers(updA3.Args[0].Str) != 1 {
		t.Fatalf("after leave: want 1 member, got %s", updA3.Args[0].Str)
	}
}

// isJSONArray reports whether s parses as a JSON array.
func isJSONArray(s string) bool {
	return countJSONArrayMembers(s) > 0
}

// countJSONArrayMembers returns the number of elements in a JSON array string.
func countJSONArrayMembers(s string) int {
	var arr []int64
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return -1
	}
	return len(arr)
}
