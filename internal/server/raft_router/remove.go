package raft_router

import (
	"errors"
	"fmt"
	"github.com/go-chi/render"
	"github.com/hashicorp/raft"
	"net/http"
)

type requestRemove struct {
	NodeID string `json:"node_id"`
}

func (j *requestRemove) Bind(r *http.Request) error {
	return nil
}

type responseRemove struct {
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

func (j *responseRemove) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// RemoveRaftHandler handling removing raft
func (h *Handler) RemoveRaft(w http.ResponseWriter, r *http.Request) {
	var data = &requestRemove{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var nodeID = data.NodeID

	if h.raft.State() != raft.Leader {
		render.Render(w, r, ErrInvalidRequest(errors.New("not the leader")))
		return
	}

	configFuture := h.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("failed to get raft configuration: %s", err.Error())))
		return
	}

	future := h.raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
	if err := future.Error(); err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("error removing existing node %s: %s", nodeID, err.Error())))
		return
	}

	response := &responseRemove{
		Message: fmt.Sprintf("node %s removed successfully", nodeID),
		Data:    h.raft.Stats(),
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}
