package store_router

import (
	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/raft"
)

type Handler struct {
	raft *raft.Raft
	db   *badger.DB
}

func New(raft *raft.Raft, db *badger.DB) *Handler {
	return &Handler{
		raft: raft,
		db:   db,
	}
}
