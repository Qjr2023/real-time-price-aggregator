package types

// PriceData represents the price data structure
type PriceData struct {
	Asset     string  `json:"asset"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"last_updated"`
}
