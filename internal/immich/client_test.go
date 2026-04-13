package immich_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malc/screenshot-sync/internal/immich"
)

func fakeServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/assets":
			json.NewEncoder(w).Encode(map[string]any{"id": "asset-1", "status": "created"})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/assets/"):
			json.NewEncoder(w).Encode(map[string]any{"id": "asset-1"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			json.NewEncoder(w).Encode([]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/tags":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			json.NewEncoder(w).Encode(map[string]any{"id": "tag-" + body["name"], "name": body["name"]})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/tags/"):
			json.NewEncoder(w).Encode(map[string]any{"successIds": []string{"asset-1"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums":
			json.NewEncoder(w).Encode([]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/albums":
			json.NewEncoder(w).Encode(map[string]any{"id": "album-1", "albumName": "Minecraft"})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/albums/"):
			json.NewEncoder(w).Encode(map[string]any{"successIds": []string{"asset-1"}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, &calls
}

func TestUploadAsset(t *testing.T) {
	srv, _ := fakeServer(t)
	defer srv.Close()

	f := filepath.Join(t.TempDir(), "screenshot.png")
	os.WriteFile(f, []byte("PNG"), 0644)

	c := immich.NewClient(srv.URL, "testkey")
	assetID, dup, err := c.UploadAsset(context.Background(), "prism/World/screenshot.png", f)
	if err != nil {
		t.Fatalf("UploadAsset: %v", err)
	}
	if assetID != "asset-1" {
		t.Errorf("assetID = %q", assetID)
	}
	if dup {
		t.Error("should not be duplicate")
	}
}

func TestSetDescription(t *testing.T) {
	srv, _ := fakeServer(t)
	defer srv.Close()

	c := immich.NewClient(srv.URL, "testkey")
	if err := c.SetDescription(context.Background(), "asset-1", "test description"); err != nil {
		t.Fatalf("SetDescription: %v", err)
	}
}

func TestApplyTagsCreatesAndCaches(t *testing.T) {
	srv, calls := fakeServer(t)
	defer srv.Close()

	c := immich.NewClient(srv.URL, "testkey")
	tags := []string{"minecraft", "mc-instance:Survival"}

	if err := c.ApplyTags(context.Background(), "asset-1", tags); err != nil {
		t.Fatalf("ApplyTags: %v", err)
	}
	// Call again — tag cache should prevent duplicate GET /api/tags
	if err := c.ApplyTags(context.Background(), "asset-1", tags); err != nil {
		t.Fatalf("second ApplyTags: %v", err)
	}
	getTagsCalls := 0
	for _, c := range *calls {
		if c == "GET /api/tags" {
			getTagsCalls++
		}
	}
	if getTagsCalls > 1 {
		t.Errorf("GET /api/tags called %d times, want 1 (should be cached)", getTagsCalls)
	}
}

func TestAddToAlbumCreatesAndCaches(t *testing.T) {
	srv, calls := fakeServer(t)
	defer srv.Close()

	c := immich.NewClient(srv.URL, "testkey")
	if err := c.AddToAlbum(context.Background(), "asset-1", "Minecraft"); err != nil {
		t.Fatalf("AddToAlbum: %v", err)
	}
	if err := c.AddToAlbum(context.Background(), "asset-1", "Minecraft"); err != nil {
		t.Fatalf("second AddToAlbum: %v", err)
	}
	getAlbumCalls := 0
	for _, c := range *calls {
		if c == "GET /api/albums" {
			getAlbumCalls++
		}
	}
	if getAlbumCalls > 1 {
		t.Errorf("GET /api/albums called %d times, want 1 (should be cached)", getAlbumCalls)
	}
}
