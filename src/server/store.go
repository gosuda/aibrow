package server

import (
	"github.com/yjlee/aibrow/src/store"
)

// ImageRecord is an alias for store.ImageRecord so existing callers can keep
// using the server package type.
type ImageRecord = store.ImageRecord

// Store is an alias for store.Store so existing callers can keep using the
// server package type.
type Store = store.Store

// NewStore creates an empty Store.
func NewStore() *Store {
	return store.NewStore()
}
