package immich

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Client is an Immich REST API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	hostname   string

	mu         sync.Mutex
	tagCache   map[string]string // tag name -> tag ID
	albumCache map[string]string // album name -> album ID
}

// NewClient creates a new Immich client.
func NewClient(baseURL, apiKey string) *Client {
	hostname, _ := os.Hostname()
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		hostname:   hostname,
		tagCache:   make(map[string]string),
		albumCache: make(map[string]string),
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("x-api-key", c.apiKey)
	return c.httpClient.Do(req)
}

// UploadAsset uploads a file to Immich. Returns the asset ID and whether it was a duplicate.
func (c *Client) UploadAsset(ctx context.Context, deviceAssetID, filePath string) (string, bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", false, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", false, fmt.Errorf("stat file: %w", err)
	}
	modTime := info.ModTime().UTC().Format(time.RFC3339)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("deviceAssetId", deviceAssetID)
	mw.WriteField("deviceId", c.hostname)
	mw.WriteField("fileCreatedAt", modTime)
	mw.WriteField("fileModifiedAt", modTime)
	mw.WriteField("isFavorite", "false")

	part, err := mw.CreateFormFile("assetData", filepath.Base(filePath))
	if err != nil {
		return "", false, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", false, err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/assets", &body)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("upload: status %d: %s", resp.StatusCode, b)
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, fmt.Errorf("decode upload response: %w", err)
	}
	return result.ID, result.Status == "duplicate", nil
}

// SetDescription sets the description on an existing Immich asset.
func (c *Client) SetDescription(ctx context.Context, assetID, description string) error {
	body, _ := json.Marshal(map[string]string{"description": description})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/assets/"+assetID, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set description: status %d: %s", resp.StatusCode, b)
	}
	return nil
}

// ApplyTags looks up or creates each tag, then applies them to the asset.
func (c *Client) ApplyTags(ctx context.Context, assetID string, tags []string) error {
	if err := c.ensureTagsExist(ctx, tags); err != nil {
		return err
	}
	c.mu.Lock()
	tagIDs := make([]string, 0, len(tags))
	for _, t := range tags {
		if id, ok := c.tagCache[t]; ok {
			tagIDs = append(tagIDs, id)
		}
	}
	c.mu.Unlock()

	for _, tagID := range tagIDs {
		body, _ := json.Marshal(map[string][]string{"ids": {assetID}})
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/tags/"+tagID+"/assets", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("apply tag %s: status %d", tagID, resp.StatusCode)
		}
	}
	return nil
}

func (c *Client) ensureTagsExist(ctx context.Context, tags []string) error {
	c.mu.Lock()
	missing := []string{}
	for _, t := range tags {
		if _, ok := c.tagCache[t]; !ok {
			missing = append(missing, t)
		}
	}
	c.mu.Unlock()

	if len(missing) == 0 {
		return nil
	}

	// Fetch existing tags once
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var existing []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&existing); err != nil {
		return fmt.Errorf("decode tags response: %w", err)
	}

	c.mu.Lock()
	for _, t := range existing {
		c.tagCache[t.Name] = t.ID
	}
	// Recalculate missing after fetching
	stillMissing := []string{}
	for _, t := range missing {
		if _, ok := c.tagCache[t]; !ok {
			stillMissing = append(stillMissing, t)
		}
	}
	c.mu.Unlock()

	for _, name := range stillMissing {
		body, _ := json.Marshal(map[string]string{"name": name})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/tags", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.do(req)
		if err != nil {
			return err
		}
		var created struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&created); err == nil && created.ID != "" {
			c.mu.Lock()
			c.tagCache[created.Name] = created.ID
			c.mu.Unlock()
		}
		resp.Body.Close()
	}
	return nil
}

// AddToAlbum looks up or creates the named album, then adds the asset to it.
func (c *Client) AddToAlbum(ctx context.Context, assetID, albumName string) error {
	albumID, err := c.ensureAlbumExists(ctx, albumName)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string][]string{"ids": {assetID}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/albums/"+albumID+"/assets", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("add to album %s: status %d", albumID, resp.StatusCode)
	}
	return nil
}

func (c *Client) ensureAlbumExists(ctx context.Context, albumName string) (string, error) {
	c.mu.Lock()
	if id, ok := c.albumCache[albumName]; ok {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/albums", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var albums []struct {
		ID        string `json:"id"`
		AlbumName string `json:"albumName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&albums); err != nil {
		return "", fmt.Errorf("decode albums response: %w", err)
	}

	c.mu.Lock()
	for _, a := range albums {
		c.albumCache[a.AlbumName] = a.ID
	}
	if id, ok := c.albumCache[albumName]; ok {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	// Create the album
	body, _ := json.Marshal(map[string]any{"albumName": albumName, "description": "", "assetIds": []string{}})
	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/albums", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := c.do(createReq)
	if err != nil {
		return "", err
	}
	defer createResp.Body.Close()
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	if created.ID == "" {
		return "", fmt.Errorf("create album returned empty ID")
	}
	c.mu.Lock()
	c.albumCache[albumName] = created.ID
	c.mu.Unlock()
	return created.ID, nil
}
