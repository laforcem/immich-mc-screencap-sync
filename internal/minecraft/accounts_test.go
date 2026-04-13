package minecraft_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/minecraft"
)

func writePrismAccounts(t *testing.T, dir string, accounts []map[string]any, activeLocalId string) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{
		"accounts":             accounts,
		"activeAccountLocalId": activeLocalId,
	})
	os.WriteFile(filepath.Join(dir, "accounts.json"), data, 0644)
}

func TestReadPrismAccount(t *testing.T) {
	dir := t.TempDir()
	writePrismAccounts(t, dir, []map[string]any{
		{
			"localId": "local-1",
			"profile": map[string]any{"name": "Steve"},
		},
	}, "local-1")

	name, err := minecraft.ReadPrismAccount(dir)
	if err != nil {
		t.Fatalf("ReadPrismAccount: %v", err)
	}
	if name != "Steve" {
		t.Errorf("name = %q, want Steve", name)
	}
}

func TestReadPrismAccountFallsBackToFirst(t *testing.T) {
	dir := t.TempDir()
	writePrismAccounts(t, dir, []map[string]any{
		{
			"localId": "local-1",
			"profile": map[string]any{"name": "Alex"},
		},
	}, "no-match")

	name, err := minecraft.ReadPrismAccount(dir)
	if err != nil {
		t.Fatalf("ReadPrismAccount: %v", err)
	}
	if name != "Alex" {
		t.Errorf("name = %q, want Alex", name)
	}
}

func TestReadPrismAccountMissingFile(t *testing.T) {
	_, err := minecraft.ReadPrismAccount(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing accounts.json")
	}
}

func TestReadVanillaAccountNewFormat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	mcDir := filepath.Join(dir, ".minecraft")
	os.MkdirAll(mcDir, 0755)

	data, _ := json.Marshal(map[string]any{
		"accounts": map[string]any{
			"acc-1": map[string]any{
				"minecraftProfile": map[string]any{"name": "Herobrine"},
			},
		},
		"activeAccountLocalId": "acc-1",
	})
	os.WriteFile(filepath.Join(mcDir, "launcher_accounts.json"), data, 0644)

	name, err := minecraft.ReadVanillaAccount()
	if err != nil {
		t.Fatalf("ReadVanillaAccount: %v", err)
	}
	if name != "Herobrine" {
		t.Errorf("name = %q, want Herobrine", name)
	}
}
