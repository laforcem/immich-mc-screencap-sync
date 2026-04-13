package upload_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/state"
	"github.com/malc/screenshot-sync/internal/upload"
)

type mockUploader struct {
	uploadedID  string
	description string
	tags        []string
	album       string
}

func (m *mockUploader) UploadAsset(_ context.Context, deviceAssetID, _ string) (string, bool, error) {
	m.uploadedID = deviceAssetID
	return "asset-123", false, nil
}
func (m *mockUploader) SetDescription(_ context.Context, _, desc string) error {
	m.description = desc
	return nil
}
func (m *mockUploader) ApplyTags(_ context.Context, _ string, tags []string) error {
	m.tags = tags
	return nil
}
func (m *mockUploader) AddToAlbum(_ context.Context, _, album string) error {
	m.album = album
	return nil
}

func TestScreenshotUploadsPrism(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "2024-01-15_12.30.00.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	st, _ := state.Load(filepath.Join(dir, "state.json"))
	u := &mockUploader{}
	src := minecraft.Source{
		Type:         minecraft.SourcePrism,
		InstanceName: "Survival",
		AccountName:  "Steve",
	}

	if err := upload.Screenshot(context.Background(), u, st, "Minecraft", src, f); err != nil {
		t.Fatalf("Screenshot: %v", err)
	}

	if u.uploadedID != "prism/Survival/2024-01-15_12.30.00.png" {
		t.Errorf("deviceAssetID = %q", u.uploadedID)
	}
	if u.description != "Minecraft screenshot | Instance: Survival | Account: Steve" {
		t.Errorf("description = %q", u.description)
	}
	expectedTags := []string{"minecraft", "mc-instance:Survival", "mc-account:Steve"}
	if len(u.tags) != len(expectedTags) {
		t.Errorf("tags = %v, want %v", u.tags, expectedTags)
	}
	if u.album != "Minecraft" {
		t.Errorf("album = %q", u.album)
	}
	if !st.IsUploaded("prism/Survival/2024-01-15_12.30.00.png") {
		t.Error("state not marked as uploaded")
	}
}

func TestScreenshotSkipsAlreadyUploaded(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	st, _ := state.Load(filepath.Join(dir, "state.json"))
	st.MarkUploaded("prism/Survival/a.png", "existing-id")

	u := &mockUploader{}
	src := minecraft.Source{Type: minecraft.SourcePrism, InstanceName: "Survival", AccountName: "Steve"}

	upload.Screenshot(context.Background(), u, st, "Minecraft", src, f)

	if u.uploadedID != "" {
		t.Error("should have skipped upload for already-uploaded file")
	}
}

func TestScreenshotVanillaDeviceAssetID(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "2024-02-01_10.00.00.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	st, _ := state.Load(filepath.Join(dir, "state.json"))
	u := &mockUploader{}
	src := minecraft.Source{Type: minecraft.SourceVanilla, InstanceName: "vanilla", AccountName: "Alex"}

	upload.Screenshot(context.Background(), u, st, "Minecraft", src, f)

	if u.uploadedID != "vanilla/2024-02-01_10.00.00.png" {
		t.Errorf("deviceAssetID = %q", u.uploadedID)
	}
}
