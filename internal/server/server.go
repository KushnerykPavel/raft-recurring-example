package server

import (
	"github.com/KushnerykPavel/raft-test-project/internal/server/raft_router"
	"github.com/KushnerykPavel/raft-test-project/internal/server/store_router"
	"github.com/dgraph-io/badger/v2"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/raft"
	"net/http"
	_ "net/http/pprof"
	"time"
)

type Srv struct {
	listenAddress string
	raft          *raft.Raft
	router        *chi.Mux
}

// Start start the server
func (s Srv) Start() error {
	server := &http.Server{
		Addr:         s.listenAddress,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		Handler:      s.router,
	}
	return server.ListenAndServe()
}

func New(listenAddr string, badgerDB *badger.DB, r *raft.Raft) *Srv {
	router := chi.NewRouter()
	router.Mount("/debug/pprof", http.DefaultServeMux)

	raftRouter := raft_router.New(r)
	router.Get("/raft/stats", raftRouter.StatsRaft)
	router.Post("/raft/join", raftRouter.JoinRaft)
	router.Post("/raft/remove", raftRouter.RemoveRaft)

	storeRouter := store_router.New(r, badgerDB, listenAddr)
	router.Post("/api/pay", storeRouter.Pay)
	router.Post("/api/recurring", storeRouter.Recurring)
	router.Get("/api/status/{order_id}", storeRouter.Status)

	return &Srv{
		listenAddress: listenAddr,
		raft:          r,
		router:        router,
	}
}
