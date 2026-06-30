package store

import (
	"testing"

	"github.com/yjlee/aibrow/src/ffmpeg"
)

func TestStore(t *testing.T) {
	s := NewStore()
	if s.Len() != 0 {
		t.Errorf("len = %d, want 0", s.Len())
	}

	record := s.Save(ImageRecord{OriginalPath: "/tmp/in.png"})
	if record.ID == "" {
		t.Error("expected generated id")
	}
	if s.Len() != 1 {
		t.Errorf("len = %d, want 1", s.Len())
	}

	got, ok := s.Get(record.ID)
	if !ok {
		t.Fatal("expected record")
	}
	if got.OriginalPath != "/tmp/in.png" {
		t.Errorf("path = %q", got.OriginalPath)
	}

	got.Analysis = ffmpeg.Analysis{Width: 100}
	if err := s.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Get(record.ID)
	if got.Analysis.Width != 100 {
		t.Errorf("width = %d", got.Analysis.Width)
	}
}

func TestStoreGetMissing(t *testing.T) {
	s := NewStore()
	_, ok := s.Get("missing")
	if ok {
		t.Error("expected not found")
	}
}

func TestStoreUpdateMissing(t *testing.T) {
	s := NewStore()
	err := s.Update(ImageRecord{ID: "missing"})
	if err == nil {
		t.Error("expected error")
	}
}
