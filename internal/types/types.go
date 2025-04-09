package types

import "time"

// PriceData represents the price data structure
type PriceData struct {
	Asset     string  `json:"asset"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"last_updated"`
}

// PriceDataResponse represents the price data structure for API responses
type PriceDataResponse struct {
	Asset       string  `json:"asset"`
	Price       float64 `json:"price"`
	LastUpdated string  `json:"last_updated"`
}

// FormatTimestamp converts a Unix timestamp to "YYYY-MM-DD HH:MM:SS" format in local time
func FormatTimestamp(timestamp int64) string {
	return time.Unix(timestamp, 0).Local().Format("2006-01-02 15:04:05")
}

// ToResponse converts PriceData to PriceDataResponse with formatted timestamp
func (p *PriceData) ToResponse() PriceDataResponse {
	return PriceDataResponse{
		Asset:       p.Asset,
		Price:       p.Price,
		LastUpdated: FormatTimestamp(p.Timestamp),
	}
}
