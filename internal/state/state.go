package state

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// State tracks which screenshots have been uploaded to Immich.
type State struct {
	mu       sync.Mutex
	uploaded map[string]string // deviceAssetID -> immich asset ID
	path     string
}

// Load reads state from path. Returns an empty state if the file is missing or corrupt.
func Load(path string) (*State, error) {
	s := &State{
		uploaded: make(map[string]string),
		path:     path,
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, nil
	}
	_ = json.Unmarshal(data, &s.uploaded) // silently ignore corrupt JSON
	return s, nil
}

// IsUploaded reports whether deviceAssetID has already been uploaded.
func (s *State) IsUploaded(deviceAssetID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.uploaded[deviceAssetID]
	return ok
}

// MarkUploaded records a successful upload and saves state atomically to disk.
func (s *State) MarkUploaded(deviceAssetID, assetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uploaded[deviceAssetID] = assetID
	return s.save()
}

func (s *State) save() error {
	data, err := json.MarshalIndent(s.uploaded, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}
