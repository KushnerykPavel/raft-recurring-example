package raft_router

import (
	"errors"
	"fmt"
	"github.com/go-chi/render"
	"github.com/hashicorp/raft"
	"net/http"
)

type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

type requestJoin struct {
	NodeID      string `json:"node_id"`
	RaftAddress string `json:"raft_address"`
}

func (j *requestJoin) Bind(r *http.Request) error {
	return nil
}

type responseJoin struct {
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

func (j *responseJoin) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (h *Handler) JoinRaft(w http.ResponseWriter, r *http.Request) {
	var data = &requestJoin{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var (
		nodeID   = data.NodeID
		raftAddr = data.RaftAddress
	)

	if h.raft.State() != raft.Leader {
		render.Render(w, r, ErrInvalidRequest(errors.New("not the leader")))
		return
	}

	configFuture := h.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("failed to get raft configuration: %s", err.Error())))
		return
	}

	// This must be run on the leader or it will fail.
	f := h.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(raftAddr), 0, 0)
	if f.Error() != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("error add voter: %s", f.Error().Error())))
		return
	}

	response := &responseJoin{
		Message: fmt.Sprintf("node %s at %s joined successfully", nodeID, raftAddr),
		Data:    h.raft.Stats(),
	}
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}
