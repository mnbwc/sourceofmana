package main

// Phase-3 persistence: the AccountStore is backed by SQLite (see
// sqlite_store.go for the schema and write-through helpers). The in-memory
// cache is still the fast path for reads; SQLite is the source of truth.
//
// On startup, loadStore() opens the SQLite database (SOM_DB_FILE, default
// som.db). If the database is empty and a legacy JSON file exists
// (SOM_STORE_FILE, default som_store.json), the JSON data is imported once.
//
// saveLocked() is a no-op: SQLite writes are immediate and atomic.

import (
	"log"
	"os"
)

// storeFileDefault is used when SOM_STORE_FILE is not set. It is only used
// for the one-time JSON -> SQLite migration.
const storeFileDefault = "som_store.json"

// storeFile returns the configured JSON migration path.
func storeFile() string {
	if p := os.Getenv("SOM_STORE_FILE"); p != "" {
		return p
	}
	return storeFileDefault
}

// ---- JSON wire format (for migration only) ----

type storeFileFormat struct {
	NextCharID int64              `json:"nextCharID"`
	Accounts   []storeFileAccount `json:"accounts"`
}

type storeFileAccount struct {
	Name       string               `json:"name"`
	Password   string               `json:"password"`
	Token      string               `json:"token"`
	Permission int                  `json:"permission"`
	Characters []storeFileCharacter `json:"characters"`
}

type storeFileCharacter struct {
	ID         int64                `json:"id"`
	Name       string               `json:"name"`
	Level      int64                `json:"level"`
	Race       int64                `json:"race"`
	Gender     int64                `json:"gender"`
	Hairstyle  int64                `json:"hairstyle"`
	Haircolor  int64                `json:"haircolor"`
	Skintone   int64                `json:"skintone"`
	Experience int64                `json:"experience"`
	GP         int64                `json:"gp"`
	Health     int64                `json:"health"`
	Mana       int64                `json:"mana"`
	Stamina    int64                `json:"stamina"`
	Karma      int64                `json:"karma"`
	Weight     int64                `json:"weight"`
	Inventory  []storeFileItem      `json:"inventory"`
	Equipment  []storeFileEquipment `json:"equipment"`
}

type storeFileItem struct {
	ItemID      int64  `json:"item_id"`
	Customfield string `json:"customfield"`
	Count       int64  `json:"count"`
}

type storeFileEquipment struct {
	Slot        int64  `json:"slot"`
	ItemID      int64  `json:"item_id"`
	Customfield string `json:"customfield"`
}

// ---- load ----

// loadStore opens the SQLite database and loads all accounts into memory.
// If the database is empty and a legacy JSON file exists, it is imported
// first. A missing database is not an error: it yields an empty store.
func loadStore() *AccountStore {
	path := dbFile()
	db, err := openSQLite(path)
	if err != nil {
		log.Printf("store: open sqlite %s: %v (starting empty)", path, err)
		return NewAccountStore()
	}

	// One-time migration: if the DB is empty, try importing from JSON.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM account`).Scan(&count); err != nil {
		log.Printf("store: count accounts: %v", err)
	} else if count == 0 {
		jsonPath := storeFile()
		if _, err := os.Stat(jsonPath); err == nil {
			log.Printf("store: importing from %s into %s", jsonPath, path)
			if err := importFromJSON(db, jsonPath); err != nil {
				log.Printf("store: import from %s: %v", jsonPath, err)
			}
		}
	}

	s, err := loadFromSQLite(db)
	if err != nil {
		log.Printf("store: load from sqlite: %v (starting empty)", err)
		return NewAccountStore()
	}
	log.Printf("store: loaded %d account(s) from %s", len(s.accounts), path)
	return s
}

// ---- save ----

// saveLocked is a no-op: SQLite writes are immediate and atomic. It is kept
// for API compatibility with existing callers.
func (s *AccountStore) saveLocked() {
	// SQLite writes are immediate; no batching needed.
}

// ---- mutation hooks ----

// Add persists after creating an account.
func (s *AccountStore) Add(name, password string) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := &Account{Name: name, Password: password}
	s.accounts[name] = acc
	s.dbAddAccount(acc)
	s.saveLocked()
	return acc
}

// DefaultInventoryItemID is the item hash for "Short Sword" — given to every
// new character so they don't spawn empty-handed.
// Computed as "Short Sword".hash() in Godot (3056796356).
const DefaultInventoryItemID int64 = 3056796356

// AddCharacter persists after adding a character.
func (s *AccountStore) AddCharacter(acc *Account, name string, traits map[string]Variant) *Character {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextCharID
	s.nextCharID++
	c := &Character{
		ID:         id,
		Name:       name,
		Level:      0, // SoM Stats.level starts at 0 (0-based)
		Race:       dictInt(traits, "race"),
		Gender:     dictInt(traits, "gender"),
		Hairstyle:  dictInt(traits, "hairstyle"),
		Haircolor:  dictInt(traits, "haircolor"),
		Skintone:   dictInt(traits, "skintone"),
		Experience: 0,
		GP:         0,
		Health:     100,
		Mana:       50,
		Stamina:    100,
		Karma:      0,
		Weight:     0,
		Inventory: []ItemRecord{
			{ItemID: DefaultInventoryItemID, Customfield: "", Count: 1},
		},
		Equipment: map[int64]ItemRecord{},
	}
	acc.Characters = append(acc.Characters, c)
	s.dbAddCharacter(acc, c)
	s.saveLocked()
	return c
}

// RemoveCharacter persists after removing a character.
func (s *AccountStore) RemoveCharacter(acc *Account, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range acc.Characters {
		if c.Name == name {
			acc.Characters = append(acc.Characters[:i], acc.Characters[i+1:]...)
			s.dbRemoveCharacter(c)
			s.saveLocked()
			return true
		}
	}
	return false
}

// SetToken persists a newly issued remember-me token.
func (s *AccountStore) SetToken(acc *Account, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc.Token = token
	s.dbSetToken(acc, token)
	s.saveLocked()
}
