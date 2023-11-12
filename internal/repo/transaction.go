package repo

type TransactionType string

var (
	FirstTransactionType     TransactionType = "first"
	RecurringTransactionType TransactionType = "recurring"
)

type Transaction struct {
	ID       string          `json:"id"`
	Type     TransactionType `json:"type"`
	Amount   float64         `json:"amount"`
	Currency string          `json:"currency"`
}
