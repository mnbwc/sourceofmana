package main

import (
	"net"
	"net/http"
	"testing"
)

// TestPersistenceSurvivesRestart verifies that an account created on one
// server instance is visible to a second instance sharing the same store file.
func TestPersistenceSurvivesRestart(t *testing.T) {
	path := t.TempDir() + "/store.json"
	t.Setenv("SOM_STORE_FILE", path)

	// Instance 1: create the account and a character.
	s1 := NewServer()
	ln1, _ := net.Listen("tcp", "127.0.0.1:0")
	srv1 := &http.Server{Handler: s1.routes()}
	go srv1.Serve(ln1)
	defer srv1.Close()
	acc := s1.store.Add("persist_user", "pw123456")
	s1.store.AddCharacter(acc, "PersistChar", map[string]Variant{"race": NewInt(1), "gender": NewInt(0)})

	// Instance 2: same store file, must see the account.
	s2 := NewServer()
	if got := s2.store.Get("persist_user"); got == nil {
		t.Fatal("account not persisted across restart")
	} else if got.Password != "pw123456" {
		t.Errorf("password mismatch: %q", got.Password)
	}
	if c := s2.store.findCharacter(s2.store.Get("persist_user"), "PersistChar"); c == nil {
		t.Fatal("character not persisted across restart")
	} else if c.Race != 1 {
		t.Errorf("race mismatch: %d", c.Race)
	}
}
