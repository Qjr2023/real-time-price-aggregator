package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"real-time-price-aggregator/internal/types"
	"strings"
)

// Fetcher interface defines price fetching operations
type Fetcher interface {
	FetchPrice(symbol string) (*types.PriceData, error)
}

// fetcher struct implements the Fetcher interface
type fetcher struct {
	endpoints []string
}

// NewFetcher creates a new Fetcher instance
func NewFetcher(endpoints []string) Fetcher {
	return &fetcher{endpoints: endpoints}
}

// FetchPrice fetches the price for a symbol from mock exchanges and calculates a weighted average
func (f *fetcher) FetchPrice(symbol string) (*types.PriceData, error) {
	type mockResponse struct {
		Symbol    string  `json:"symbol"`
		Price     float64 `json:"price"`
		Volume    float64 `json:"volume"`
		Timestamp int64   `json:"timestamp"`
	}

	var responses []mockResponse
	var lastErr error

	// Fetch data from all endpoints
	for _, endpoint := range f.endpoints {
		url := fmt.Sprintf("%s/%s", endpoint, symbol)
		resp, err := http.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch from %s: %v", endpoint, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code from %s: %d", endpoint, resp.StatusCode)
			continue
		}

		var mockResp mockResponse
		if err := json.NewDecoder(resp.Body).Decode(&mockResp); err != nil {
			lastErr = fmt.Errorf("failed to decode response from %s: %v", endpoint, err)
			continue
		}

		responses = append(responses, mockResp)
	}

	// Check if we have any valid responses
	if len(responses) == 0 {
		if lastErr == nil {
			lastErr = fmt.Errorf("no valid data received from any endpoint")
		}
		return nil, lastErr
	}

	// Calculate weighted average
	var totalPrice, totalVolume float64
	var latestTimestamp int64
	for _, resp := range responses {
		totalPrice += resp.Price * resp.Volume
		totalVolume += resp.Volume
		if resp.Timestamp > latestTimestamp {
			latestTimestamp = resp.Timestamp
		}
	}

	if totalVolume == 0 {
		return nil, fmt.Errorf("total volume is zero, cannot calculate weighted average")
	}

	priceData := &types.PriceData{
		Asset:     strings.ToLower(symbol),
		Price:     totalPrice / totalVolume,
		Timestamp: latestTimestamp,
	}

	return priceData, nil
}
