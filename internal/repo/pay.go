package repo

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
)

type PayRequest struct {
	CardNumber string  `json:"card_number"`
	ExpiredAt  string  `json:"expired_at"`
	Cvv        string  `json:"cvv"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	OrderID    string  `json:"order_id"`
}

func (p *PayRequest) Bind(r *http.Request) error {
	return nil
}

func (p *PayRequest) GetRecurringToken() string {
	s := fmt.Sprintf("%s_%s_%s_%f_%s", p.CardNumber, p.ExpiredAt, p.Cvv, p.Amount, p.OrderID)
	h := sha1.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))
}

type PayResponse struct {
	Token string `json:"token"`
	Addr  string `json:"addr"`
}

func (rd *PayResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
