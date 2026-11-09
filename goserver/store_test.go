package main

// Phase-3 persistence tests: verify the AccountStore survives a restart by
// round-tripping through the JSON file on disk.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStorePersistenceRoundTrip writes a store to a temp file, loads it into a
// fresh store, and checks that accounts, characters, tokens, and the char-ID
// counter all survive.
func TestStorePersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	t.Setenv("SOM_STORE_FILE", path)

	s := NewAccountStore()
	acc := s.Add("alice", "secret1")
	s.SetToken(acc, "tok-alice")
	s.AddCharacter(acc, "Alica", map[string]Variant{
		"race": NewInt(1), "gender": NewInt(0), "hairstyle": NewInt(3),
	})
	acc2 := s.Add("bob", "hunter2")
	s.AddCharacter(acc2, "Bobby", map[string]Variant{"race": NewInt(2)})

	// Load into a fresh store from disk.
	got := loadStore()
	if got == nil {
		t.Fatal("loadStore returned nil")
	}
	if len(got.accounts) != 2 {
		t.Fatalf("want 2 accounts, got %d", len(got.accounts))
	}
	ga := got.Get("alice")
	if ga == nil {
		t.Fatal("alice missing after reload")
	}
	if ga.Password != "secret1" {
		t.Errorf("password: want secret1, got %q", ga.Password)
	}
	if ga.Token != "tok-alice" {
		t.Errorf("token: want tok-alice, got %q", ga.Token)
	}
	if len(ga.Characters) != 1 {
		t.Fatalf("alice characters: want 1, got %d", len(ga.Characters))
	}
	c := ga.Characters[0]
	if c.Name != "Alica" || c.Race != 1 || c.Hairstyle != 3 {
		t.Errorf("character mismatch: %+v", c)
	}
	// The char-ID counter must continue after the highest persisted ID.
	next := got.AddCharacter(ga, "Alica2", map[string]Variant{})
	if next.ID <= c.ID {
		t.Errorf("nextCharID not restored: new id %d <= previous %d", next.ID, c.ID)
	}
	// Bob's character survived too.
	gb := got.Get("bob")
	if gb == nil || len(gb.Characters) != 1 || gb.Characters[0].Name != "Bobby" {
		t.Errorf("bob/characters lost: %+v", gb)
	}
}

// TestStorePersistenceRemove verifies RemoveCharacter persists.
func TestStorePersistenceRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	t.Setenv("SOM_STORE_FILE", path)

	s := NewAccountStore()
	acc := s.Add("carol", "pw")
	s.AddCharacter(acc, "Cara", map[string]Variant{})
	if !s.RemoveCharacter(acc, "Cara") {
		t.Fatal("RemoveCharacter failed")
	}

	got := loadStore()
	ga := got.Get("carol")
	if ga == nil {
		t.Fatal("carol missing after reload")
	}
	if len(ga.Characters) != 0 {
		t.Errorf("characters after remove: want 0, got %d", len(ga.Characters))
	}
}

// TestStorePersistenceMissingFile verifies a missing file yields an empty store.
func TestStorePersistenceMissingFile(t *testing.T) {
	t.Setenv("SOM_STORE_FILE", filepath.Join(t.TempDir(), "does-not-exist.json"))
	s := loadStore()
	if s == nil || len(s.accounts) != 0 {
		t.Fatalf("missing file: want empty store, got %+v", s)
	}
	// A subsequent save must create the file.
	s.Add("dave", "pw")
	if _, err := os.Stat(storeFile()); err != nil {
		t.Errorf("save after load: %v", err)
	}
}
