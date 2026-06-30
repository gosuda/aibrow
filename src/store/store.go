// Package store holds the in-memory repository for uploaded images and their
// processed derivatives. It lives in its own package so that tool implementations
// can use it without creating an import cycle with the server package.
package store

import (
	"fmt"
	"sync"

	"github.com/yjlee/aibrow/src/ffmpeg"
)

// ImageRecord tracks an uploaded image and its processed derivatives.
type ImageRecord struct {
	ID            string              `json:"id"`
	OriginalPath  string              `json:"original_path"`
	ProcessedPath string              `json:"processed_path"`
	Analysis      ffmpeg.Analysis     `json:"analysis"`
	Params        ffmpeg.AdjustParams `json:"params"`
}

// Store is an in-memory repository of uploaded images.
type Store struct {
	mu      sync.RWMutex
	records map[string]ImageRecord
	seq     int
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{records: make(map[string]ImageRecord)}
}

// Save adds a record to the store and returns it with a generated ID.
func (s *Store) Save(record ImageRecord) ImageRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	record.ID = fmt.Sprintf("img-%d", s.seq)
	s.records[record.ID] = record
	return record
}

// Get retrieves a record by ID.
func (s *Store) Get(id string) (ImageRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	return r, ok
}

// Update replaces an existing record. The ID must already exist.
func (s *Store) Update(record ImageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[record.ID]; !ok {
		return fmt.Errorf("record %s not found", record.ID)
	}
	s.records[record.ID] = record
	return nil
}

// Len returns the number of stored records.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}
