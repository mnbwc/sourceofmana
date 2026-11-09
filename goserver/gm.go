package main

// GM (Game Master) web API — HTTP JSON endpoints for browser-based server
// management. Serves a single-page admin UI at /gm/ and exposes REST
// endpoints under /gm/api/ for account, character, item, and command
// management.
//
// Endpoints:
//   GET  /gm/                     → GM web page (embedded HTML)
//   GET  /gm/api/status           → server status (online players, uptime)
//   GET  /gm/api/accounts         → list all accounts with characters
//   POST /gm/api/accounts/create  → create account {name, password, permission}
//   POST /gm/api/accounts/delete  → delete account {name}
//   POST /gm/api/accounts/perm    → set account permission {name, permission}
//   GET  /gm/api/characters       → list all characters (across accounts)
//   POST /gm/api/characters/stats → update character stats {char_id, level, gp, health, mana, stamina, experience, karma}
//   POST /gm/api/characters/delete→ delete character {char_id}
//   GET  /gm/api/players          → list online players
//   POST /gm/api/players/kick     → kick player {name}
//   POST /gm/api/players/ban      → ban player {name, reason}
//   GET  /gm/api/items            → list item catalog
//   POST /gm/api/items/give       → give item to character {char_id, item_id, count}
//   POST /gm/api/items/remove     → remove item from character {char_id, item_id, count}
//   POST /gm/api/command          → execute GM command as server {command}
//   POST /gm/api/broadcast        → broadcast message to all online players {message}

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed gm.html
var gmFS embed.FS

// gmStartTime is set in main() for uptime reporting.
var gmStartTime = time.Now()

// ---- JSON response helpers ----

func gmWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

func gmError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": msg})
}

func gmOK(w http.ResponseWriter, data interface{}) {
	gmWriteJSON(w, map[string]interface{}{"ok": true, "data": data})
}

// ---- request helpers ----

func gmDecode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func gmStrField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		case float64:
			return strconv.FormatInt(int64(t), 10)
		}
	}
	return ""
}

func gmIntField(m map[string]interface{}, key string, def int64) int64 {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case float64:
			return int64(t)
		case string:
			if n, err := strconv.ParseInt(t, 10, 64); err == nil {
				return n
			}
		}
	}
	return def
}

// ---- data structures for JSON responses ----

type gmAccountJSON struct {
	Name       string              `json:"name"`
	Password   string              `json:"password,omitempty"`
	Token      string              `json:"token,omitempty"`
	Permission int                 `json:"permission"`
	Characters []gmCharacterJSON   `json:"characters"`
}

type gmCharacterJSON struct {
	ID         int64            `json:"id"`
	Name       string           `json:"name"`
	Level      int64            `json:"level"`
	Race       int64            `json:"race"`
	Gender     int64            `json:"gender"`
	Hairstyle  int64            `json:"hairstyle"`
	Haircolor  int64            `json:"haircolor"`
	Skintone   int64            `json:"skintone"`
	Experience int64            `json:"experience"`
	GP         int64            `json:"gp"`
	Health     int64            `json:"health"`
	Mana       int64            `json:"mana"`
	Stamina    int64            `json:"stamina"`
	Karma      int64            `json:"karma"`
	Weight     int64            `json:"weight"`
	Inventory  []gmItemJSON      `json:"inventory"`
	Equipment  map[string]gmItemJSON `json:"equipment"`
	Attributes gmAttrJSON        `json:"attributes"`
}

type gmItemJSON struct {
	ItemID      int64  `json:"item_id"`
	Customfield string `json:"customfield"`
	Count       int64  `json:"count"`
	Name        string `json:"name,omitempty"`
}

type gmAttrJSON struct {
	Strength      int64 `json:"strength"`
	Vitality      int64 `json:"vitality"`
	Agility       int64 `json:"agility"`
	Endurance     int64 `json:"endurance"`
	Concentration int64 `json:"concentration"`
}

type gmPlayerJSON struct {
	PeerID  int     `json:"peerID"`
	CharID  int64   `json:"charID"`
	Name    string  `json:"name"`
	MapID   int     `json:"mapID"`
	PosX    float32 `json:"posX"`
	PosY    float32 `json:"posY"`
	Health  int64   `json:"health"`
	MaxHealth int64 `json:"maxHealth"`
	Level   int64   `json:"level"`
	IsNpc   bool    `json:"isNpc"`
}

type gmItemDefJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slot      int64  `json:"slot"`
	SlotName  string `json:"slot_name"`
	Stackable bool   `json:"stackable"`
	Usable    bool   `json:"usable"`
	BuyPrice  int64  `json:"buy_price"`
	SellPrice int64  `json:"sell_price"`
	Weight    int64  `json:"weight"`
}

var slotNames = map[int64]string{
	-1: "NONE",
	0:  "CHEST",
	1:  "LEGS",
	2:  "FEET",
	3:  "HANDS",
	4:  "HEAD",
	5:  "NECK",
	6:  "WEAPON",
	7:  "SHIELD",
	8:  "AMMUNITION",
	9:  "ACCESSORY1",
	10: "ACCESSORY2",
}

func itemToJSON(rec ItemRecord) gmItemJSON {
	j := gmItemJSON{
		ItemID:      rec.ItemID,
		Customfield: rec.Customfield,
		Count:       rec.Count,
	}
	if cell := itemCatalog[rec.ItemID]; cell != nil {
		j.Name = cell.Name
	}
	return j
}

func charToJSON(c *Character) gmCharacterJSON {
	inv := make([]gmItemJSON, len(c.Inventory))
	for i, rec := range c.Inventory {
		inv[i] = itemToJSON(rec)
	}
	eq := make(map[string]gmItemJSON)
	for slot, rec := range c.Equipment {
		name := slotNames[slot]
		if name == "" {
			name = strconv.FormatInt(slot, 10)
		}
		eq[name] = itemToJSON(rec)
	}
	return gmCharacterJSON{
		ID:         c.ID,
		Name:       c.Name,
		Level:      c.Level,
		Race:       c.Race,
		Gender:     c.Gender,
		Hairstyle:  c.Hairstyle,
		Haircolor:  c.Haircolor,
		Skintone:   c.Skintone,
		Experience: c.Experience,
		GP:         c.GP,
		Health:     c.Health,
		Mana:       c.Mana,
		Stamina:    c.Stamina,
		Karma:      c.Karma,
		Weight:     c.Weight,
		Inventory:  inv,
		Equipment:  eq,
		Attributes: gmAttrJSON{
			Strength:      c.Attributes.Strength,
			Vitality:      c.Attributes.Vitality,
			Agility:       c.Attributes.Agility,
			Endurance:     c.Attributes.Endurance,
			Concentration: c.Attributes.Concentration,
		},
	}
}

// ---- GM handlers ----

// gmHandlePage serves the embedded GM web page.
func gmHandlePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/gm/" && r.URL.Path != "/gm" {
		http.NotFound(w, r)
		return
	}
	data, err := gmFS.ReadFile("gm.html")
	if err != nil {
		http.Error(w, "GM page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// gmHandleStatus returns server status.
func (s *Server) gmHandleStatus(w http.ResponseWriter, r *http.Request) {
	players := s.gmOnlinePlayers()
	s.store.mu.Lock()
	accountCount := len(s.store.accounts)
	s.store.mu.Unlock()
	gmOK(w, map[string]interface{}{
		"uptime":        time.Since(gmStartTime).String(),
		"onlinePlayers": len(players),
		"accounts":      accountCount,
		"timestamp":     time.Now().Unix(),
	})
}

// gmHandleAccountsList returns all accounts.
func (s *Server) gmHandleAccountsList(w http.ResponseWriter, r *http.Request) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	names := make([]string, 0, len(s.store.accounts))
	for name := range s.store.accounts {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]gmAccountJSON, 0, len(names))
	for _, name := range names {
		acc := s.store.accounts[name]
		chars := make([]gmCharacterJSON, len(acc.Characters))
		for i, c := range acc.Characters {
			chars[i] = charToJSON(c)
		}
		out = append(out, gmAccountJSON{
			Name:       acc.Name,
			Permission: acc.Permission,
			Characters: chars,
		})
	}
	gmOK(w, out)
}

// gmHandleAccountCreate creates a new account.
func (s *Server) gmHandleAccountCreate(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	name := strings.TrimSpace(gmStrField(req, "name"))
	password := gmStrField(req, "password")
	permission := int(gmIntField(req, "permission", 0))

	if name == "" {
		gmError(w, http.StatusBadRequest, "name is required")
		return
	}
	if password == "" {
		gmError(w, http.StatusBadRequest, "password is required")
		return
	}
	if validateName(name, accountNameMinSize, accountNameMaxSize) != 0 {
		gmError(w, http.StatusBadRequest, "invalid account name")
		return
	}
	if permission < 0 || permission > 3 {
		gmError(w, http.StatusBadRequest, "permission must be 0-3")
		return
	}
	s.store.mu.Lock()
	if _, exists := s.store.accounts[name]; exists {
		s.store.mu.Unlock()
		gmError(w, http.StatusConflict, "account already exists")
		return
	}
	s.store.mu.Unlock()

	acc := s.store.Add(name, password)
	s.store.mu.Lock()
	acc.Permission = permission
	s.store.mu.Unlock()
	log.Printf("gm: account %q created (permission=%d)", name, permission)
	gmOK(w, map[string]interface{}{"name": name, "permission": permission})
}

// gmHandleAccountDelete deletes an account.
func (s *Server) gmHandleAccountDelete(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	name := strings.TrimSpace(gmStrField(req, "name"))
	if name == "" {
		gmError(w, http.StatusBadRequest, "name is required")
		return
	}
	s.store.mu.Lock()
	_, exists := s.store.accounts[name]
	s.store.mu.Unlock()
	if !exists {
		gmError(w, http.StatusNotFound, "account not found")
		return
	}
	// Kick any online characters belonging to this account.
	for _, e := range s.world.snapshot() {
		chr := s.store.findCharacterByID(e.CharID)
		if chr != nil {
			acc := s.store.findAccountByID(chr.ID)
			if acc != nil && acc.Name == name {
				if tc := s.clientByPeer(e.PeerID); tc != nil {
					tc.conn.Close()
				}
			}
		}
	}
	s.store.mu.Lock()
	delete(s.store.accounts, name)
	s.store.mu.Unlock()
	if s.store.db != nil {
		s.store.db.Exec(`DELETE FROM account WHERE username = ?`, name)
	}
	log.Printf("gm: account %q deleted", name)
	gmOK(w, map[string]interface{}{"name": name, "deleted": true})
}

// gmHandleAccountPerm sets an account's permission level.
func (s *Server) gmHandleAccountPerm(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	name := strings.TrimSpace(gmStrField(req, "name"))
	permission := int(gmIntField(req, "permission", -1))
	if name == "" {
		gmError(w, http.StatusBadRequest, "name is required")
		return
	}
	if permission < 0 || permission > 3 {
		gmError(w, http.StatusBadRequest, "permission must be 0-3")
		return
	}
	s.store.mu.Lock()
	acc, exists := s.store.accounts[name]
	if !exists {
		s.store.mu.Unlock()
		gmError(w, http.StatusNotFound, "account not found")
		return
	}
	acc.Permission = permission
	s.store.mu.Unlock()
	if s.store.db != nil {
		s.store.db.Exec(`UPDATE account SET flags = ? WHERE username = ?`, permission, name)
	}
	log.Printf("gm: account %q permission set to %d", name, permission)
	gmOK(w, map[string]interface{}{"name": name, "permission": permission})
}

// gmHandleCharactersList returns all characters across all accounts.
func (s *Server) gmHandleCharactersList(w http.ResponseWriter, r *http.Request) {
	s.store.mu.Lock()
	var out []gmCharacterJSON
	for _, acc := range s.store.accounts {
		for _, c := range acc.Characters {
			out = append(out, charToJSON(c))
		}
	}
	s.store.mu.Unlock()
	// Sort by ID for stable output.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	gmOK(w, out)
}

// gmHandleCharacterStats updates a character's stats.
func (s *Server) gmHandleCharacterStats(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	charID := gmIntField(req, "char_id", 0)
	if charID <= 0 {
		gmError(w, http.StatusBadRequest, "char_id is required")
		return
	}
	// findCharacterByID locks internally; don't hold the lock here.
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		gmError(w, http.StatusNotFound, "character not found")
		return
	}
	// Update only provided fields (use gmIntField for safe type conversion).
	s.store.mu.Lock()
	if _, ok := req["level"]; ok {
		chr.Level = gmIntField(req, "level", chr.Level)
	}
	if _, ok := req["gp"]; ok {
		chr.GP = gmIntField(req, "gp", chr.GP)
	}
	if _, ok := req["health"]; ok {
		chr.Health = gmIntField(req, "health", chr.Health)
	}
	if _, ok := req["mana"]; ok {
		chr.Mana = gmIntField(req, "mana", chr.Mana)
	}
	if _, ok := req["stamina"]; ok {
		chr.Stamina = gmIntField(req, "stamina", chr.Stamina)
	}
	if _, ok := req["experience"]; ok {
		chr.Experience = gmIntField(req, "experience", chr.Experience)
	}
	if _, ok := req["karma"]; ok {
		chr.Karma = gmIntField(req, "karma", chr.Karma)
	}
	// Attributes (optional nested object).
	if attrRaw, ok := req["attributes"]; ok {
		if attrMap, ok := attrRaw.(map[string]interface{}); ok {
			if _, ok := attrMap["strength"]; ok {
				chr.Attributes.Strength = gmIntField(attrMap, "strength", chr.Attributes.Strength)
			}
			if _, ok := attrMap["vitality"]; ok {
				chr.Attributes.Vitality = gmIntField(attrMap, "vitality", chr.Attributes.Vitality)
			}
			if _, ok := attrMap["agility"]; ok {
				chr.Attributes.Agility = gmIntField(attrMap, "agility", chr.Attributes.Agility)
			}
			if _, ok := attrMap["endurance"]; ok {
				chr.Attributes.Endurance = gmIntField(attrMap, "endurance", chr.Attributes.Endurance)
			}
			if _, ok := attrMap["concentration"]; ok {
				chr.Attributes.Concentration = gmIntField(attrMap, "concentration", chr.Attributes.Concentration)
			}
		}
	}
	out := charToJSON(chr)
	s.store.mu.Unlock()

	s.persist()
	log.Printf("gm: character %s (id=%d) stats updated", chr.Name, chr.ID)
	gmOK(w, out)
}

// gmHandleCharacterDelete deletes a character.
func (s *Server) gmHandleCharacterDelete(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	charID := gmIntField(req, "char_id", 0)
	if charID <= 0 {
		gmError(w, http.StatusBadRequest, "char_id is required")
		return
	}
	// findCharacterByID locks internally.
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		gmError(w, http.StatusNotFound, "character not found")
		return
	}
	// Find owning account and remove (RemoveCharacter locks internally).
	s.store.mu.Lock()
	var owner *Account
	for _, acc := range s.store.accounts {
		for _, c := range acc.Characters {
			if c.ID == charID {
				owner = acc
				break
			}
		}
		if owner != nil {
			break
		}
	}
	s.store.mu.Unlock()
	if owner != nil {
		s.store.RemoveCharacter(owner, chr.Name)
	}

	// Kick the character if online.
	if e := s.world.get(charID); e != nil {
		if tc := s.clientByPeer(e.PeerID); tc != nil {
			tc.conn.Close()
		}
		s.world.remove(charID)
	}
	log.Printf("gm: character %s (id=%d) deleted", chr.Name, charID)
	gmOK(w, map[string]interface{}{"char_id": charID, "deleted": true})
}

// gmOnlinePlayers builds the online player list.
func (s *Server) gmOnlinePlayers() []gmPlayerJSON {
	var out []gmPlayerJSON
	for _, e := range s.world.snapshot() {
		isNpc := e.CharID < 0
		out = append(out, gmPlayerJSON{
			PeerID:    e.PeerID,
			CharID:    e.CharID,
			Name:      e.Name,
			MapID:     e.MapID,
			PosX:      e.Pos.X,
			PosY:      e.Pos.Y,
			Health:    e.Health,
			MaxHealth: e.MaxHealth,
			Level:     e.Level,
			IsNpc:     isNpc,
		})
	}
	return out
}

// gmHandlePlayersList returns online players.
func (s *Server) gmHandlePlayersList(w http.ResponseWriter, r *http.Request) {
	gmOK(w, s.gmOnlinePlayers())
}

// gmHandlePlayerKick kicks a player by name.
func (s *Server) gmHandlePlayerKick(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	name := strings.TrimSpace(gmStrField(req, "name"))
	if name == "" {
		gmError(w, http.StatusBadRequest, "name is required")
		return
	}
	target := s.world.byName(name)
	if target == nil {
		gmError(w, http.StatusNotFound, "player is not online")
		return
	}
	if tc := s.clientByPeer(target.PeerID); tc != nil {
		tc.conn.Close()
	}
	log.Printf("gm: player %q kicked", name)
	gmOK(w, map[string]interface{}{"name": name, "kicked": true})
}

// gmHandlePlayerBan bans a player by name.
func (s *Server) gmHandlePlayerBan(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	name := strings.TrimSpace(gmStrField(req, "name"))
	reason := gmStrField(req, "reason")
	if name == "" {
		gmError(w, http.StatusBadRequest, "name is required")
		return
	}
	// Find the character and its account.
	var charID int64 = -1
	for _, e := range s.world.snapshot() {
		if e.Name == name {
			charID = e.CharID
			break
		}
	}
	if charID >= 0 {
		chr := s.store.findCharacterByID(charID)
		if chr != nil {
			acc := s.store.findAccountByID(chr.ID)
			if acc != nil {
				// Set permission to -1 (banned).
				s.store.mu.Lock()
				acc.Permission = -1
				s.store.mu.Unlock()
				if s.store.db != nil {
					s.store.db.Exec(`UPDATE account SET flags = ? WHERE username = ?`, -1, acc.Name)
				}
			}
		}
	}
	// Kick the player.
	target := s.world.byName(name)
	if target != nil {
		if tc := s.clientByPeer(target.PeerID); tc != nil {
			tc.conn.Close()
		}
	}
	log.Printf("gm: player %q banned (reason: %s)", name, reason)
	gmOK(w, map[string]interface{}{"name": name, "banned": true})
}

// gmHandleItemList returns the item catalog.
func (s *Server) gmHandleItemList(w http.ResponseWriter, r *http.Request) {
	var out []gmItemDefJSON
	for _, cell := range itemCatalog {
		out = append(out, gmItemDefJSON{
			ID:        cell.ID,
			Name:      cell.Name,
			Slot:      cell.Slot,
			SlotName:  slotNames[cell.Slot],
			Stackable: cell.Stackable,
			Usable:    cell.Usable,
			BuyPrice:  cell.BuyPrice,
			SellPrice: cell.SellPrice,
			Weight:    cell.Weight,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	gmOK(w, out)
}

// gmHandleItemGive gives an item to a character.
func (s *Server) gmHandleItemGive(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	charID := gmIntField(req, "char_id", 0)
	itemID := gmIntField(req, "item_id", 0)
	count := gmIntField(req, "count", 1)
	if charID <= 0 || itemID <= 0 {
		gmError(w, http.StatusBadRequest, "char_id and item_id are required")
		return
	}
	if count <= 0 {
		gmError(w, http.StatusBadRequest, "count must be positive")
		return
	}
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		gmError(w, http.StatusNotFound, "character not found")
		return
	}
	s.store.mu.Lock()
	cell := itemCatalog[itemID]
	if cell != nil && cell.Stackable {
		added := false
		for i := range chr.Inventory {
			if chr.Inventory[i].ItemID == itemID && chr.Inventory[i].Customfield == "" {
				chr.Inventory[i].Count += count
				added = true
				break
			}
		}
		if !added {
			chr.Inventory = append(chr.Inventory, ItemRecord{ItemID: itemID, Customfield: "", Count: count})
		}
	} else {
		for i := int64(0); i < count; i++ {
			chr.Inventory = append(chr.Inventory, ItemRecord{ItemID: itemID, Customfield: "", Count: 1})
		}
	}
	s.store.mu.Unlock()

	s.persist()
	// Push ItemAdded to the character if online.
	if e := s.world.get(charID); e != nil {
		if tc := s.clientByPeer(e.PeerID); tc != nil {
			sendToClient(tc, nil, rpcItemAdded, []Variant{NewInt(itemID), NewStringName(""), NewInt(count)})
		}
	}
	log.Printf("gm: gave item %d x%d to character %s", itemID, count, chr.Name)
	gmOK(w, map[string]interface{}{"char_id": charID, "item_id": itemID, "count": count})
}

// gmHandleItemRemove removes an item from a character.
func (s *Server) gmHandleItemRemove(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	charID := gmIntField(req, "char_id", 0)
	itemID := gmIntField(req, "item_id", 0)
	count := gmIntField(req, "count", 1)
	if charID <= 0 || itemID <= 0 {
		gmError(w, http.StatusBadRequest, "char_id and item_id are required")
		return
	}
	if count <= 0 {
		gmError(w, http.StatusBadRequest, "count must be positive")
		return
	}
	chr := s.store.findCharacterByID(charID)
	if chr == nil {
		gmError(w, http.StatusNotFound, "character not found")
		return
	}
	s.store.mu.Lock()
	removeItem(chr, itemID, "", count)
	s.store.mu.Unlock()

	s.persist()
	// Push ItemRemoved to the character if online.
	if e := s.world.get(charID); e != nil {
		if tc := s.clientByPeer(e.PeerID); tc != nil {
			sendToClient(tc, nil, rpcItemRemoved, []Variant{NewInt(itemID), NewStringName(""), NewInt(count), NewInt(-1)})
		}
	}
	log.Printf("gm: removed item %d x%d from character %s", itemID, count, chr.Name)
	gmOK(w, map[string]interface{}{"char_id": charID, "item_id": itemID, "count": count})
}

// gmHandleCommand executes a GM command as the server (no player entity).
// The command is dispatched through the same path as in-game commands but
// with a synthetic caller. Commands that require a player entity will
// return an error.
func (s *Server) gmHandleCommand(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	command := strings.TrimSpace(gmStrField(req, "command"))
	if command == "" {
		gmError(w, http.StatusBadRequest, "command is required")
		return
	}
	parts := strings.Fields(command)
	name := strings.ToLower(parts[0])
	args := parts[1:]

	// Server-side commands that don't need a player entity.
	switch name {
	case "broadcast":
		if len(args) < 1 {
			gmError(w, http.StatusBadRequest, "usage: broadcast <message>")
			return
		}
		msg := strings.Join(args, " ")
		s.gmBroadcastAll(msg)
		gmOK(w, map[string]interface{}{"command": command, "result": "broadcast sent"})
		return
	case "list":
		players := s.gmOnlinePlayers()
		gmOK(w, map[string]interface{}{"command": command, "players": players})
		return
	case "online":
		players := s.gmOnlinePlayers()
		gmOK(w, map[string]interface{}{"command": command, "players": players})
		return
	case "kick":
		if len(args) < 1 {
			gmError(w, http.StatusBadRequest, "usage: kick <player_name>")
			return
		}
		target := s.world.byName(args[0])
		if target == nil {
			gmError(w, http.StatusNotFound, "player is not online")
			return
		}
		if tc := s.clientByPeer(target.PeerID); tc != nil {
			tc.conn.Close()
		}
		gmOK(w, map[string]interface{}{"command": command, "result": fmt.Sprintf("kicked %s", args[0])})
		return
	case "ban":
		if len(args) < 1 {
			gmError(w, http.StatusBadRequest, "usage: ban <player_name> [reason]")
			return
		}
		// Find character and ban its account.
		target := s.world.byName(args[0])
		if target != nil {
			chr := s.store.findCharacterByID(target.CharID)
			if chr != nil {
				acc := s.store.findAccountByID(chr.ID)
				if acc != nil {
					s.store.mu.Lock()
					acc.Permission = -1
					s.store.mu.Unlock()
					if s.store.db != nil {
						s.store.db.Exec(`UPDATE account SET flags = ? WHERE username = ?`, -1, acc.Name)
					}
				}
			}
			if tc := s.clientByPeer(target.PeerID); tc != nil {
				tc.conn.Close()
			}
		}
		gmOK(w, map[string]interface{}{"command": command, "result": fmt.Sprintf("banned %s", args[0])})
		return
	case "spawn":
		gmError(w, http.StatusBadRequest, "spawn requires a player entity; use the in-game GM command")
		return
	case "warp", "jump", "goto", "recall", "godmode", "hide", "invisible",
		"health", "mana", "stamina", "speed", "item", "skill",
		"level", "experience", "gp", "stat", "setstat",
		"quest", "bestiary", "killall", "kill", "revive",
		"permission", "ipcheck", "ipban", "ipunban", "deploy",
		"recallnpc", "disablenpc", "banlist", "ipbanlist":
		gmError(w, http.StatusBadRequest, fmt.Sprintf("command '%s' requires a player entity; use the in-game GM command", name))
		return
	default:
		gmError(w, http.StatusBadRequest, fmt.Sprintf("unknown GM command: %s", name))
		return
	}
}

// gmBroadcastAll sends a ChatSystem message to all online players.
func (s *Server) gmBroadcastAll(msg string) {
	for _, e := range s.world.snapshot() {
		if e.CharID < 0 {
			continue // skip NPCs
		}
		if tc := s.clientByPeer(e.PeerID); tc != nil {
			sendToClient(tc, nil, rpcChatSystem, []Variant{NewString(msg), NewInt(0)})
		}
	}
	log.Printf("gm: broadcast: %s", msg)
}

// gmHandleBroadcast sends a message to all online players.
func (s *Server) gmHandleBroadcast(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := gmDecode(r, &req); err != nil {
		gmError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	msg := strings.TrimSpace(gmStrField(req, "message"))
	if msg == "" {
		gmError(w, http.StatusBadRequest, "message is required")
		return
	}
	s.gmBroadcastAll(msg)
	gmOK(w, map[string]interface{}{"message": msg, "recipients": len(s.gmOnlinePlayers())})
}

// ---- routing ----

// gmRoutes registers the GM HTTP endpoints on the mux.
func (s *Server) gmRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/gm/", gmHandlePage)
	mux.HandleFunc("/gm", gmHandlePage)

	mux.HandleFunc("/gm/api/status", s.gmHandleStatus)
	mux.HandleFunc("/gm/api/accounts", s.gmHandleAccountsList)
	mux.HandleFunc("/gm/api/accounts/create", s.gmHandleAccountCreate)
	mux.HandleFunc("/gm/api/accounts/delete", s.gmHandleAccountDelete)
	mux.HandleFunc("/gm/api/accounts/perm", s.gmHandleAccountPerm)
	mux.HandleFunc("/gm/api/characters", s.gmHandleCharactersList)
	mux.HandleFunc("/gm/api/characters/stats", s.gmHandleCharacterStats)
	mux.HandleFunc("/gm/api/characters/delete", s.gmHandleCharacterDelete)
	mux.HandleFunc("/gm/api/players", s.gmHandlePlayersList)
	mux.HandleFunc("/gm/api/players/kick", s.gmHandlePlayerKick)
	mux.HandleFunc("/gm/api/players/ban", s.gmHandlePlayerBan)
	mux.HandleFunc("/gm/api/items", s.gmHandleItemList)
	mux.HandleFunc("/gm/api/items/give", s.gmHandleItemGive)
	mux.HandleFunc("/gm/api/items/remove", s.gmHandleItemRemove)
	mux.HandleFunc("/gm/api/command", s.gmHandleCommand)
	mux.HandleFunc("/gm/api/broadcast", s.gmHandleBroadcast)
}
