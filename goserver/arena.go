package main

// Arena module — episode lifecycle + scoring.
//
// Mirrors SoM's sources/network/ArenaController.gd: a timed challenge where a
// player must kill N monsters within a time limit. The bridge ops
// (arena_start / arena_status / arena_stop) drive the lifecycle; kills are
// scored through the very same combat path a human client uses (applyDamage),
// so damage/death/loot/experience all behave identically.
//
// Lifecycle:
//   Start  — validate template + peer, teleport to the template spawn point,
//            spawn up to maxActive monsters around the player, emit the
//            "arena_start" event.
//   Status — return the current episode (score / template / monsters).
//   Stop   — mark the episode stopped, remove every spawned monster, emit the
//            "arena_end" event.
//
// A background ticker (every second) expires episodes whose time limit has
// elapsed, mirrors player deaths into the score, and cleans up episodes whose
// player has left the world.
//
// Thread safety: the service owns one mutex; episode state is only mutated
// under it. World/entity access goes through the world's own lock and event
// pushes through the bridge's lock (always arena -> world/bridge, never the
// reverse), so no lock-order inversion is possible.

import (
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

// Episode statuses (ArenaController.gd "status").
const (
	arenaStatusActive   = "active"
	arenaStatusFinished = "finished"
	arenaStatusExpired  = "expired"
	arenaStatusStopped  = "stopped"
)

// End reasons (the Go migration's names; ArenaController.gd uses
// kills_required_met / expired / explicit_stop).
const (
	arenaEndKillsTargetMet = "kills_target_met"
	arenaEndTimeLimit      = "time_limit"
	arenaEndStopped        = "stopped"
)

// arenaEntityIDBase keeps arena monster CharIDs clear of the seeded NPCs
// (charID -1, -2) and GM spawns (-(rid*1000+i+1)): monsters use
// -(arenaEntityIDBase + seq), e.g. -1000001.
const arenaEntityIDBase int64 = 1000000

// arenaTickDefault is the production expiry-check cadence (the migration
// spec asks for once per second).
const arenaTickDefault = time.Second

// arenaSpawnRadius is the ring radius monsters spawn on around the player.
// It stays well inside the bridge attack reach (Melee: 32+0+32 = 64px).
const arenaSpawnRadius = 40.0

// arenaTemplate is one difficulty preset (ArenaController.gd TEMPLATES).
type arenaTemplate struct {
	Key           string
	Label         string
	Brief         string
	SpawnPos      Vec2
	MonsterNick   string
	KillsRequired int
	MaxActive     int
	TimeLimitSecs int
	MonsterHealth int64
	MonsterLevel  int64
}

// defaultArenaTemplates mirrors the two SoM presets. The Go migration spec
// overrides killsRequired/timeLimit (3/60s and 5/120s).
var defaultArenaTemplates = map[string]*arenaTemplate{
	"tutorial_maggot": {
		Key: "tutorial_maggot", Label: "新手：蛆虫×3",
		Brief:         "击杀 3 只蛆虫。安全区，适合热身。",
		SpawnPos:      Vec2{X: 2230, Y: 1250},
		MonsterNick:   "Maggot",
		KillsRequired: 3,
		MaxActive:     4,
		TimeLimitSecs: 60,
		MonsterHealth: 30,
		MonsterLevel:  1,
	},
	"grind_maggot": {
		Key: "grind_maggot", Label: "进阶：蛆虫×5",
		Brief:         "击杀 5 只蛆虫，注意血量与 respawn。",
		SpawnPos:      Vec2{X: 2230, Y: 1250},
		MonsterNick:   "Maggot",
		KillsRequired: 5,
		MaxActive:     6,
		TimeLimitSecs: 120,
		MonsterHealth: 30,
		MonsterLevel:  1,
	},
}

// ArenaMonster is one spawned monster tracked by an episode.
type ArenaMonster struct {
	EntityID  int64  // unique ID for this monster (world entity CharID)
	Nick      string // "Maggot"
	Pos       Vec2
	Health    int64
	MaxHealth int64
	Alive     bool
}

// ArenaEpisode is one player's arena run.
type ArenaEpisode struct {
	ID            string // "ep-<peerID>-<timestamp>"
	PeerID        int
	Nick          string
	Template      string // "tutorial_maggot" or "grind_maggot"
	KillsRequired int
	Kills         int
	Deaths        int
	XPGained      int
	ItemsLooted   int
	StartTime     time.Time
	EndTime       time.Time
	Status        string // "active", "finished", "expired", "stopped"
	EndReason     string // "kills_target_met", "time_limit", "stopped"
	SpawnPos      Vec2
	Monsters      []*ArenaMonster // spawned monsters

	// mapID is the map the player (and monsters) were on when the episode
	// started; used to despawn monsters on cleanup.
	mapID int
	// deathCounted suppresses double-counting a single death between ticks.
	deathCounted bool
}

// findMonster returns the tracked monster for a world entity CharID.
func (ep *ArenaEpisode) findMonster(entityID int64) *ArenaMonster {
	for _, m := range ep.Monsters {
		if m.EntityID == entityID {
			return m
		}
	}
	return nil
}

// arenaService holds every episode plus the template table. It is owned by
// the Bridge (one service per bridge) and guarded by mu.
type arenaService struct {
	mu        sync.Mutex
	templates map[string]*arenaTemplate
	episodes  map[int]*ArenaEpisode // peerID -> latest episode
	nextSeq   int64                 // episode/monster sequence
	bridge    *Bridge
	tickEvery time.Duration
}

func newArenaService(b *Bridge) *arenaService {
	templates := make(map[string]*arenaTemplate, len(defaultArenaTemplates))
	for k, v := range defaultArenaTemplates {
		templates[k] = v
	}
	return &arenaService{
		templates: templates,
		episodes:  make(map[int]*ArenaEpisode),
		bridge:    b,
		tickEvery: arenaTickDefault,
	}
}

// registerTemplate adds (or replaces) a template. Used by tests to install a
// short time-limit preset; production uses the two SoM defaults.
func (a *arenaService) registerTemplate(t *arenaTemplate) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.templates[t.Key] = t
}

// runTicker expires episodes and mirrors deaths until done is closed. The
// interval is re-read every pass so tests can speed it up after StartBridge.
func (a *arenaService) runTicker(done chan struct{}) {
	for {
		timer := time.NewTimer(a.tickInterval())
		select {
		case <-timer.C:
			a.tick()
		case <-done:
			timer.Stop()
			return
		}
	}
}

// tickInterval reads the (test-overridable) tick cadence.
func (a *arenaService) tickInterval() time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tickEvery <= 0 {
		return arenaTickDefault
	}
	return a.tickEvery
}

// ending describes an episode that ended during a tick pass.
type ending struct {
	peerID  int
	episode *ArenaEpisode
	status  string
	reason  string
}

// tick runs one expiry/death-mirroring pass.
func (a *arenaService) tick() {
	a.mu.Lock()
	now := time.Now()
	var ended []ending
	for peerID, ep := range a.episodes {
		if ep.Status != arenaStatusActive {
			continue
		}
		e := a.bridge.entityByPeer(peerID)
		if e == nil {
			// The player left the world mid-episode: tear it down quietly.
			ep.Status = arenaStatusStopped
			ep.EndReason = arenaEndStopped
			ep.EndTime = now
			a.removeMonstersLocked(ep)
			ended = append(ended, ending{peerID, ep, ep.Status, ep.EndReason})
			continue
		}
		// Mirror player deaths into the score (once per death).
		if e.State == stateDeath {
			if !ep.deathCounted {
				ep.Deaths++
				ep.deathCounted = true
			}
		} else {
			ep.deathCounted = false
		}
		tmpl := a.templates[ep.Template]
		if tmpl == nil {
			continue
		}
		if now.Sub(ep.StartTime) >= time.Duration(tmpl.TimeLimitSecs)*time.Second {
			ep.Status = arenaStatusExpired
			ep.EndReason = arenaEndTimeLimit
			ep.EndTime = now
			a.removeMonstersLocked(ep)
			ended = append(ended, ending{peerID, ep, ep.Status, ep.EndReason})
		}
	}
	a.mu.Unlock()

	for _, e := range ended {
		a.pushEpisodeEnd(e.peerID, e.episode, e.status, e.reason)
	}
}

// start validates the request, teleports the player, spawns the initial
// monsters and records the episode. It mirrors ArenaController.Start.
func (a *arenaService) start(peerID int, nick, templateKey string) map[string]interface{} {
	a.mu.Lock()
	if ep := a.episodes[peerID]; ep != nil && ep.Status == arenaStatusActive {
		a.mu.Unlock()
		return map[string]interface{}{
			"ok": false, "code": "EPISODE_ACTIVE",
			"detail":    "peer already has an active arena episode",
			"episodeId": ep.ID,
		}
	}
	tmpl, ok := a.templates[templateKey]
	if !ok {
		a.mu.Unlock()
		return map[string]interface{}{
			"ok": false, "code": "UNKNOWN_TEMPLATE", "known": a.templateKeys(),
		}
	}
	e := a.bridge.entityByPeer(peerID)
	if e == nil {
		a.mu.Unlock()
		return map[string]interface{}{
			"ok": false, "code": "SESSION_CONFLICT", "detail": "peer not found",
		}
	}

	a.nextSeq++
	ep := &ArenaEpisode{
		ID:            fmt.Sprintf("ep-%d-%d", peerID, time.Now().UnixMilli()),
		PeerID:        peerID,
		Nick:          nick,
		Template:      templateKey,
		KillsRequired: tmpl.KillsRequired,
		StartTime:     time.Now(),
		Status:        arenaStatusActive,
		SpawnPos:      tmpl.SpawnPos,
		mapID:         e.MapID,
	}

	// Authoritative teleport: WarpPlayer to the template spawn, exactly like
	// the GM warp path (NpcCommons.Warp in SoM).
	e.Pos = tmpl.SpawnPos
	e.Vel = Vec2{}
	e.Target = nil
	if c := a.bridge.srv.clientByPeer(peerID); c != nil {
		sendToClient(c, nil, rpcWarpPlayer, []Variant{
			NewInt(int64(e.MapID)), NewVec2(e.Pos.X, e.Pos.Y),
		})
	}
	a.bridge.srv.notifyNeighbours(e, rpcFullUpdateEntity, fullUpdateArgs(e), true)

	// Initial monsters around the player (up to maxActive).
	for i := 0; i < tmpl.MaxActive; i++ {
		a.spawnMonsterLocked(ep, tmpl, e.Pos, ringOffset(i, tmpl.MaxActive))
	}
	a.episodes[peerID] = ep
	monsterCount := len(ep.Monsters)
	episodeID := ep.ID
	a.mu.Unlock()

	a.bridge.pushEvent(peerID, "arena_start", map[string]interface{}{
		"arenaKind":     "arena_start",
		"episodeId":     episodeID,
		"template":      templateKey,
		"brief":         tmpl.Brief,
		"killsRequired": tmpl.KillsRequired,
		"timeLimit":     tmpl.TimeLimitSecs,
		"monsters":      monsterCount,
	})
	log.Printf("arena: %s (peer %d) started %s episode %s (%d monsters, %ds limit)",
		nick, peerID, templateKey, episodeID, monsterCount, tmpl.TimeLimitSecs)
	return map[string]interface{}{
		"ok": true, "episodeId": episodeID, "template": templateKey, "brief": tmpl.Brief,
	}
}

// status returns the peer's episode, or null when there is none.
func (a *arenaService) status(peerID int) map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	ep := a.episodes[peerID]
	if ep == nil {
		return map[string]interface{}{"ok": true, "episode": nil}
	}
	return map[string]interface{}{"ok": true, "episode": a.exportLocked(ep)}
}

// exportLocked builds the status payload; caller holds a.mu.
func (a *arenaService) exportLocked(ep *ArenaEpisode) map[string]interface{} {
	score := map[string]interface{}{
		"kills":       ep.Kills,
		"deaths":      ep.Deaths,
		"xpGained":    ep.XPGained,
		"itemsLooted": ep.ItemsLooted,
	}
	tmplOut := map[string]interface{}{
		"killsRequired": ep.KillsRequired,
		"timeLimit":     0,
		"spawnPos":      bridgeVec2{X: ep.SpawnPos.X, Y: ep.SpawnPos.Y},
	}
	if tmpl := a.templates[ep.Template]; tmpl != nil {
		tmplOut["timeLimit"] = tmpl.TimeLimitSecs
		tmplOut["label"] = tmpl.Label
		tmplOut["monsterNick"] = tmpl.MonsterNick
		tmplOut["maxActive"] = tmpl.MaxActive
	}
	monsters := make([]map[string]interface{}, 0, len(ep.Monsters))
	for _, m := range ep.Monsters {
		monsters = append(monsters, map[string]interface{}{
			"entityID": m.EntityID, "nick": m.Nick, "alive": m.Alive,
			"health": m.Health, "maxHealth": m.MaxHealth,
			"pos": bridgeVec2{X: m.Pos.X, Y: m.Pos.Y},
		})
	}
	out := map[string]interface{}{
		"id":          ep.ID,
		"nick":        ep.Nick,
		"templateKey": ep.Template,
		"template":    tmplOut,
		"status":      ep.Status,
		"score":       score,
		"monsters":    monsters,
		"startedAt":   ep.StartTime.UnixMilli(),
	}
	if !ep.EndTime.IsZero() {
		out["endedAt"] = ep.EndTime.UnixMilli()
	}
	if ep.EndReason != "" {
		out["endReason"] = ep.EndReason
	}
	return out
}

// stop ends the peer's active episode (explicit_stop).
func (a *arenaService) stop(peerID int) map[string]interface{} {
	a.mu.Lock()
	ep := a.episodes[peerID]
	if ep == nil || ep.Status != arenaStatusActive {
		a.mu.Unlock()
		return map[string]interface{}{"ok": false, "code": "NO_ACTIVE_EPISODE"}
	}
	ep.Status = arenaStatusStopped
	ep.EndReason = arenaEndStopped
	ep.EndTime = time.Now()
	episodeID := ep.ID
	a.removeMonstersLocked(ep)
	a.mu.Unlock()

	a.pushEpisodeEnd(peerID, ep, arenaStatusStopped, arenaEndStopped)
	log.Printf("arena: peer %d stopped episode %s", peerID, episodeID)
	return map[string]interface{}{"ok": true, "episodeId": episodeID, "status": arenaStatusStopped}
}

// handleArenaCommand implements the player-facing "/arena" command, which is
// how the human client drives the optional arena. The AgentBridge stays bound
// to localhost on the server, so the command is routed through the normal game
// connection instead; it is self-scoped (the episode always belongs to the
// calling peer) and therefore safe to expose to every player.
//
//	/arena start <template>   start an episode
//	/arena status             show the current score
//	/arena stop               end the episode and remove its monsters
func (s *Server) handleArenaCommand(c *Client, command string) {
	fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(command), "/"))
	action := "status"
	template := ""
	if len(fields) > 1 {
		action = strings.ToLower(fields[1])
	}
	if len(fields) > 2 {
		template = fields[2]
	}

	nick := ""
	if chr := s.store.findCharacterByID(c.session.charID); chr != nil {
		nick = chr.Name
	}

	switch action {
	case "start":
		if template == "" {
			s.commandFeedback(c, "Usage: /arena start <template> | status | stop")
			return
		}
		res := s.bridge.arena.start(c.peerID, nick, template)
		if res["ok"] != true {
			code, _ := res["code"].(string)
			detail, _ := res["detail"].(string)
			msg := "Arena start failed: " + code
			if detail != "" {
				msg += " (" + detail + ")"
			}
			if known, ok := res["known"].([]string); ok && len(known) > 0 {
				msg += " — known templates: " + strings.Join(known, ", ")
			}
			s.commandFeedback(c, msg)
			return
		}
		brief, _ := res["brief"].(string)
		s.commandFeedback(c, fmt.Sprintf("Arena started: %s — %s", res["template"], brief))
	case "status":
		ep := s.bridge.arena.status(c.peerID)
		episode, _ := ep["episode"].(map[string]interface{})
		if episode == nil {
			s.commandFeedback(c, "Arena: no episode (use /arena start <template>).")
			return
		}
		score, _ := episode["score"].(map[string]interface{})
		tmpl, _ := episode["template"].(map[string]interface{})
		if score == nil {
			score = map[string]interface{}{}
		}
		if tmpl == nil {
			tmpl = map[string]interface{}{}
		}
		s.commandFeedback(c, fmt.Sprintf(
			"Arena %s: kills %v/%v, deaths %v, xp %v%s",
			episode["status"], score["kills"], tmpl["killsRequired"],
			score["deaths"], score["xpGained"], arenaEndSuffix(episode)))
	case "stop":
		res := s.bridge.arena.stop(c.peerID)
		if res["ok"] != true {
			code, _ := res["code"].(string)
			s.commandFeedback(c, "Arena stop failed: "+code)
			return
		}
		s.commandFeedback(c, "Arena stopped.")
	default:
		s.commandFeedback(c, "Usage: /arena start <template> | status | stop")
	}
}

// arenaEndSuffix appends the end reason when an episode has finished.
func arenaEndSuffix(episode map[string]interface{}) string {
	reason, _ := episode["endReason"].(string)
	if reason == "" {
		return ""
	}
	return " (" + reason + ")"
}

// handleMonsterDeath processes the death of a world entity that may be an
// arena monster. It returns true when the entity belongs to an arena (the
// caller must not emit its generic kill/died events then).
//
// On a live arena kill it: marks the monster dead, increments the score,
// emits the "kill" + "arena" events, finishes the episode when the kill target
// is met, or spawns a replacement while the episode stays active.
func (a *arenaService) handleMonsterDeath(attacker *Entity, target *Entity) bool {
	a.mu.Lock()
	var ep *ArenaEpisode
	for _, cand := range a.episodes {
		if cand.findMonster(target.CharID) != nil {
			ep = cand
			break
		}
	}
	if ep == nil {
		a.mu.Unlock()
		return false
	}
	mon := ep.findMonster(target.CharID)
	if !mon.Alive {
		a.mu.Unlock()
		return true // corpse of an arena monster: nothing more to do
	}

	mon.Alive = false
	mon.Health = 0
	ep.Kills++
	// Mirror grantKillExperience's NPC formula (5 + level*2) so the episode's
	// XP counter tracks the experience the combat system actually awarded.
	ep.XPGained += int(5 + target.Level*2)
	kills, required, episodeID := ep.Kills, ep.KillsRequired, ep.ID
	attackerNick := ""
	if attacker != nil {
		attackerNick = attacker.Name
	}

	var replacement *ArenaMonster
	finished := kills >= required
	if finished {
		ep.Status = arenaStatusFinished
		ep.EndReason = arenaEndKillsTargetMet
		ep.EndTime = time.Now()
		a.removeMonstersLocked(ep)
	} else if ep.Status == arenaStatusActive {
		// Replacement keeps maxActive monsters in play.
		if tmpl := a.templates[ep.Template]; tmpl != nil {
			if e := a.bridge.entityByPeer(ep.PeerID); e != nil {
				replacement = a.spawnMonsterLocked(ep, tmpl, e.Pos, ringOffset(int(a.nextSeq), tmpl.MaxActive))
			}
		}
	}
	peerID := ep.PeerID
	nick := target.Name
	targetRID := target.CharID
	a.mu.Unlock()

	a.bridge.pushEvent(peerID, "kill", map[string]interface{}{
		"rid": targetRID, "nick": nick, "attackerNick": attackerNick,
	})
	a.bridge.pushEvent(peerID, "arena", map[string]interface{}{
		"arenaKind": "kill", "episodeId": episodeID, "rid": targetRID,
		"nick": nick, "kills": kills, "killsRequired": required,
	})
	if finished {
		a.pushEpisodeEnd(peerID, ep, arenaStatusFinished, arenaEndKillsTargetMet)
		log.Printf("arena: %s finished episode %s (%d/%d kills)",
			attackerNick, episodeID, kills, required)
	} else if replacement != nil {
		log.Printf("arena: episode %s kill %d/%d, replacement %s spawned",
			episodeID, kills, required, replacement.Nick)
	}
	return true
}

// pushEpisodeEnd emits the arena_end event with the final score.
func (a *arenaService) pushEpisodeEnd(peerID int, ep *ArenaEpisode, status, reason string) {
	a.bridge.pushEvent(peerID, "arena_end", map[string]interface{}{
		"arenaKind": "arena_end",
		"episodeId": ep.ID,
		"template":  ep.Template,
		"status":    status,
		"endReason": reason,
		"score": map[string]interface{}{
			"kills":       ep.Kills,
			"deaths":      ep.Deaths,
			"xpGained":    ep.XPGained,
			"itemsLooted": ep.ItemsLooted,
		},
	})
}

// spawnMonsterLocked creates one world entity + episode record. The caller
// holds a.mu; the world entity is added immediately so combat can target it.
func (a *arenaService) spawnMonsterLocked(ep *ArenaEpisode, tmpl *arenaTemplate, center Vec2, offset Vec2) *ArenaMonster {
	a.nextSeq++
	charID := -(arenaEntityIDBase + a.nextSeq)
	pos := Vec2{X: center.X + offset.X, Y: center.Y + offset.Y}
	entity := &Entity{
		CharID:    charID,
		Name:      tmpl.MonsterNick,
		PeerID:    0,
		MapID:     ep.mapID,
		Pos:       pos,
		State:     stateIdle,
		Health:    tmpl.MonsterHealth,
		MaxHealth: tmpl.MonsterHealth,
		Level:     tmpl.MonsterLevel,
	}
	a.bridge.srv.world.add(entity)
	mon := &ArenaMonster{
		EntityID:  charID,
		Nick:      tmpl.MonsterNick,
		Pos:       pos,
		Health:    tmpl.MonsterHealth,
		MaxHealth: tmpl.MonsterHealth,
		Alive:     true,
	}
	ep.Monsters = append(ep.Monsters, mon)

	// Announce the monster to the map (PreloadEntity + UpdateEntity, the same
	// sequence enterWorld uses for other actors).
	s := a.bridge.srv
	s.broadcastToMap(entity.MapID, rpcPreloadEntity, []Variant{
		NewInt(entity.CharID), NewInt(actorTypeNPC), NewInt(0),
		NewString(entity.Name), NewInt(entity.State),
	})
	s.broadcastToMap(entity.MapID, rpcUpdateEntity, updateEntityArgs(entity))
	return mon
}

// removeMonstersLocked despawns every monster the episode spawned. The
// caller holds a.mu.
func (a *arenaService) removeMonstersLocked(ep *ArenaEpisode) {
	s := a.bridge.srv
	for _, m := range ep.Monsters {
		s.world.remove(m.EntityID)
		s.broadcastToMap(ep.mapID, rpcRemoveEntity, []Variant{NewInt(m.EntityID)})
	}
	ep.Monsters = nil
}

// templateKeys lists the known template keys (for UNKNOWN_TEMPLATE replies).
func (a *arenaService) templateKeys() []string {
	keys := make([]string, 0, len(a.templates))
	for k := range a.templates {
		keys = append(keys, k)
	}
	return keys
}

// ringOffset spreads n spawn points evenly on a ring around the player, so
// melee-range bridge attacks (reach 64 with the Melee skill) always land.
func ringOffset(i, n int) Vec2 {
	if n <= 0 {
		n = 1
	}
	angle := 2 * math.Pi * float64(i) / float64(n)
	return Vec2{
		X: float32(arenaSpawnRadius * math.Cos(angle)),
		Y: float32(arenaSpawnRadius * math.Sin(angle)),
	}
}

// shutdown silently tears down active episodes when the bridge stops.
func (a *arenaService) shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for peerID, ep := range a.episodes {
		if ep.Status != arenaStatusActive {
			continue
		}
		ep.Status = arenaStatusStopped
		ep.EndReason = arenaEndStopped
		ep.EndTime = time.Now()
		a.removeMonstersLocked(ep)
		log.Printf("arena: bridge stopped; episode %s for peer %d torn down", ep.ID, peerID)
	}
}
