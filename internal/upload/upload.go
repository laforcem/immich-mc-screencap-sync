package upload

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/malc/screenshot-sync/internal/minecraft"
	"github.com/malc/screenshot-sync/internal/state"
)

// Uploader is the interface for uploading assets to Immich.
// *immich.Client satisfies this interface.
type Uploader interface {
	UploadAsset(ctx context.Context, deviceAssetID, filePath string) (assetID string, duplicate bool, err error)
	SetDescription(ctx context.Context, assetID, description string) error
	ApplyTags(ctx context.Context, assetID string, tags []string) error
	AddToAlbum(ctx context.Context, assetID, albumName string) error
}

// Screenshot uploads a single Minecraft screenshot to Immich.
// It is a no-op if the file has already been uploaded according to st.
// Each API step is retried once on failure before returning an error.
func Screenshot(ctx context.Context, u Uploader, st *state.State, albumName string, src minecraft.Source, filePath string) error {
	id := deviceAssetID(src, filePath)
	if st.IsUploaded(id) {
		return nil
	}

	var assetID string
	if err := retry(func() error {
		var err error
		assetID, _, err = u.UploadAsset(ctx, id, filePath)
		return err
	}); err != nil {
		return fmt.Errorf("upload asset: %w", err)
	}

	desc := fmt.Sprintf("Minecraft screenshot | Instance: %s | Account: %s", src.InstanceName, src.AccountName)
	if err := retry(func() error {
		return u.SetDescription(ctx, assetID, desc)
	}); err != nil {
		return fmt.Errorf("set description: %w", err)
	}

	tags := []string{
		"minecraft",
		fmt.Sprintf("mc-instance:%s", src.InstanceName),
		fmt.Sprintf("mc-account:%s", src.AccountName),
	}
	if err := retry(func() error {
		return u.ApplyTags(ctx, assetID, tags)
	}); err != nil {
		return fmt.Errorf("apply tags: %w", err)
	}

	if err := retry(func() error {
		return u.AddToAlbum(ctx, assetID, albumName)
	}); err != nil {
		return fmt.Errorf("add to album: %w", err)
	}

	return st.MarkUploaded(id, assetID)
}

// retry calls fn, retrying once on failure.
func retry(fn func() error) error {
	if err := fn(); err != nil {
		return fn()
	}
	return nil
}

func deviceAssetID(src minecraft.Source, filePath string) string {
	filename := filepath.Base(filePath)
	if src.Type == minecraft.SourcePrism {
		return fmt.Sprintf("prism/%s/%s", src.InstanceName, filename)
	}
	return fmt.Sprintf("vanilla/%s", filename)
}
