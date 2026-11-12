package main

// Phase-3 NPC / quest system.
//
// Mirrors SoM's server-side NPC interaction flow (sources/actor/agent/NpcScript.gd,
// sources/actor/agent/NpcCommons.gd, sources/actor/Progress.gd):
//
//   TriggerInteract(npcRID)  ->  ContextText greeting / quest offer
//                              ->  ContextChoice (accept / turn-in / menu)
//                                 or ContextClose
//   TriggerChoice(id)        ->  accept the quest, turn the stage in, or pick
//                                which quest to talk about
//   TriggerNextContext       ->  close the dialogue
//   TriggerCloseContext      ->  close the dialogue
//   TriggerSelect(npcRID)    ->  UpdatePublicStats for the target
//   TriggerExplore           ->  (stub: exploration is client-side)
//   TriggerCommand(cmd)      ->  CommandFeedback for unknown commands
//
// Quest definitions live in quests.go; this file holds the NPC roster, the
// per-character progress store and the dialogue engine that renders them.

import (
	"log"
	"sort"
	"strings"
	"sync"
)

// ActorCommons.Type (sourceofmana/sources/actor/ActorCommons.gd)
const (
	actorTypeNPC = 1
)

// ActorCommons.State (sourceofmana/sources/actor/ActorCommons.gd)
const (
	actorStateIdle = 0
)

// ProgressCommons quest states (sourceofmana/sources/actor/ProgressCommons.gd).
const (
	questStateInactive = 0
	questStateActive   = 1
)

// Npc is a server-side NPC entity. Unlike player entities, NPCs are static:
// they never move and are spawned at world creation.
type Npc struct {
	RID          int64
	Nick         string
	Shape        int64
	Pos          Vec2
	MapID        int
	DefaultState int64
	QuestID      int64 // quest this NPC gives (0 = none)
	Greeting     string
	// Hostile marks a monster rather than a friendly villager. Hostile NPCs use
	// a negative CharID so clients render them as enemies, and they respawn
	// after being killed.
	Hostile bool
	// Drops is the monster's loot table; empty means "use the default roll".
	Drops []npcDrop
	// Public stats (mirror BaseAgent.stat, used by TriggerSelect).
	Level      int64
	Health     int64
	MaxHealth  int64
	Hairstyle  int64
	Haircolor  int64
	Gender     int64
	Race       int64
	Skintone   int64
}

// npcDrop is one entry of an NPC's drop table. chance is 0..1.
type npcDrop struct {
	itemID int64
	count  int64
	chance float64
}

// npcWorld holds all NPCs and the quest catalog.
type npcWorld struct {
	mu      sync.RWMutex
	npcs    map[int64]*Npc
	catalog map[int64]*questDef
	// byNpc maps an NPC rid to the quest ids it offers (explicit QuestID plus
	// every quest whose Giver nick matches).
	byNpc map[int64][]int64
}

func newNpcWorld() *npcWorld {
	w := &npcWorld{
		npcs:    make(map[int64]*Npc),
		catalog: make(map[int64]*questDef),
	}
	for id, q := range questCatalog {
		w.catalog[id] = q
	}
	w.seedDefault()
	w.rebuildQuestIndex()
	return w
}

// npcSpec describes one friendly NPC to place on the spawn map.
type npcSpec struct {
	rid       int64
	nick      string
	pos       Vec2
	shape     int64
	greeting  string
	level     int64
	health    int64
	hairstyle int64
	haircolor int64
	gender    int64
	race      int64
	skintone  int64
}

// spawnNpcs is the friendly roster on the spawn map. The nicks must match the
// Giver fields in quests.go: Elanore, Nina, Frost, Riskim, Gilgames, Kael,
// Ekinu, Ben, Eridu, Nathan, Naem, Nickos, Mana Tree, Peter, Mauro, Bicies,
// Splatyna, Mine Key, Thiefs Chest, Snake Pit Clue, Water Point, Chest,
// Flour Barrel, Market Baker and Abandoned Chest.
var spawnNpcs = []npcSpec{
	{1, "Elanore", Vec2{300, 160}, 100, "艾拉诺：旅人，需要点什么？", 5, 100, 1, 2, 0, 1, 3},
	{2, "Nina", Vec2{200, 200}, 101, "妮娜：嗨！要听点城里的八卦吗？", 2, 60, 3, 5, 1, 1, 2},
	{3, "Frost", Vec2{140, 140}, 102, "弗罗斯特：唉，老骨头不中用了。", 4, 80, 0, 3, 0, 1, 4},
	{4, "Riskim", Vec2{420, 140}, 103, "里斯基姆：面粉！上好的面粉！", 3, 70, 2, 1, 0, 1, 2},
	{5, "Gilgames", Vec2{360, 110}, 104, "吉尔伽美什：远行的人都需要水。", 6, 140, 1, 4, 0, 1, 3},
	{6, "Kael", Vec2{520, 200}, 105, "凯尔：站住。哦，是自己人。", 5, 120, 1, 0, 0, 1, 2},
	{7, "Ekinu", Vec2{300, 420}, 106, "艾基努：矿井的事，问我准没错。", 4, 90, 2, 2, 0, 1, 3},
	{8, "Ben", Vec2{120, 420}, 107, "本：城墙上风大，小心着凉。", 3, 70, 1, 1, 0, 1, 2},
	{9, "Eridu", Vec2{480, 340}, 108, "埃里杜：沙漠把一切都变成了沙子。", 4, 85, 0, 5, 0, 1, 4},
	{10, "Nathan", Vec2{600, 260}, 109, "内森：热死了，热死了……", 3, 75, 2, 0, 0, 1, 2},
	{11, "Naem", Vec2{650, 330}, 110, "纳姆：矿井里掉石头是常事，小心脑袋。", 3, 80, 1, 3, 0, 1, 3},
	{12, "Nickos", Vec2{680, 390}, 111, "尼科斯：我昨天在矿井深处看到的东西……唉。", 3, 80, 0, 2, 0, 1, 2},
	{13, "Mana Tree", Vec2{720, 470}, 112, "（魔力树静立在矿井深处，枝叶泛着微光。）", 8, 200, 0, 0, 0, 3, 0},
	{14, "Peter", Vec2{760, 420}, 113, "彼得：船舱里全是 Ratto，快来帮忙！", 4, 95, 1, 1, 0, 1, 2},
	{15, "Mauro", Vec2{780, 180}, 114, "毛罗：唉，又被蛇咬了……", 3, 70, 2, 4, 0, 1, 3},
	{16, "Bicies", Vec2{840, 300}, 115, "比西斯：史莱普蒂娜大人在前面等你。", 4, 90, 3, 2, 1, 1, 2},
	{17, "Splatyna", Vec2{880, 420}, 116, "史莱普蒂娜：咯咯……有客人来了。", 9, 300, 0, 0, 1, 2, 0},
	{18, "Mine Key", Vec2{640, 470}, 117, "（一把生锈的 Mine Key，插在沙土里。）", 1, 1, 0, 0, 0, 0, 0},
	{19, "Thiefs Chest", Vec2{900, 200}, 118, "（一个被撬开过、又合上的 Thiefs Chest。）", 1, 1, 0, 0, 0, 0, 0},
	{20, "Snake Pit Clue", Vec2{860, 140}, 119, "（沙地上插着一张 Snake Pit Clue，字迹潦草。）", 1, 1, 0, 0, 0, 0, 0},
	{21, "Water Point", Vec2{820, 60}, 120, "（一汪清泉，是这附近唯一的 Water Point。）", 1, 1, 0, 0, 0, 0, 0},
	{22, "Chest", Vec2{940, 300}, 121, "（一个被沙土半埋的 Chest，锁已经锈住了。）", 1, 1, 0, 0, 0, 0, 0},
	{23, "Flour Barrel", Vec2{500, 420}, 122, "（一只 Flour Barrel，桶口还沾着面粉。）", 1, 1, 0, 0, 0, 0, 0},
	{24, "Market Baker", Vec2{240, 120}, 123, "面包师傅：刚出炉的 Croissant！", 3, 70, 1, 2, 1, 1, 2},
	{25, "Abandoned Chest", Vec2{700, 520}, 124, "（废弃矿井深处的 Abandoned Chest，锁孔里插着半截钥匙。）", 1, 1, 0, 0, 0, 0, 0},
}

// seedDefault places the friendly roster and the field monsters on the spawn
// map. Real SoM data comes from the DB; this is the server-side equivalent of
// a minimal world seed covering the original quest-givers.
func (w *npcWorld) seedDefault() {
	for _, spec := range spawnNpcs {
		w.npcs[spec.rid] = &Npc{
			RID: spec.rid, Nick: spec.nick, Shape: spec.shape, Pos: spec.pos,
			MapID: spawnMap, DefaultState: actorStateIdle,
			Greeting:   spec.greeting,
			Level:      spec.level,
			Health:     spec.health,
			MaxHealth:  spec.health,
			Hairstyle:  spec.hairstyle,
			Haircolor:  spec.haircolor,
			Gender:     spec.gender,
			Race:       spec.race,
			Skintone:   spec.skintone,
		}
	}
	w.seedFieldMonsters()
}

// monsterRIDBase keeps monster RIDs clear of the friendly NPC RIDs (1..25).
const monsterRIDBase int64 = 1000

// fieldMonsterSpec is one monster placed in the starting field.
type fieldMonsterSpec struct {
	nick   string
	pos    Vec2
	level  int64
	health int64
	shape  int64
	drops  []npcDrop
}

// drop is a compact npcDrop literal: drop("Maggot Slime", 1, 0.6).
func drop(name string, count int64, chance float64) npcDrop {
	return npcDrop{itemID: itemID(name), count: count, chance: chance}
}

// fieldMonsters mirrors the low-level creatures around the SoM starting area
// (tools/export/entities.json + tools/export/drops). They give the "normal
// game flow" something to fight after the Short Sword, and their drops feed
// the gathering quests (Maggot Slime, Bone, Salt, Sulphur Powder, Pitaya,
// Snake Skin).
var fieldMonsters = []fieldMonsterSpec{
	{nick: "Piou", pos: Vec2{210, 250}, level: 1, health: 30, shape: 201},
	{nick: "Piou", pos: Vec2{260, 290}, level: 1, health: 30, shape: 201},
	{nick: "Maggot", pos: Vec2{330, 250}, level: 1, health: 50, shape: 202,
		drops: []npcDrop{drop("Maggot Slime", 1, 0.6)}},
	{nick: "Maggot", pos: Vec2{380, 300}, level: 1, health: 50, shape: 202,
		drops: []npcDrop{drop("Maggot Slime", 1, 0.6)}},
	{nick: "Maggot", pos: Vec2{430, 240}, level: 2, health: 70, shape: 202,
		drops: []npcDrop{drop("Maggot Slime", 1, 0.6)}},
	{nick: "Skeleton", pos: Vec2{540, 420}, level: 2, health: 60, shape: 203,
		drops: []npcDrop{drop("Bone", 1, 0.33)}},
	{nick: "Sea Slime", pos: Vec2{180, 480}, level: 1, health: 40, shape: 204,
		drops: []npcDrop{drop("Salt", 1, 0.33)}},
	{nick: "Fire Goblin", pos: Vec2{620, 180}, level: 2, health: 70, shape: 205,
		drops: []npcDrop{drop("Sulphur Powder", 1, 0.2)}},
	{nick: "Peyote", pos: Vec2{460, 480}, level: 1, health: 45, shape: 206,
		drops: []npcDrop{drop("Pitaya", 1, 0.33)}},
	{nick: "Snake", pos: Vec2{880, 360}, level: 2, health: 80, shape: 207,
		drops: []npcDrop{drop("Snake Skin", 1, 0.1)}},
	{nick: "Ratto", pos: Vec2{760, 480}, level: 1, health: 35, shape: 208},
	{nick: "Ratto", pos: Vec2{800, 520}, level: 1, health: 35, shape: 208},
	{nick: "Ratto", pos: Vec2{840, 480}, level: 1, health: 35, shape: 208},
	{nick: "Ratto", pos: Vec2{720, 540}, level: 1, health: 35, shape: 208},
	{nick: "Ratto", pos: Vec2{820, 440}, level: 1, health: 35, shape: 208},
	{nick: "Ratto", pos: Vec2{860, 520}, level: 1, health: 35, shape: 208},
}

// seedFieldMonsters adds the starting-field monsters to the NPC table. They
// are Hostile, so they use a negative CharID and respawn after death.
func (w *npcWorld) seedFieldMonsters() {
	for i, spec := range fieldMonsters {
		rid := monsterRIDBase + int64(i)
		w.npcs[rid] = &Npc{
			RID: rid, Nick: spec.nick, Shape: spec.shape, Pos: spec.pos,
			MapID: spawnMap, DefaultState: actorStateIdle,
			Hostile: true,
			Drops:   spec.drops,
			Level:   spec.level, Health: spec.health, MaxHealth: spec.health,
			Hairstyle: 0, Haircolor: 0, Gender: 0, Race: 2, Skintone: 0,
		}
	}
}

// isHostile reports whether the NPC is a monster (negative CharID namespace).
func (n *Npc) isHostile() bool {
	return n != nil && n.Hostile
}

// preloadRID returns the agentRID to send in PreloadEntity / UpdatePublicStats.
// Friendly NPCs keep their positive RID (clients treat positive RIDs as
// villagers); hostile monsters use the negative namespace so clients render
// them as enemies. Either way the client derives the world-entity key (-RID)
// from this value, matching the entity's CharID.
func (n *Npc) preloadRID() int64 {
	if n == nil {
		return 0
	}
	if n.Hostile {
		return -n.RID
	}
	return n.RID
}

func (w *npcWorld) get(npcRID int64) *Npc {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.npcs[npcRID]
}

func (w *npcWorld) allNpcs() []*Npc {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]*Npc, 0, len(w.npcs))
	for _, n := range w.npcs {
		out = append(out, n)
	}
	return out
}

// rebuildQuestIndex regenerates the NPC -> quest index from the catalog.
func (w *npcWorld) rebuildQuestIndex() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.byNpc = make(map[int64][]int64)
	for rid, n := range w.npcs {
		if n.QuestID != 0 {
			w.byNpc[rid] = append(w.byNpc[rid], n.QuestID)
		}
	}
	for id, q := range w.catalog {
		if q.Hidden || q.Giver == "" {
			continue
		}
		n := w.npcByNickLocked(q.Giver)
		if n == nil {
			log.Printf("quests: giver %q has no NPC; quest %s (%d) will not be offered",
				q.Giver, q.Name, id)
			continue
		}
		w.byNpc[n.RID] = append(w.byNpc[n.RID], id)
	}
}

// npcByNickLocked returns the lowest-RID NPC with that nick, so lookups stay
// deterministic even though monster names repeat. Callers hold mu.
func (w *npcWorld) npcByNickLocked(nick string) *Npc {
	var found *Npc
	for _, n := range w.npcs {
		if n.Nick != nick {
			continue
		}
		if found == nil || n.RID < found.RID {
			found = n
		}
	}
	return found
}

// npcByNick finds an NPC by its nick (safe for concurrent use).
func (w *npcWorld) npcByNick(nick string) *Npc {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.npcByNickLocked(nick)
}

// allQuestIDs returns the catalog ids in a stable order.
func (w *npcWorld) allQuestIDs() []int64 {
	w.mu.RLock()
	out := make([]int64, 0, len(w.catalog))
	for id := range w.catalog {
		out = append(out, id)
	}
	w.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// questsForNpc returns the quest ids offered by an NPC, sorted by id so the
// menu order is deterministic (the catalog is a map).
func (w *npcWorld) questsForNpc(npcRID int64) []int64 {
	w.mu.RLock()
	out := make([]int64, 0, len(w.byNpc[npcRID]))
	out = append(out, w.byNpc[npcRID]...)
	w.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ---- per-character progress ----

// characterProgress is the quest/bestiary/skill progress of one character.
// It mirrors SoM's ActorProgress (quests/bestiary/skills dictionaries).
type characterProgress struct {
	mu       sync.Mutex
	quests   map[int64]int64
	bestiary map[int64]int64
	skills   map[int64]int64
}

func newCharacterProgress() *characterProgress {
	return &characterProgress{
		quests:   make(map[int64]int64),
		bestiary: make(map[int64]int64),
		skills:   make(map[int64]int64),
	}
}

// progressStore holds progress per character ID.
type progressStore struct {
	mu   sync.Mutex
	byID map[int64]*characterProgress
}

func newProgressStore() *progressStore {
	return &progressStore{byID: make(map[int64]*characterProgress)}
}

func (p *progressStore) get(charID int64) *characterProgress {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp, ok := p.byID[charID]
	if !ok {
		cp = newCharacterProgress()
		p.byID[charID] = cp
	}
	return cp
}

// SetQuest mirrors Progress.SetQuest: updates the state, persists it, and
// pushes UpdateQuest to the owning peer.
func (p *progressStore) SetQuest(charID int64, questID int64, state int64) {
	cp := p.get(charID)
	cp.mu.Lock()
	cp.quests[questID] = state
	cp.mu.Unlock()
	log.Printf("progress: char %d quest %d -> state %d", charID, questID, state)
}

// GetQuest mirrors Progress.GetQuest.
func (p *progressStore) GetQuest(charID int64, questID int64) int64 {
	cp := p.get(charID)
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if s, ok := cp.quests[questID]; ok {
		return s
	}
	return questStateInactive
}

// AddBestiary mirrors Progress.AddBestiary: increments the kill count for a
// monster and returns the new total. The caller is responsible for pushing
// UpdateBestiary to the owning peer (so this stays a pure data operation).
func (p *progressStore) AddBestiary(charID int64, entityID int64, killedCount int64) int64 {
	if killedCount <= 0 {
		killedCount = 1
	}
	cp := p.get(charID)
	cp.mu.Lock()
	total := cp.bestiary[entityID] + killedCount
	cp.bestiary[entityID] = total
	cp.mu.Unlock()
	log.Printf("progress: char %d bestiary %d -> %d", charID, entityID, total)
	return total
}

// GetBestiary mirrors Progress.GetBestiary.
func (p *progressStore) GetBestiary(charID int64, monsterID int64) int64 {
	cp := p.get(charID)
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.bestiary[monsterID]
}

// AddSkill mirrors Progress.AddSkill: sets a skill's level (0 = remove).
// Returns the previous level so callers can detect a change.
func (p *progressStore) AddSkill(charID int64, skillID int64, level int64) int64 {
	cp := p.get(charID)
	cp.mu.Lock()
	prev := cp.skills[skillID]
	if level <= 0 {
		delete(cp.skills, skillID)
	} else {
		cp.skills[skillID] = level
	}
	cp.mu.Unlock()
	log.Printf("progress: char %d skill %d -> level %d (was %d)", charID, skillID, level, prev)
	return prev
}

// RemoveSkill mirrors Progress.RemoveSkill: erases the skill entry.
func (p *progressStore) RemoveSkill(charID int64, skillID int64) {
	cp := p.get(charID)
	cp.mu.Lock()
	delete(cp.skills, skillID)
	cp.mu.Unlock()
	log.Printf("progress: char %d skill %d removed", charID, skillID)
}

// GetSkillLevel mirrors Progress.GetSkillLevel.
func (p *progressStore) GetSkillLevel(charID int64, skillID int64) int64 {
	cp := p.get(charID)
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.skills[skillID]
}

// ---- dialogue state per peer ----

// dialogue tracks an open NPC conversation for one peer.
type dialogue struct {
	npcRID  int64
	step    int
	mode    string // "", "menu", "accept", "quest"
	questID int64
	stage   int
	quests  []int64 // menu mode: the quests on offer
}

// dialogues maps peerID -> open dialogue.
type dialogueStore struct {
	mu   sync.Mutex
	open map[int]*dialogue
}

func newDialogueStore() *dialogueStore {
	return &dialogueStore{open: make(map[int]*dialogue)}
}

func (d *dialogueStore) begin(peerID int, npcRID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.open[peerID] = &dialogue{npcRID: npcRID}
}

func (d *dialogueStore) get(peerID int) *dialogue {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.open[peerID]
}

func (d *dialogueStore) close(peerID int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.open, peerID)
}

// ---- server handlers ----

// handleTriggerInteract mirrors Server.gd TriggerInteract: the player interacts
// with an NPC. Friendly NPCs with quests open the quest dialogue; a NPC with
// several quests first offers a menu, and one without any just greets.
func (s *Server) handleTriggerInteract(c *Client, rpc *ParsedRPC) {
	npcRID := argInt(rpc.Args, 0)
	npc := s.npcs.get(npcRID)
	if npc == nil {
		log.Printf("peer %d TriggerInteract: unknown npcRID=%d", c.peerID, npcRID)
		return
	}
	if npc.Hostile {
		// Monsters do not talk; attacking them is the interaction.
		return
	}
	// Talking to an NPC fulfils the current "talk" stage of any active quest
	// that targets them, even when the NPC offers no quests of its own.
	if advanced := s.advanceTalkQuests(c, npc); len(advanced) > 0 {
		q := advanced[len(advanced)-1]
		state := s.progress.GetQuest(c.session.charID, q.ID)
		if state >= questStateDone {
			s.showQuestDone(c, npc, q)
		} else {
			s.showQuestStage(c, npc, q, int(state))
		}
		return
	}
	quests := s.npcs.questsForNpc(npcRID)
	if len(quests) == 0 {
		s.dialogues.begin(c.peerID, npcRID)
		s.applyDialogueStep(c, npc, 0)
		return
	}
	if len(quests) == 1 {
		s.openQuest(c, npc, quests[0])
		return
	}
	// Several quests: let the player pick one.
	s.dialogues.begin(c.peerID, npcRID)
	dlg := s.dialogues.get(c.peerID)
	dlg.mode = "menu"
	dlg.quests = quests
	sendToClient(c, nil, rpcContextText, []Variant{NewString(npc.Nick), NewString(npc.Greeting)})
	choices := make([]Variant, 0, len(quests)+1)
	for _, qid := range quests {
		q := s.npcs.questDef(qid)
		if q == nil {
			continue
		}
		choices = append(choices, NewString(q.Name))
	}
	choices = append(choices, NewString("离开"))
	sendToClient(c, nil, rpcContextChoice, []Variant{{Type: VariantPackedStringArray, Arr: choices}})
}

// advanceTalkQuests completes the current stage of every active quest whose
// requirement is "talk to npc". SoM advances these on interaction regardless
// of who gives the quest, so a chain such as Elanore -> Kael -> Ekinu works
// even though Kael offers nothing himself. Returns the quests that advanced,
// in catalog order.
func (s *Server) advanceTalkQuests(c *Client, npc *Npc) []*questDef {
	charID := c.session.charID
	var advanced []*questDef
	for _, qid := range s.npcs.allQuestIDs() {
		state := s.progress.GetQuest(charID, qid)
		if state <= questStateInactive || state >= questStateDone {
			continue
		}
		q := s.npcs.questDef(qid)
		if q == nil || int(state) > len(q.Stages) {
			continue
		}
		st := q.Stages[state-1]
		if st.Req.Kind != "talk" || st.Req.TalkNPC != npc.Nick {
			continue
		}
		s.completeStage(c, q, int(state))
		advanced = append(advanced, q)
	}
	return advanced
}

// openQuest renders the dialogue for one quest at an NPC: the offer, the
// current stage, or the completion line.
func (s *Server) openQuest(c *Client, npc *Npc, qid int64) {
	q := s.npcs.questDef(qid)
	if q == nil || len(q.Stages) == 0 {
		s.dialogues.begin(c.peerID, npc.RID)
		s.applyDialogueStep(c, npc, 0)
		return
	}
	charID := c.session.charID
	state := s.progress.GetQuest(charID, qid)
	if state >= questStateDone {
		if q.Repeatable {
			// Re-arm the quest so it can be taken again.
			s.progress.SetQuest(charID, qid, questStateInactive)
			state = questStateInactive
		} else {
			s.dialogues.begin(c.peerID, npc.RID)
			sendToClient(c, nil, rpcContextText, []Variant{
				NewString(npc.Nick), NewString("【" + q.Name + "】已经完成了。干得漂亮。"),
			})
			sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
			s.dialogues.close(c.peerID)
			return
		}
	}
	if state == questStateInactive {
		s.dialogues.begin(c.peerID, npc.RID)
		dlg := s.dialogues.get(c.peerID)
		dlg.mode = "accept"
		dlg.questID = qid
		sendToClient(c, nil, rpcContextText, []Variant{
			NewString(npc.Nick), NewString(stageText(q.Stages[0], npc.Nick)),
		})
		sendToClient(c, nil, rpcContextChoice, []Variant{
			{Type: VariantPackedStringArray, Arr: []Variant{
				NewString("接受任务"), NewString("以后再说"),
			}},
		})
		return
	}
	s.showQuestStage(c, npc, q, int(state))
}

// showQuestStage renders stage k of q at npc. Talk requirements are fulfilled
// by the interaction itself and auto-advance; Auto stages advance as soon as
// their requirement is met. The loop handles chains such as
// Elanore -> Kael -> Ekinu without the player re-clicking.
func (s *Server) showQuestStage(c *Client, npc *Npc, q *questDef, k int) {
	if len(q.Stages) == 0 {
		s.dialogues.close(c.peerID)
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		return
	}
	charID := c.session.charID
	chr := s.store.findCharacterByID(charID)
	for k <= len(q.Stages) {
		st := q.Stages[k-1]
		if st.Req.Kind == "talk" && st.Req.TalkNPC == npc.Nick {
			k = int(s.completeStage(c, q, k))
			continue
		}
		if ok, _ := s.stageReqSatisfied(chr, q, k, npc.Nick); st.Auto && ok {
			k = int(s.completeStage(c, q, k))
			continue
		}
		break
	}
	if k > len(q.Stages) {
		s.showQuestDone(c, npc, q)
		return
	}
	st := q.Stages[k-1]
	text := stageText(st, npc.Nick)
	s.dialogues.begin(c.peerID, npc.RID)
	dlg := s.dialogues.get(c.peerID)
	dlg.mode = "quest"
	dlg.questID = q.ID
	dlg.stage = k
	if ok, _ := s.stageReqSatisfied(chr, q, k, npc.Nick); ok {
		label := "完成任务"
		if st.Req.Kind == "" {
			label = "继续"
		}
		sendToClient(c, nil, rpcContextText, []Variant{NewString(npc.Nick), NewString(text)})
		sendToClient(c, nil, rpcContextChoice, []Variant{
			{Type: VariantPackedStringArray, Arr: []Variant{
				NewString(label), NewString("稍后再说"),
			}},
		})
		return
	}
	if hint := s.reqHint(chr, q, k); hint != "" {
		text += "\n（还差：" + hint + "）"
	}
	sendToClient(c, nil, rpcContextText, []Variant{NewString(npc.Nick), NewString(text)})
	sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
	s.dialogues.close(c.peerID)
}

// showQuestDone tells the player the quest is finished.
func (s *Server) showQuestDone(c *Client, npc *Npc, q *questDef) {
	s.dialogues.begin(c.peerID, npc.RID)
	sendToClient(c, nil, rpcContextText, []Variant{
		NewString(npc.Nick), NewString("【" + q.Name + "】任务完成！"),
	})
	sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
	s.dialogues.close(c.peerID)
}

// handleTriggerChoice mirrors NpcScript.InteractChoice: the player picked a
// dialogue option (menu entry, accept/decline, or turn-in).
func (s *Server) handleTriggerChoice(c *Client, rpc *ParsedRPC) {
	dlg := s.dialogues.get(c.peerID)
	if dlg == nil {
		return
	}
	npc := s.npcs.get(dlg.npcRID)
	if npc == nil {
		s.dialogues.close(c.peerID)
		return
	}
	choiceID := int(argInt(rpc.Args, 0))
	switch dlg.mode {
	case "menu":
		if choiceID >= 0 && choiceID < len(dlg.quests) {
			s.openQuest(c, npc, dlg.quests[choiceID])
			return
		}
		s.dialogues.close(c.peerID)
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		return
	case "accept":
		q := s.npcs.questDef(dlg.questID)
		s.dialogues.close(c.peerID)
		if q == nil || len(q.Stages) == 0 {
			return
		}
		if choiceID == 0 {
			s.acceptQuest(c, q)
			s.showQuestStage(c, npc, q, 1)
			return
		}
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		return
	case "quest":
		q := s.npcs.questDef(dlg.questID)
		s.dialogues.close(c.peerID)
		if q == nil || len(q.Stages) == 0 {
			return
		}
		if choiceID == 0 {
			next := s.completeStage(c, q, dlg.stage)
			if next >= questStateDone {
				s.showQuestDone(c, npc, q)
			} else {
				s.showQuestStage(c, npc, q, int(next))
			}
			return
		}
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		return
	}
	// Plain greeting: nothing to choose.
	s.dialogues.close(c.peerID)
	sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
}

// handleTriggerNextContext mirrors NpcScript.OnContinue. The quest dialogue
// shows a single text page per step, so continuing just closes it.
func (s *Server) handleTriggerNextContext(c *Client, rpc *ParsedRPC) {
	s.dialogues.close(c.peerID)
	sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
}

// handleTriggerCloseContext mirrors NpcCommons.TryCloseContext.
func (s *Server) handleTriggerCloseContext(c *Client, rpc *ParsedRPC) {
	s.dialogues.close(c.peerID)
	sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
}

// handleTriggerSelect mirrors Server.gd TriggerSelect: push the target's
// public stats to the selecting player.
func (s *Server) handleTriggerSelect(c *Client, rpc *ParsedRPC) {
	targetRID := argInt(rpc.Args, 0)
	if npc := s.npcs.get(targetRID); npc != nil {
		sendToClient(c, nil, rpcUpdatePublicStats, npcUpdatePublicStatsArgs(npc))
	}
}

// handleTriggerExplore mirrors Server.gd TriggerExplore. Exploration is a
// client-side action in SoM; the server just acknowledges.
func (s *Server) handleTriggerExplore(c *Client, rpc *ParsedRPC) {
	log.Printf("peer %d TriggerExplore", c.peerID)
}

// handleTriggerCommand mirrors Server.gd TriggerCommand: dispatch to the
// CommandManager. Known commands are handled; unknown commands get a
// CommandFeedback message.
func (s *Server) handleTriggerCommand(c *Client, rpc *ParsedRPC) {
	command := argString(rpc.Args, 0)
	log.Printf("peer %d TriggerCommand %q", c.peerID, command)
	// /trade <nick> initiates a trade (mirrors WorldCommands.CommandTrade).
	if strings.HasPrefix(strings.TrimSpace(command), "trade ") {
		s.handleTradeCommand(c, command)
		return
	}
	// /arena start|status|stop runs the optional AI arena for the calling peer
	// only. Routed through the game connection so the AgentBridge can stay
	// bound to localhost on the server. Tolerate a leading "/" (Chat.gd strips
	// it, but other clients may not).
	if trimmed := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(command), "/")); trimmed == "arena" ||
		strings.HasPrefix(trimmed, "arena ") {
		s.handleArenaCommand(c, command)
		return
	}
	// Phase-8: dispatch to the GM command set.
	if s.dispatchGMCommand(c, command) {
		return
	}
	sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("Unknown command: " + command), NewInt(int64(c.peerID))})
}

// applyDialogueStep pushes one step of a plain (questless) NPC dialogue,
// mirroring NpcScript.ApplyStep: text, then choices (or continue/close).
func (s *Server) applyDialogueStep(c *Client, npc *Npc, stepIdx int) {
	if stepIdx > 0 || npc.Greeting == "" {
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		return
	}
	sendToClient(c, nil, rpcContextText, []Variant{NewString(npc.Nick), NewString(npc.Greeting)})
	sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
}

// pushQuestState sends UpdateQuest for every tracked quest to the player.
func (s *Server) pushQuestState(c *Client, charID int64) {
	cp := s.progress.get(charID)
	cp.mu.Lock()
	defer cp.mu.Unlock()
	for questID, state := range cp.quests {
		sendToClient(c, nil, rpcUpdateQuest, []Variant{NewInt(questID), NewInt(state)})
	}
}

// pushRefreshProgress sends RefreshProgress with the full progress dictionaries.
func (s *Server) pushRefreshProgress(c *Client, charID int64) {
	cp := s.progress.get(charID)
	cp.mu.Lock()
	quests := make(map[string]Variant, len(cp.quests))
	for id, st := range cp.quests {
		quests[ftoa(float64(id))] = NewInt(st)
	}
	bestiary := make(map[string]Variant, len(cp.bestiary))
	for id, cnt := range cp.bestiary {
		bestiary[ftoa(float64(id))] = NewInt(cnt)
	}
	skills := make(map[string]Variant, len(cp.skills))
	for id, lv := range cp.skills {
		skills[ftoa(float64(id))] = NewInt(lv)
	}
	cp.mu.Unlock()
	sendToClient(c, nil, rpcRefreshProgress, []Variant{
		NewDict(skills), NewDict(quests), NewDict(bestiary),
	})
}

// npcPublicStatsArgs mirrors World.BulkPreload's PreloadEntity payload for an
// NPC (agentRID, actorType, currentShape, nick, defaultState).
func npcPublicStatsArgs(npc *Npc) []Variant {
	return []Variant{
		NewInt(npc.preloadRID()), // agentRID
		NewInt(actorTypeNPC),     // actorType
		NewInt(npc.Shape),        // currentShape
		NewString(npc.Nick),      // nick
		NewInt(npc.DefaultState), // defaultState
	}
}

// npcUpdatePublicStatsArgs mirrors Server.TriggerSelect's UpdatePublicStats
// payload (agentRID, level, health, maxHealth, hairstyle, haircolor, gender,
// race, skintone, currentShape).
func npcUpdatePublicStatsArgs(npc *Npc) []Variant {
	return []Variant{
		NewInt(npc.preloadRID()),
		NewInt(npc.Level),
		NewInt(npc.Health),
		NewInt(npc.MaxHealth),
		NewInt(npc.Hairstyle),
		NewInt(npc.Haircolor),
		NewInt(npc.Gender),
		NewInt(npc.Race),
		NewInt(npc.Skintone),
		NewInt(npc.Shape),
	}
}
