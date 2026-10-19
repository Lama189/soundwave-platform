package fx

import "github.com/shopspring/decimal"

type erAPIResponse struct {
	Result   string                     `json:"result"`
	BaseCode string                     `json:"base_code"`
	Rates    map[string]decimal.Decimal `json:"rates"`
}
