package repo

import "net/http"

type RecurringRequest struct {
	Token    string  `json:"token"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	OrderID  string  `json:"order_id"`
}

func (p *RecurringRequest) Bind(r *http.Request) error {
	return nil
}

type RecurringResponse struct {
	Addr string `json:"addr"`
}

func (rd *RecurringResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
