package fetcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"real-time-price-aggregator/internal/circuitbreaker"
	"real-time-price-aggregator/internal/metrics"
	"real-time-price-aggregator/internal/types"
	"strings"
	"time"
)

// Error definitions
var (
	ErrAssetNotSupported = errors.New("asset not supported")
	ErrNoValidData       = errors.New("no valid data received from any endpoint")
	ErrZeroVolume        = errors.New("total volume is zero, cannot calculate weighted average")
)

// Fetcher interface defines price fetching operations
type Fetcher interface {
	FetchPrice(symbol string) (*types.PriceData, error)
}

// fetcher struct implements the Fetcher interface
type fetcher struct {
	endpoints       []string
	client          *http.Client
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker
	metrics         *metrics.MetricsService
}

// mockResponse represents the response from a mock exchange
type mockResponse struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
}

// NewFetcher creates a new Fetcher instance
func NewFetcher(endpoints []string, m *metrics.MetricsService) Fetcher {
	// Initialize HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Initialize circuit breakers for each endpoint
	circuitBreakers := make(map[string]*circuitbreaker.CircuitBreaker)
	for _, endpoint := range endpoints {
		name := strings.TrimPrefix(endpoint, "http://")
		name = strings.TrimPrefix(name, "https://")

		// Circuit opens after 5 failures, resets after 30 seconds, allows 2 retries in half-open state
		circuitBreakers[endpoint] = circuitbreaker.New(
			name,
			5,              // Failure threshold
			30*time.Second, // Reset timeout
			2,              // Half-open max retries
		)
	}

	return &fetcher{
		endpoints:       endpoints,
		client:          client,
		circuitBreakers: circuitBreakers,
		metrics:         m,
	}
}

// fetchFromEndpoint fetches price data from a single endpoint
func (f *fetcher) fetchFromEndpoint(endpoint, symbol string) (*mockResponse, error) {
	url := fmt.Sprintf("%s/%s", endpoint, symbol)

	// Record the request
	f.metrics.RecordExchangeRequest(endpoint)
	startTime := time.Now()

	// Check if the asset is supported
	state := f.circuitBreakers[endpoint].GetState()
	f.metrics.RecordCircuitBreakerState(endpoint, int(state))

	// Execute the HTTP request with circuit breaker protection
	var response *http.Response
	var err error

	fetchErr := f.circuitBreakers[endpoint].Execute(func() error {
		response, err = f.client.Get(url)
		if err != nil {
			return err
		}

		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		return nil
	})

	// Record the response time
	duration := time.Since(startTime)
	f.metrics.ObserveExchangeRequestDuration(endpoint, duration)

	if fetchErr != nil {
		if fetchErr == circuitbreaker.ErrCircuitOpen {
			f.metrics.RecordExchangeError(endpoint, "circuit_open")
			return nil, fmt.Errorf("circuit open for endpoint %s", endpoint)
		}
		f.metrics.RecordExchangeError(endpoint, "request_error")
		return nil, fetchErr
	}

	defer response.Body.Close()

	var mockResp mockResponse
	if err := json.NewDecoder(response.Body).Decode(&mockResp); err != nil {
		f.metrics.RecordExchangeError(endpoint, "decode_error")
		return nil, err
	}

	return &mockResp, nil
}

// FetchPrice fetches the price for a symbol from mock exchanges and calculates a weighted average
func (f *fetcher) FetchPrice(symbol string) (*types.PriceData, error) {
	var responses []*mockResponse
	var errors []error

	// Fetch data from all endpoints
	for _, endpoint := range f.endpoints {
		resp, err := f.fetchFromEndpoint(endpoint, symbol)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		responses = append(responses, resp)
	}

	// Check if we have any valid responses
	if len(responses) == 0 {
		var errMsg string
		if len(errors) > 0 {
			errMsg = errors[0].Error()
			for i := 1; i < len(errors); i++ {
				errMsg += "; " + errors[i].Error()
			}
		} else {
			errMsg = ErrNoValidData.Error()
		}
		return nil, fmt.Errorf("%w: %s", ErrNoValidData, errMsg)
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
		return nil, ErrZeroVolume
	}

	priceData := &types.PriceData{
		Asset:     strings.ToLower(symbol),
		Price:     totalPrice / totalVolume,
		Timestamp: latestTimestamp,
	}

	return priceData, nil
}
