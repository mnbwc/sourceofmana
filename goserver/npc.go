package main

// Phase-3 NPC / quest system.
//
// Mirrors SoM's server-side NPC interaction flow (sources/actor/agent/NpcScript.gd,
// sources/actor/agent/NpcCommons.gd, sources/actor/Progress.gd):
//
//   TriggerInteract(npcRID)  ->  ContextText greeting
//                              ->  ContextChoice (if the NPC has quest choices)
//                                 or ContextContinue / ContextClose
//   TriggerChoice(id)        ->  advance the dialogue, apply the quest action
//   TriggerNextContext       ->  advance one dialogue step
//   TriggerCloseContext      ->  close the dialogue window
//   TriggerSelect(npcRID)    ->  UpdatePublicStats for the target
//   TriggerExplore           ->  (stub: exploration is client-side)
//   TriggerCommand(cmd)      ->  CommandFeedback for unknown commands
//
// Quest progress is stored per character and persisted through the AccountStore.

import (
	"log"
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
	questStateDone     = 2
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
	// Public stats (mirror BaseAgent.stat, used by TriggerSelect).
	Level     int64
	Health    int64
	MaxHealth int64
	Hairstyle int64
	Haircolor int64
	Gender    int64
	Race      int64
	Skintone  int64
}

// Quest is a minimal quest definition with a dialogue tree.
type Quest struct {
	ID          int64
	Name        string
	Description string
	Giver       string // NPC nick
	Steps       []DialogueStep
}

// DialogueStep is one step of an NPC dialogue. A step either shows text,
// shows choices, or runs an action.
type DialogueStep struct {
	Text    string
	Author  string
	Choices []DialogueChoice
}

// DialogueChoice is one option in a choice step.
type DialogueChoice struct {
	Text   string
	Action func(*Server, int64) // applies the quest state change
}

// npcWorld holds all NPCs and quests.
type npcWorld struct {
	mu     sync.RWMutex
	npcs   map[int64]*Npc
	quests map[int64]*Quest
}

func newNpcWorld() *npcWorld {
	w := &npcWorld{npcs: make(map[int64]*Npc), quests: make(map[int64]*Quest)}
	w.seedDefault()
	return w
}

// seedDefault creates a small built-in NPC/quest set so the e2e and loopback
// tests have something to interact with. Real SoM data comes from the DB;
// this is the server-side equivalent of a minimal world seed.
func (w *npcWorld) seedDefault() {
	// Quest 1: "Welcome to Mana" given by the Elder NPC.
	w.quests[1] = &Quest{
		ID:          1,
		Name:        "Welcome to Mana",
		Description: "Speak with the Elder.",
		Giver:       "Elder",
		Steps: []DialogueStep{
			{Text: "Ah, a new face. Welcome to our village, traveller.", Author: "Elder"},
			{Text: "These lands are dangerous. Will you help us?", Author: "Elder", Choices: []DialogueChoice{
				{Text: "I will help.", Action: func(s *Server, charID int64) {
					s.progress.SetQuest(charID, 1, questStateActive)
				}},
				{Text: "Not now.", Action: func(s *Server, charID int64) {
					s.progress.SetQuest(charID, 1, questStateInactive)
				}},
			}},
			{Text: "Then let us begin. Return to me when you are ready.", Author: "Elder"},
		},
	}
	// Quest 2: "A Small Favor" given by the Merchant.
	w.quests[2] = &Quest{
		ID:          2,
		Name:        "A Small Favor",
		Description: "Bring 3 herbs to the Merchant.",
		Giver:       "Merchant",
		Steps: []DialogueStep{
			{Text: "Business is slow. Could you fetch me some herbs?", Author: "Merchant", Choices: []DialogueChoice{
				{Text: "Sure, I'll help.", Action: func(s *Server, charID int64) {
					s.progress.SetQuest(charID, 2, questStateActive)
				}},
				{Text: "Busy right now.", Action: func(s *Server, charID int64) {
					s.progress.SetQuest(charID, 2, questStateInactive)
				}},
			}},
		},
	}
	// NPCs, placed near the spawn point so they are inside the AOI radius.
	w.npcs[1] = &Npc{
		RID: 1, Nick: "Elder", Shape: 100, Pos: Vec2{300, 160}, MapID: spawnMap,
		DefaultState: actorStateIdle, QuestID: 1,
		Level: 5, Health: 100, MaxHealth: 100, Hairstyle: 1, Haircolor: 2,
		Gender: 0, Race: 1, Skintone: 3,
	}
	w.npcs[2] = &Npc{
		RID: 2, Nick: "Merchant", Shape: 101, Pos: Vec2{120, 300}, MapID: spawnMap,
		DefaultState: actorStateIdle, QuestID: 2,
		Level: 3, Health: 80, MaxHealth: 80, Hairstyle: 2, Haircolor: 4,
		Gender: 1, Race: 1, Skintone: 2,
	}
}

func (w *npcWorld) get(npcRID int64) *Npc {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.npcs[npcRID]
}

func (w *npcWorld) quest(questID int64) *Quest {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.quests[questID]
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
	npcRID int64
	step   int
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

func (d *dialogueStore) advance(peerID int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if dlg, ok := d.open[peerID]; ok {
		dlg.step++
	}
}

func (d *dialogueStore) close(peerID int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.open, peerID)
}

// ---- server handlers ----

// handleTriggerInteract mirrors Server.gd TriggerInteract: the player interacts
// with an NPC. It starts a dialogue, pushing the greeting text and the first
// quest choice (or continue/close).
func (s *Server) handleTriggerInteract(c *Client, rpc *ParsedRPC) {
	npcRID := argInt(rpc.Args, 0)
	npc := s.npcs.get(npcRID)
	if npc == nil {
		log.Printf("peer %d TriggerInteract: unknown npcRID=%d", c.peerID, npcRID)
		return
	}
	s.dialogues.begin(c.peerID, npcRID)
	s.applyDialogueStep(c, npc, 0)
}

// handleTriggerChoice mirrors NpcScript.InteractChoice: the player picked a
// dialogue option. It applies the choice's action and advances.
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
	quest := s.npcs.quest(npc.QuestID)
	if quest == nil || dlg.step >= len(quest.Steps) {
		s.dialogues.close(c.peerID)
		return
	}
	choiceID := int(argInt(rpc.Args, 0))
	step := quest.Steps[dlg.step]
	if choiceID < 0 || choiceID >= len(step.Choices) {
		return
	}
	choice := step.Choices[choiceID]
	// Echo the chosen text back to the player, like NpcCommons.ContextText.
	sendToClient(c, nil, rpcContextText, []Variant{NewString(npc.Nick), NewString(choice.Text)})
	if choice.Action != nil {
		choice.Action(s, c.session.charID)
	}
	s.dialogues.close(c.peerID)
	s.pushQuestState(c, c.session.charID)
}

// handleTriggerNextContext mirrors NpcScript.OnContinue: advance the dialogue.
func (s *Server) handleTriggerNextContext(c *Client, rpc *ParsedRPC) {
	dlg := s.dialogues.get(c.peerID)
	if dlg == nil {
		return
	}
	npc := s.npcs.get(dlg.npcRID)
	if npc == nil {
		s.dialogues.close(c.peerID)
		return
	}
	quest := s.npcs.quest(npc.QuestID)
	if quest == nil {
		s.dialogues.close(c.peerID)
		return
	}
	next := dlg.step + 1
	if next >= len(quest.Steps) {
		s.dialogues.close(c.peerID)
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		return
	}
	s.dialogues.advance(c.peerID)
	s.applyDialogueStep(c, npc, next)
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
	// Phase-8: dispatch to the GM command set.
	if s.dispatchGMCommand(c, command) {
		return
	}
	sendToClient(c, nil, rpcCommandFeedback, []Variant{NewString("Unknown command: " + command), NewInt(int64(c.peerID))})
}

// applyDialogueStep pushes one dialogue step to the player, mirroring
// NpcScript.ApplyStep: text, then choices (or continue/close).
func (s *Server) applyDialogueStep(c *Client, npc *Npc, stepIdx int) {
	quest := s.npcs.quest(npc.QuestID)
	if quest == nil || stepIdx >= len(quest.Steps) {
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		return
	}
	step := quest.Steps[stepIdx]
	author := step.Author
	if author == "" {
		author = npc.Nick
	}
	sendToClient(c, nil, rpcContextText, []Variant{NewString(author), NewString(step.Text)})
	if len(step.Choices) > 0 {
		choices := make([]Variant, 0, len(step.Choices))
		for _, ch := range step.Choices {
			choices = append(choices, NewString(ch.Text))
		}
		sendToClient(c, nil, rpcContextChoice, []Variant{{Type: VariantPackedStringArray, Arr: choices}})
		return
	}
	// No choices: continue if there is a next step, else close.
	if stepIdx+1 < len(quest.Steps) {
		sendToClient(c, nil, rpcContextContinue, []Variant{NewInt(int64(c.peerID))})
	} else {
		sendToClient(c, nil, rpcContextClose, []Variant{NewInt(int64(c.peerID))})
		s.dialogues.close(c.peerID)
	}
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
		NewInt(npc.RID),          // agentRID
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
		NewInt(npc.RID),
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
