package server

import (
	"testing"
)

func TestStoreSaveAndGet(t *testing.T) {
	s := NewStore()
	r := s.Save(ImageRecord{OriginalPath: "/tmp/a.png"})
	if r.ID == "" {
		t.Fatal("expected generated id")
	}
	got, ok := s.Get(r.ID)
	if !ok {
		t.Fatal("expected record to exist")
	}
	if got.OriginalPath != "/tmp/a.png" {
		t.Errorf("path = %q, want /tmp/a.png", got.OriginalPath)
	}
}

func TestStoreUpdate(t *testing.T) {
	s := NewStore()
	r := s.Save(ImageRecord{OriginalPath: "/tmp/a.png"})
	r.ProcessedPath = "/tmp/b.png"
	if err := s.Update(r); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get(r.ID)
	if got.ProcessedPath != "/tmp/b.png" {
		t.Errorf("processed path = %q", got.ProcessedPath)
	}
}

func TestStoreUpdateMissing(t *testing.T) {
	s := NewStore()
	err := s.Update(ImageRecord{ID: "missing"})
	if err == nil {
		t.Error("expected error updating missing record")
	}
}

func TestStoreLen(t *testing.T) {
	s := NewStore()
	if s.Len() != 0 {
		t.Errorf("len = %d, want 0", s.Len())
	}
	s.Save(ImageRecord{})
	if s.Len() != 1 {
		t.Errorf("len = %d, want 1", s.Len())
	}
}
