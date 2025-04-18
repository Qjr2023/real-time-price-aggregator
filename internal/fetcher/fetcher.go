package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"real-time-price-aggregator/internal/types"
	"strings"
	"sync"
	"time"
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
	var mu sync.Mutex // mutex for safe concurrent access to responses
	var wg sync.WaitGroup

	// create a buffered channel for error handling
	errChan := make(chan error, len(f.endpoints))

	// concurrent fetch from all endpoints
	for _, endpoint := range f.endpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()

			url := fmt.Sprintf("%s/%s", endpoint, symbol)
			// add a timeout to the HTTP client
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				errChan <- fmt.Errorf("failed to fetch from %s: %v", endpoint, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errChan <- fmt.Errorf("unexpected status code from %s: %d", endpoint, resp.StatusCode)
				return
			}

			var mockResp mockResponse
			if err := json.NewDecoder(resp.Body).Decode(&mockResp); err != nil {
				errChan <- fmt.Errorf("failed to decode response from %s: %v", endpoint, err)
				return
			}

			// safe concurrent access to responses
			mu.Lock()
			responses = append(responses, mockResp)
			mu.Unlock()
		}(endpoint)
	}

	// wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// check for errors
	var lastErr error
	for err := range errChan {
		lastErr = err
	}

	// check if we have valid responses
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
