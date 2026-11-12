package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// The exporter (tools/export/quests.json) writes a bare array, while
// dumpQuestCatalog writes an object with a "quests" key. The loader must
// accept both, otherwise the server cannot re-read its own export.
func TestLoadQuestCatalogFromJSONFormats(t *testing.T) {
	// The loader writes into the package-level catalog, and servers copy that
	// catalog when they start, so the stub quests must not outlive the test.
	defer func() {
		delete(questCatalog, 1)
		delete(questCatalog, 2)
	}()

	dir := t.TempDir()

	bare := filepath.Join(dir, "bare.json")
	if err := os.WriteFile(bare, []byte(
		`[{"id": 1, "name": "Bare Name", "giver": "Elanore"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if n, err := loadQuestCatalogFromJSON(bare); err != nil || n != 1 {
		t.Fatalf("bare array: n=%d err=%v", n, err)
	}
	if got := questCatalog[1].Name; got != "Bare Name" {
		t.Fatalf("bare array name: got %q", got)
	}

	wrapped := filepath.Join(dir, "wrapped.json")
	if err := os.WriteFile(wrapped, []byte(
		`{"quests": [{"id": 2, "name": "Wrapped Name", "giver": "Nina"}]}`),
		0o644); err != nil {
		t.Fatal(err)
	}
	if n, err := loadQuestCatalogFromJSON(wrapped); err != nil || n != 1 {
		t.Fatalf("wrapped object: n=%d err=%v", n, err)
	}
	if got := questCatalog[2].Name; got != "Wrapped Name" {
		t.Fatalf("wrapped object name: got %q", got)
	}

	// Garbage must still be reported as an error.
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"quests": 42}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadQuestCatalogFromJSON(bad); err == nil {
		t.Fatal("bad.json: want error, got nil")
	}
}

// dumpQuestCatalog -> loadQuestCatalogFromJSON must round-trip, so the file
// shipped next to the client can also be fed back to the server.
func TestDumpQuestCatalogRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quests.json")
	if err := dumpQuestCatalog(path); err != nil {
		t.Fatalf("dump: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wrapper struct {
		Quests []dumpQuest `json:"quests"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		t.Fatalf("unmarshal dump: %v", err)
	}
	if len(wrapper.Quests) == 0 {
		t.Fatal("dump is empty")
	}
	if n, err := loadQuestCatalogFromJSON(path); err != nil {
		t.Fatalf("reload: %v", err)
	} else if n != len(wrapper.Quests) {
		t.Fatalf("reload count: got %d want %d", n, len(wrapper.Quests))
	}
}

// choiceLabels extracts the strings from a ContextChoice push.
func choiceLabels(pr *ParsedRPC) []string {
	if len(pr.Args) == 0 {
		return nil
	}
	out := make([]string, 0, len(pr.Args[0].Arr))
	for _, v := range pr.Args[0].Arr {
		out = append(out, v.Str)
	}
	return out
}

// refreshInventoryIDs lists the item ids in a RefreshInventory push.
func refreshInventoryIDs(pr *ParsedRPC) []int64 {
	if len(pr.Args) == 0 {
		return nil
	}
	var out []int64
	for _, entry := range pr.Args[0].Arr {
		if id, ok := entry.Dict["item_id"]; ok {
			out = append(out, id.Int)
		}
	}
	return out
}

// enterQuestTestWorld registers a character and drives it into the spawn map,
// mirroring the setup in TestNpcQuestLoopback.
func enterQuestTestWorld(t *testing.T, a *fakeClient) {
	t.Helper()
	a.announce(t, 5, "/root/Network")
	parseReply(t, a.call(t, 5, rpcCreateAccount, []Variant{
		NewString("itemuser"), NewString("secret1"), NewString("a@b.c"),
		NewBool(false), NewInt(0)}))
	a.callCollect(t, 5, rpcCreateCharacter, []Variant{
		NewString("Ione"), NewDict(map[string]Variant{"race": NewInt(1), "gender": NewInt(0)}),
		NewDict(map[string]Variant{})}, 2)
	pkt, err := BuildRPCPacket(5, rpcConnectCharacter, []Variant{NewString("Ione")})
	if err != nil {
		t.Fatalf("build connect: %v", err)
	}
	if err := a.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("write connect: %v", err)
	}
	for {
		data := a.readPacket(t)
		cmd, pr, _, _, err := ParsePacket(data)
		if err != nil || cmd != CmdRemoteCall || pr == nil || serverPushNameIDs[pr.NameID] {
			continue
		}
		if pr.NameID != rpcCharacterError || pr.Args[0].Int != charErrOK {
			t.Fatalf("connect: want CharacterError(ERR_OK), got %s", formatArgs(pr.Args))
		}
		return
	}
}

// TestNpcQuestItemTurnin covers an item-requirement quest end to end: accept,
// see the "not there yet" hint, hand in the item, and get both the reward and
// an inventory refresh (otherwise the client keeps showing the consumed item).
func TestNpcQuestItemTurnin(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	enterQuestTestWorld(t, a)

	// Nathan (RID 10) offers exactly one quest, so interacting opens the
	// offer directly: stage text + 接受任务 / 以后再说.
	pkt, _ := BuildRPCPacket(5, rpcTriggerInteract, []Variant{NewInt(10)})
	if err := a.conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
		t.Fatalf("interact: %v", err)
	}
	offer := expectPush(t, a, rpcContextText)
	if !strings.Contains(offer.Args[1].Str, "A Drop of Relief") &&
		offer.Args[0].Str != "Nathan" {
		t.Fatalf("offer: got %s", formatArgs(offer.Args))
	}
	choices := expectPush(t, a, rpcContextChoice)
	if !strings.Contains(strings.Join(choiceLabels(choices), "|"), "接受任务") {
		t.Fatalf("offer choices: got %v", choiceLabels(choices))
	}

	// Accept -> UpdateQuest(1). The Water Bottle is missing, so Nathan shows
	// the objective hint and closes the dialogue.
	pkt, _ = BuildRPCPacket(5, rpcTriggerChoice, []Variant{NewInt(0)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	uq := expectPush(t, a, rpcUpdateQuest)
	if uq.Args[0].Int != questID("A Drop of Relief") || uq.Args[1].Int != 1 {
		t.Fatalf("UpdateQuest: got %s", formatArgs(uq.Args))
	}
	hint := expectPush(t, a, rpcContextText)
	if !strings.Contains(hint.Args[1].Str, "Water Bottle") {
		t.Fatalf("want a Water Bottle hint, got %q", hint.Args[1].Str)
	}
	_ = expectPush(t, a, rpcContextClose)

	// Give the requirement item through the store, then talk again: the stage
	// is satisfied, so Nathan offers the turn-in.
	chr := lastTestServer.store.findCharacterByID(1)
	if chr == nil {
		t.Fatal("character not found")
	}
	lastTestServer.store.mu.Lock()
	lastTestServer.addItem(chr, itemID("Water Bottle"), "", 1)
	lastTestServer.store.mu.Unlock()

	pkt, _ = BuildRPCPacket(5, rpcTriggerInteract, []Variant{NewInt(10)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	_ = expectPush(t, a, rpcContextText)
	turnin := expectPush(t, a, rpcContextChoice)
	if !strings.Contains(strings.Join(choiceLabels(turnin), "|"), "完成任务") {
		t.Fatalf("turn-in choices: got %v", choiceLabels(turnin))
	}

	// Turn in: the requirement item is consumed, the client is told about it,
	// the reward (50 exp / 100 GP) is paid and the quest reaches 255.
	pkt, _ = BuildRPCPacket(5, rpcTriggerChoice, []Variant{NewInt(0)})
	a.conn.WriteMessage(websocket.BinaryMessage, pkt)
	var (
		done       bool
		gotRefresh bool
		refreshHas bool
		gp         int64
	)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !done {
		pr := expectPushAny(t, a)
		switch pr.NameID {
		case rpcUpdateQuest:
			done = pr.Args[1].Int == questStateDone
		case rpcRefreshInventory:
			gotRefresh = true
			// The refreshed inventory must no longer contain the bottle.
			for _, id := range refreshInventoryIDs(pr) {
				if id == itemID("Water Bottle") {
					refreshHas = true
				}
			}
		case rpcUpdatePrivateStats:
			gp = pr.Args[1].Int
		}
	}
	if !done {
		t.Fatal("quest did not reach 255")
	}
	if !gotRefresh {
		t.Fatal("no RefreshInventory after consuming the requirement item")
	}
	if refreshHas {
		t.Fatal("refreshed inventory still lists the consumed Water Bottle")
	}
	if gp != 100 {
		t.Fatalf("GP after turn-in: want 100, got %d", gp)
	}
	if n := countItem(lastTestServer.store.findCharacterByID(1), itemID("Water Bottle")); n != 0 {
		t.Fatalf("Water Bottle: want 0 after turn-in, got %d", n)
	}
	doneText := expectPush(t, a, rpcContextText)
	if !strings.Contains(doneText.Args[1].Str, "任务完成") {
		t.Fatalf("done text: got %q", doneText.Args[1].Str)
	}
}
