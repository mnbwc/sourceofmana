package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer() *Server {
	s := NewServer()
	// Replace the store with a fresh in-memory one (no SQLite).
	s.store = NewAccountStore()
	return s
}

func TestGMAPI_AccountCRUD(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	// Create account
	body := `{"name":"gmtest","password":"pass123","permission":2}`
	req := httptest.NewRequest("POST", "/gm/api/accounts/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("create account: status %d, body %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("create account failed: %v", resp)
	}

	// List accounts
	req = httptest.NewRequest("GET", "/gm/api/accounts", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 account, got %d", len(data))
	}
	acc := data[0].(map[string]interface{})
	if acc["name"] != "gmtest" {
		t.Fatalf("expected name gmtest, got %v", acc["name"])
	}

	// Set permission
	body = `{"name":"gmtest","permission":3}`
	req = httptest.NewRequest("POST", "/gm/api/accounts/perm", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("set perm failed: %v", resp)
	}

	// Delete account
	body = `{"name":"gmtest"}`
	req = httptest.NewRequest("POST", "/gm/api/accounts/delete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("delete failed: %v", resp)
	}

	// Verify deletion
	req = httptest.NewRequest("GET", "/gm/api/accounts", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].([]interface{})
	if len(data) != 0 {
		t.Fatalf("expected 0 accounts after delete, got %d", len(data))
	}
}

func TestGMAPI_ItemGiveRemove(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	// Create account + character directly in store.
	s.store.Add("itemtest", "pass123")
	acc := s.store.Get("itemtest")
	s.store.AddCharacter(acc, "ItemChar", map[string]Variant{})

	// Get character ID (findCharacterByID locks internally).
	chr := s.store.findCharacterByID(1)
	if chr == nil {
		t.Fatal("character not found")
	}

	// Give item
	body := `{"char_id":1,"item_id":1001,"count":5}`
	req := httptest.NewRequest("POST", "/gm/api/items/give", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("give item failed: %v", resp)
	}

	// Verify item in inventory (findCharacterByID locks internally).
	chr = s.store.findCharacterByID(1)
	found := false
	for _, rec := range chr.Inventory {
		if rec.ItemID == 1001 && rec.Count == 5 {
			found = true
		}
	}
	if !found {
		t.Fatalf("item not found in inventory")
	}

	// Remove item
	body = `{"char_id":1,"item_id":1001,"count":2}`
	req = httptest.NewRequest("POST", "/gm/api/items/remove", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("remove item failed: %v", resp)
	}

	// Verify item count reduced.
	chr = s.store.findCharacterByID(1)
	for _, rec := range chr.Inventory {
		if rec.ItemID == 1001 && rec.Count != 3 {
			t.Fatalf("expected count 3, got %d", rec.Count)
		}
	}
}

func TestGMAPI_Broadcast(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	body := `{"message":"test broadcast"}`
	req := httptest.NewRequest("POST", "/gm/api/broadcast", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("broadcast failed: %v", resp)
	}
}

func TestGMAPI_Command(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	// Test "list" command.
	body := `{"command":"list"}`
	req := httptest.NewRequest("POST", "/gm/api/command", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("command list failed: %v", resp)
	}

	// Test invalid command.
	body = `{"command":"invalidcmd"}`
	req = httptest.NewRequest("POST", "/gm/api/command", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid command, got %d", w.Code)
	}
}

func TestGMAPI_Status(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	req := httptest.NewRequest("GET", "/gm/api/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("status failed: %v", resp)
	}
	data := resp["data"].(map[string]interface{})
	if _, ok := data["uptime"]; !ok {
		t.Fatalf("status missing uptime")
	}
	if _, ok := data["onlinePlayers"]; !ok {
		t.Fatalf("status missing onlinePlayers")
	}
}

func TestGMAPI_Page(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	req := httptest.NewRequest("GET", "/gm/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("SoM GM Panel")) {
		t.Fatalf("GM page missing title")
	}
}

func TestGMAPI_CharacterStats(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	// Create account + character.
	s.store.Add("stattest", "pass123")
	acc := s.store.Get("stattest")
	s.store.AddCharacter(acc, "StatChar", map[string]Variant{})

	// Update stats.
	body := `{"char_id":1,"level":10,"gp":5000,"health":200,"mana":100,"stamina":150,"experience":1000,"karma":50}`
	req := httptest.NewRequest("POST", "/gm/api/characters/stats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("update stats failed: %v", resp)
	}

	// Verify.
	chr := s.store.findCharacterByID(1)
	if chr.Level != 10 || chr.GP != 5000 || chr.Health != 200 {
		t.Fatalf("stats not updated: level=%d gp=%d health=%d", chr.Level, chr.GP, chr.Health)
	}
}

func TestGMAPI_CharacterDelete(t *testing.T) {
	s := newTestServer()
	mux := http.NewServeMux()
	s.gmRoutes(mux)

	// Create account + character.
	s.store.Add("deltest", "pass123")
	acc := s.store.Get("deltest")
	s.store.AddCharacter(acc, "DelChar", map[string]Variant{})

	// Delete character.
	body := `{"char_id":1}`
	req := httptest.NewRequest("POST", "/gm/api/characters/delete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["ok"].(bool) {
		t.Fatalf("delete character failed: %v", resp)
	}

	// Verify.
	chr := s.store.findCharacterByID(1)
	if chr != nil {
		t.Fatalf("character still exists")
	}
}
