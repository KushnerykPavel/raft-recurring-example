package store_router

import (
	"errors"
	"fmt"
	"github.com/KushnerykPavel/raft-test-project/internal/repo"
	"github.com/dgraph-io/badger/v2"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	"net/http"
)

func (h *Handler) Recurring(w http.ResponseWriter, r *http.Request) {
	data := &repo.RecurringRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if h.raft.State() != raft.Leader {
		render.Render(w, r, ErrInvalidRequest(errors.New("node is not leader")))
		return
	}

	var keyByte = []byte(data.Token)

	txn := h.db.NewTransaction(false)
	defer func() {
		_ = txn.Commit()
	}()

	_, err := txn.Get(keyByte)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			render.Render(w, r, ErrInvalidRequest(fmt.Errorf("key %s does not exists", data.Token)))
			return
		}
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("error getting key %s from storage: %s", data.Token, err.Error())))
		return
	}

	transaction := &repo.Transaction{
		ID:       uuid.New().String(),
		Type:     repo.RecurringTransactionType,
		Amount:   data.Amount,
		Currency: data.Currency,
	}

	if err := h.applyRaft("SET_TRANSACTIONS", data.OrderID, transaction); err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("applyRaft error: %s", err.Error())))
		return
	}

	response := &repo.RecurringResponse{Addr: h.addr}

	render.Status(r, http.StatusCreated)
	render.Render(w, r, response)
}
