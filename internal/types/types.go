package types

import (
	"fmt"
	"time"
)

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
	TimeAgo     string  `json:"time_ago"`               // New field for human-readable time
	RefreshTier string  `json:"refresh_tier,omitempty"` // Optional field to show the refresh tier
}

// FormatTimestamp converts a Unix timestamp to "YYYY-MM-DD HH:MM:SS" format in local time
func FormatTimestamp(timestamp int64) string {
	loc, err := time.LoadLocation("America/Vancouver")
	if err != nil {
		// if location loading fails, fallback to UTC
		return time.Unix(timestamp, 0).Local().Format("2006-01-02 15:04:05")
	}
	return time.Unix(timestamp, 0).In(loc).Format("2006-01-02 15:04:05")
}

// FormatTimeAgo returns a human-readable string representing the time since the timestamp
func FormatTimeAgo(timestamp int64) string {
	now := time.Now().Unix()
	diff := now - timestamp

	switch {
	case diff < 0:
		return "in the future" // For handling clock skew
	case diff < 5:
		return "just now"
	case diff < 60:
		return fmt.Sprintf("%ds ago", diff)
	case diff < 3600:
		return fmt.Sprintf("%dm ago", diff/60)
	case diff < 86400:
		return fmt.Sprintf("%dh ago", diff/3600)
	default:
		return fmt.Sprintf("%dd ago", diff/86400)
	}
}

// ToResponse converts PriceData to PriceDataResponse with formatted timestamp
func (p *PriceData) ToResponse() PriceDataResponse {
	return PriceDataResponse{
		Asset:       p.Asset,
		Price:       p.Price,
		LastUpdated: FormatTimestamp(p.Timestamp),
		TimeAgo:     FormatTimeAgo(p.Timestamp),
	}
}

// ToResponseWithTier converts PriceData to PriceDataResponse and includes the refresh tier
func (p *PriceData) ToResponseWithTier(tier string) PriceDataResponse {
	resp := p.ToResponse()
	resp.RefreshTier = tier
	return resp
}
