package store_router

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/KushnerykPavel/raft-test-project/internal/repo"
	"github.com/dgraph-io/badger/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"net/http"
)

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "order_id")
	if orderID == "" {
		render.Render(w, r, ErrInvalidRequest(errors.New("order_id must not be empty")))
		return
	}

	var keyByte = []byte(orderID)

	txn := h.db.NewTransaction(false)
	defer func() {
		_ = txn.Commit()
	}()

	item, err := txn.Get(keyByte)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			render.Render(w, r, ErrInvalidRequest(fmt.Errorf("key %s does not exists", orderID)))
			return
		}
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("error getting key %s from storage: %s", orderID, err.Error())))
		return
	}

	var value = make([]byte, 0)
	err = item.Value(func(val []byte) error {
		value = append(value, val...)
		return nil
	})

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("error appending byte value of key %s from storage: %s", orderID, err.Error())))
		return
	}

	var data []repo.Transaction
	if value != nil && len(value) > 0 {
		err = json.Unmarshal(value, &data)
	}

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("error unmarshal data to interface: %s", err.Error())))
		return
	}

	response, _ := json.Marshal(map[string]interface{}{
		"order_id":     orderID,
		"transactions": data,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(response)
}
