package session

import "testing"

func TestInMemoryStoreCopiesAndRotatesByToken(t *testing.T) {
	store := NewInMemoryStore()
	original := &SessionInfo{Token: "old", UserAgent: &UserAgentPayload{AppVersion: "1"}}
	if err := store.SaveSession(original); err != nil {
		t.Fatal(err)
	}
	original.Token = "mutated"
	loaded, _ := store.LoadSession()
	if loaded.Token != "old" {
		t.Fatalf("store aliased caller: %q", loaded.Token)
	}
	loaded.Token = "also-mutated"
	reloaded, _ := store.LoadSession()
	if reloaded.Token != "old" {
		t.Fatalf("load returned internal pointer: %q", reloaded.Token)
	}
	if err := store.UpdateToken("wrong", "new"); err == nil {
		t.Fatal("expected token mismatch")
	}
	if err := store.UpdateToken("old", "new"); err != nil {
		t.Fatal(err)
	}
	reloaded, _ = store.LoadSession()
	if reloaded.Token != "new" {
		t.Fatalf("token=%q", reloaded.Token)
	}
}
