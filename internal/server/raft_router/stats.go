package raft_router

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) StatsRaft(w http.ResponseWriter, r *http.Request) {
	response, _ := json.Marshal(h.raft.Stats())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}
