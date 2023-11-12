package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/raft"
	"io"
	"log"
	"os"
	"strings"
)

// CommandPayload is payload sent by system when calling raft.Apply(cmd []byte, timeout time.Duration)
type CommandPayload struct {
	Operation string
	Key       string
	Value     interface{}
}

// ApplyResponse response from Apply raft
type ApplyResponse struct {
	Error error
	Data  interface{}
}

type BadgerFSM struct {
	db *badger.DB
}

func (b *BadgerFSM) get(key string) (interface{}, error) {
	var keyByte = []byte(key)
	var data interface{}

	txn := b.db.NewTransaction(false)
	defer func() {
		_ = txn.Commit()
	}()

	item, err := txn.Get(keyByte)
	if err != nil {
		data = map[string]interface{}{}
		return data, err
	}

	var value = make([]byte, 0)
	err = item.Value(func(val []byte) error {
		value = append(value, val...)
		return nil
	})

	if err != nil {
		data = map[string]interface{}{}
		return data, err
	}

	if value != nil && len(value) > 0 {
		err = json.Unmarshal(value, &data)
	}

	if err != nil {
		data = map[string]interface{}{}
	}

	return data, err
}

func (b *BadgerFSM) set(key string, value interface{}) error {
	log.Print("set: key:  ", key, " value: ", value)

	var data = make([]byte, 0)
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if data == nil || len(data) <= 0 {
		return nil
	}

	txn := b.db.NewTransaction(true)
	err = txn.Set([]byte(key), data)
	if err != nil {
		txn.Discard()
		return err
	}

	return txn.Commit()
}

func (b *BadgerFSM) toTransaction(value any) (*Transaction, error) {
	trx, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var transaction Transaction
	err = json.Unmarshal(trx, &transaction)
	return &transaction, err
}

func (b *BadgerFSM) setTransactions(key string, value interface{}) error {
	log.Print("set_transactions: key:  ", key, " value: ", value)
	trx, err := b.toTransaction(value)
	if err != nil {
		return err
	}

	_, err = b.get(trx.ID)
	if err == nil {
		return nil
	}
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return err
	}
	_ = b.set(trx.ID, trx)

	trxs, err := b.get(key)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return err
	}

	if err != nil && errors.Is(err, badger.ErrKeyNotFound) {
		trxList := []interface{}{value}
		return b.set(key, trxList)

	}
	transactions, ok := trxs.([]interface{})
	if !ok {
		return errors.New("unprocessed entity")
	}
	transactions = append(transactions, value)

	return b.set(key, transactions)
}

func (b *BadgerFSM) delete(key string) error {
	var keyByte = []byte(key)

	txn := b.db.NewTransaction(true)
	defer func() {
		_ = txn.Commit()
	}()

	err := txn.Delete(keyByte)
	if err != nil {
		return err
	}

	return txn.Commit()
}

// Apply log is invoked once a log entry is committed.
// It returns a value which will be made available in the
// ApplyFuture returned by Raft.Apply method if that
// method was called on the same Raft node as the FSM.
func (b *BadgerFSM) Apply(log *raft.Log) interface{} {
	switch log.Type {
	case raft.LogCommand:
		var payload = CommandPayload{}
		if err := json.Unmarshal(log.Data, &payload); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error marshalling store payload %s\n", err.Error())
			return nil
		}

		op := strings.ToUpper(strings.TrimSpace(payload.Operation))
		switch op {
		case "SET_TRANSACTIONS":
			return &ApplyResponse{
				Error: b.setTransactions(payload.Key, payload.Value),
				Data:  payload.Value,
			}
		case "SET":
			return &ApplyResponse{
				Error: b.set(payload.Key, payload.Value),
				Data:  payload.Value,
			}
		case "GET":
			data, err := b.get(payload.Key)
			return &ApplyResponse{
				Error: err,
				Data:  data,
			}

		case "DELETE":
			return &ApplyResponse{
				Error: b.delete(payload.Key),
				Data:  nil,
			}
		}
	}

	_, _ = fmt.Fprintf(os.Stderr, "not raft log command type\n")
	return nil
}

// Snapshot will be called during make snapshot.
// Snapshot is used to support log compaction.
// No need to call snapshot since it already persisted in disk (using BadgerDB) when raft calling Apply function.
func (b *BadgerFSM) Snapshot() (raft.FSMSnapshot, error) {
	return newSnapshotNoop()
}

// Restore is used to restore an FSM from a Snapshot. It is not called
// concurrently with any other command. The FSM must discard all previous
// state.
// Restore will update all data in BadgerDB
func (b *BadgerFSM) Restore(rClose io.ReadCloser) error {
	defer func() {
		if err := rClose.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "[FINALLY RESTORE] close error %s\n", err.Error())
		}
	}()

	_, _ = fmt.Fprintf(os.Stdout, "[START RESTORE] read all message from snapshot\n")
	var totalRestored int

	decoder := json.NewDecoder(rClose)
	for decoder.More() {
		var data = &CommandPayload{}
		err := decoder.Decode(data)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "[END RESTORE] error decode data %s\n", err.Error())
			return err
		}

		if err := b.set(data.Key, data.Value); err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "[END RESTORE] error persist data %s\n", err.Error())
			return err
		}

		totalRestored++
	}

	// read closing bracket
	_, err := decoder.Token()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "[END RESTORE] error %s\n", err.Error())
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "[END RESTORE] success restore %d messages in snapshot\n", totalRestored)
	return nil
}

func NewBadger(badgerDB *badger.DB) *BadgerFSM {
	return &BadgerFSM{
		db: badgerDB,
	}
}
