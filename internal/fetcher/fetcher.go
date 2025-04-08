package fetcher

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Fetcher interface defines price fetching operations
type Fetcher interface {
	FetchPrice(symbol string) (AggregatedPrice, error)
}

// HttpFetcher implements the Fetcher interface
type HttpFetcher struct {
	endpoints []string
	client    *http.Client
}

// ExchangeResponse is the format of the exchange API response
type ExchangeResponse struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
}

// PriceData represents price data from an exchange
type PriceData struct {
	Price     float64
	Volume    float64
	Timestamp int64
}

// AggregatedPrice represents aggregated price data
type AggregatedPrice struct {
	Price     float64
	Timestamp int64
}

// NewFetcher creates a new price fetcher
func NewFetcher(endpoints []string) Fetcher {
	return &HttpFetcher{
		endpoints: endpoints,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// FetchPrice fetches prices from multiple exchanges and calculates weighted average
func (f *HttpFetcher) FetchPrice(symbol string) (AggregatedPrice, error) {
	var wg sync.WaitGroup
	results := make([]PriceData, 0, len(f.endpoints))
	var mu sync.Mutex
	errCount := 0
	var errMu sync.Mutex

	// Fetch data in parallel
	for _, endpoint := range f.endpoints {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			fullURL := fmt.Sprintf("%s/%s", url, symbol)
			log.Printf("Fetching from: %s", fullURL)

			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				log.Printf("Error creating request for %s: %v", url, err)
				errMu.Lock()
				errCount++
				errMu.Unlock()
				return
			}

			resp, err := f.client.Do(req)
			if err != nil {
				log.Printf("Error fetching from %s: %v", url, err)
				errMu.Lock()
				errCount++
				errMu.Unlock()
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("Error: Non-200 status code from %s: %d", url, resp.StatusCode)
				errMu.Lock()
				errCount++
				errMu.Unlock()
				return
			}

			// Decode response
			var exchange ExchangeResponse
			err = json.NewDecoder(resp.Body).Decode(&exchange)
			if err != nil {
				log.Printf("Error decoding response from %s: %v", url, err)
				errMu.Lock()
				errCount++
				errMu.Unlock()
				return
			}

			// Validate data
			if exchange.Price <= 0 || exchange.Volume <= 0 {
				log.Printf("Warning: Invalid price or volume from %s: %v", url, exchange)
				errMu.Lock()
				errCount++
				errMu.Unlock()
				return
			}

			log.Printf("Received from %s: symbol=%s, price=%f, volume=%f",
				url, exchange.Symbol, exchange.Price, exchange.Volume)

			// Add to results
			mu.Lock()
			results = append(results, PriceData{
				Price:     exchange.Price,
				Volume:    exchange.Volume,
				Timestamp: exchange.Timestamp,
			})
			mu.Unlock()
		}(endpoint)
	}

	// Wait for all requests to complete
	wg.Wait()

	// Check if we have enough data
	if len(results) == 0 {
		return AggregatedPrice{}, fmt.Errorf("all fetches failed, received 0 valid responses")
	}

	if errCount > 0 {
		log.Printf("Warning: %d out of %d exchanges failed to provide data", errCount, len(f.endpoints))
	}

	// Calculate weighted average
	var totalPrice, totalVolume float64
	var latestTimestamp int64
	for _, data := range results {
		totalPrice += data.Price * data.Volume
		totalVolume += data.Volume
		if data.Timestamp > latestTimestamp {
			latestTimestamp = data.Timestamp
		}
	}

	if totalVolume == 0 {
		return AggregatedPrice{}, fmt.Errorf("total volume is zero, cannot calculate weighted average")
	}

	avgPrice := totalPrice / totalVolume
	log.Printf("Calculated weighted average price for %s: %f based on %d exchanges",
		symbol, avgPrice, len(results))

	return AggregatedPrice{
		Price:     avgPrice,
		Timestamp: latestTimestamp,
	}, nil
}
