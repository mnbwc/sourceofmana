package main

// Phase-3 persistence: the AccountStore is mirrored to a JSON file so that
// accounts, characters, and their traits survive a server restart. Writes are
// serialized through the store's own mutex and are atomic (write to a temp
// file, then rename), so a crash mid-write cannot corrupt the live database.
//
// The file format is deliberately simple:
//
//	{
//	  "nextCharID": 7,
//	  "accounts": [
//	    {"name": "alice", "password": "secret1", "token": "...",
//	     "characters": [{"id": 3, "name": "Alica", "level": 1, "race": 1, ...}]}
//	  ]
//	}

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// storeFileDefault is used when SOM_STORE_FILE is not set.
const storeFileDefault = "som_store.json"

// storeFile returns the configured persistence path.
func storeFile() string {
	if p := os.Getenv("SOM_STORE_FILE"); p != "" {
		return p
	}
	return storeFileDefault
}

// ---- wire format ----

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

// loadStore reads the store from disk. A missing file is not an error: it
// yields an empty store, matching the pre-persistence behaviour.
func loadStore() *AccountStore {
	s := NewAccountStore()
	path := storeFile()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("store: no file at %s, starting empty", path)
			return s
		}
		log.Printf("store: read %s: %v (starting empty)", path, err)
		return s
	}
	var f storeFileFormat
	if err := json.Unmarshal(data, &f); err != nil {
		log.Printf("store: parse %s: %v (starting empty)", path, err)
		return s
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextCharID = f.NextCharID
	if s.nextCharID < 1 {
		s.nextCharID = 1
	}
	for _, fa := range f.Accounts {
		acc := &Account{Name: fa.Name, Password: fa.Password, Token: fa.Token, Permission: fa.Permission}
		for _, fc := range fa.Characters {
			c := &Character{
				ID: fc.ID, Name: fc.Name, Level: fc.Level, Race: fc.Race,
				Gender: fc.Gender, Hairstyle: fc.Hairstyle,
				Haircolor: fc.Haircolor, Skintone: fc.Skintone,
				Experience: fc.Experience, GP: fc.GP,
				Health: fc.Health, Mana: fc.Mana, Stamina: fc.Stamina,
				Karma: fc.Karma, Weight: fc.Weight,
				Inventory: []ItemRecord{},
				Equipment: map[int64]ItemRecord{},
			}
			for _, it := range fc.Inventory {
				c.Inventory = append(c.Inventory, ItemRecord{
					ItemID: it.ItemID, Customfield: it.Customfield, Count: it.Count,
				})
			}
			for _, eq := range fc.Equipment {
				c.Equipment[eq.Slot] = ItemRecord{
					ItemID: eq.ItemID, Customfield: eq.Customfield,
				}
			}
			acc.Characters = append(acc.Characters, c)
		}
		s.accounts[fa.Name] = acc
	}
	log.Printf("store: loaded %d account(s) from %s", len(s.accounts), path)
	return s
}

// ---- save ----

// saveLocked writes the store to disk. Callers must hold s.mu.
func (s *AccountStore) saveLocked() {
	path := storeFile()
	f := storeFileFormat{NextCharID: s.nextCharID}
	// Sort account names so the file is deterministic across runs.
	names := make([]string, 0, len(s.accounts))
	for n := range s.accounts {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		acc := s.accounts[n]
		fa := storeFileAccount{Name: acc.Name, Password: acc.Password, Token: acc.Token, Permission: acc.Permission}
		for _, c := range acc.Characters {
			fc := storeFileCharacter{
				ID: c.ID, Name: c.Name, Level: c.Level, Race: c.Race,
				Gender: c.Gender, Hairstyle: c.Hairstyle,
				Haircolor: c.Haircolor, Skintone: c.Skintone,
				Experience: c.Experience, GP: c.GP,
				Health: c.Health, Mana: c.Mana, Stamina: c.Stamina,
				Karma: c.Karma, Weight: c.Weight,
				Inventory: []storeFileItem{},
				Equipment: []storeFileEquipment{},
			}
			for _, it := range c.Inventory {
				fc.Inventory = append(fc.Inventory, storeFileItem{
					ItemID: it.ItemID, Customfield: it.Customfield, Count: it.Count,
				})
			}
			for slot, eq := range c.Equipment {
				fc.Equipment = append(fc.Equipment, storeFileEquipment{
					Slot: slot, ItemID: eq.ItemID, Customfield: eq.Customfield,
				})
			}
			fa.Characters = append(fa.Characters, fc)
		}
		f.Accounts = append(f.Accounts, fa)
	}
	data, err := json.MarshalIndent(&f, "", "  ")
	if err != nil {
		log.Printf("store: marshal: %v", err)
		return
	}
	// Atomic write: temp file in the same directory, then rename.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		log.Printf("store: temp file: %v", err)
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		log.Printf("store: write temp: %v", err)
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		log.Printf("store: close temp: %v", err)
		os.Remove(tmpName)
		return
	}
	if err := os.Rename(tmpName, path); err != nil {
		log.Printf("store: rename %s -> %s: %v", tmpName, path, err)
		os.Remove(tmpName)
	}
}

// ---- mutation hooks ----

// Add persists after creating an account.
func (s *AccountStore) Add(name, password string) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc := &Account{Name: name, Password: password}
	s.accounts[name] = acc
	s.saveLocked()
	return acc
}

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
		Inventory:  []ItemRecord{},
		Equipment:  map[int64]ItemRecord{},
	}
	acc.Characters = append(acc.Characters, c)
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
	s.saveLocked()
}
