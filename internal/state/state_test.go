package state_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/state"
)

func TestMarkAndCheck(t *testing.T) {
	st, _ := state.Load(filepath.Join(t.TempDir(), "state.json"))
	const id = "prism/Survival/2024-01-15_12.30.00.png"
	if st.IsUploaded(id) {
		t.Fatal("should not be uploaded before marking")
	}
	if err := st.MarkUploaded(id, "asset-abc"); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	if !st.IsUploaded(id) {
		t.Fatal("should be uploaded after marking")
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, _ := state.Load(path)
	st.MarkUploaded("prism/World/a.png", "id1")

	st2, err := state.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !st2.IsUploaded("prism/World/a.png") {
		t.Fatal("state not persisted")
	}
}

func TestMissingFileStartsEmpty(t *testing.T) {
	st, err := state.Load(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if st.IsUploaded("x") {
		t.Fatal("fresh state should be empty")
	}
}

func TestCorruptFileStartsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	os.WriteFile(path, []byte("not json {{{"), 0644)
	st, err := state.Load(path)
	if err != nil {
		t.Fatalf("corrupt state should not error: %v", err)
	}
	if st.IsUploaded("x") {
		t.Fatal("corrupt state should start empty")
	}
}
