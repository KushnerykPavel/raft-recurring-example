package store_router

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/KushnerykPavel/raft-test-project/internal/repo"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	"net/http"
	"time"
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

func (h *Handler) applyRaft(operation, key string, value any) error {
	payload := repo.CommandPayload{
		Operation: operation,
		Key:       key,
		Value:     value,
	}

	raftPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error preparing saving data payload: %s", err.Error())
	}

	applyFuture := h.raft.Apply(raftPayload, 500*time.Microsecond)
	if err := applyFuture.Error(); err != nil {
		return fmt.Errorf("error persisting data in raft cluster: %s", err.Error())
	}

	_, ok := applyFuture.Response().(*repo.ApplyResponse)
	if !ok {
		return errors.New("error response is not match apply response")
	}

	return nil
}

func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	data := &repo.PayRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if h.raft.State() != raft.Leader {
		render.Render(w, r, ErrInvalidRequest(errors.New("node is not leader")))
		return
	}

	transaction := &repo.Transaction{
		ID:       uuid.New().String(),
		Type:     repo.FirstTransactionType,
		Amount:   data.Amount,
		Currency: data.Currency,
	}

	token := data.GetRecurringToken()

	if err := h.applyRaft("SET", token, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("applyRaft error: %s", err.Error())))
		return
	}

	if err := h.applyRaft("SET_TRANSACTIONS", data.OrderID, transaction); err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("applyRaft error: %s", err.Error())))
		return
	}

	response := &repo.PayResponse{Token: token, Addr: h.addr}

	render.Status(r, http.StatusCreated)
	render.Render(w, r, response)
}
