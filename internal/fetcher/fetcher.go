package fetcher

import (
	"math/rand"
	"strings"
	"time"

	"real-time-price-aggregator/internal/types"
)

// Fetcher interface defines price fetching operations
type Fetcher interface {
	FetchPrice(symbol string) (*types.PriceData, error)
}

// fetcherImpl implements the Fetcher interface
type fetcherImpl struct {
	endpoints []string
}

// NewFetcher creates a new Fetcher instance
func NewFetcher(endpoints []string) Fetcher {
	return &fetcherImpl{endpoints: endpoints}
}

// FetchPrice fetches the price for a symbol (mock implementation)
func (f *fetcherImpl) FetchPrice(symbol string) (*types.PriceData, error) {
	// Mock implementation: return a random price
	price := 50.0 + rand.Float64()*50.0 // create a random price between 50 and 100
	return &types.PriceData{
		Asset:     strings.ToLower(symbol),
		Price:     price,
		Timestamp: time.Now().Unix(),
	}, nil
}
