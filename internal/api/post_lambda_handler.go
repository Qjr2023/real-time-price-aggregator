package api

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/storage"
)

// RefreshHandler structure includes all the necessary components for refreshing prices
type RefreshHandler struct {
	fetcher fetcher.Fetcher
	cache   cache.Cache
	storage storage.Storage
}

// used to store the current batch of low-priority assets
var (
	currentLowTierBatch int
	batchMutex          sync.Mutex
)

func init() {
	// initialize the current low-tier batch to 0
	currentLowTierBatch = 0
}

// NewRefreshHandler creates a new RefreshHandler instance
func NewRefreshHandler(f fetcher.Fetcher, c cache.Cache, s storage.Storage) *RefreshHandler {
	return &RefreshHandler{
		fetcher: f,
		cache:   c,
		storage: s,
	}
}

// RefreshPrice refreshes the price for a given asset symbol
func (h *RefreshHandler) RefreshPrice(symbol string) (string, int, error) {
	// use fetcher to get the price data
	priceData, err := h.fetcher.FetchPrice(symbol)
	if err != nil {
		log.Printf("Failed to fetch price for %s: %v", symbol, err)
		return "Failed to fetch price", 500, err
	}

	// create a new record for the price data
	record := storage.ConvertPriceDataToRecord(priceData)

	if err := h.storage.Save(record); err != nil {
		log.Printf("Failed to save record for %s: %v", symbol, err)
		return "Failed to save price to DynamoDB", 500, err
	}

	// update the cache with the new price data
	if err := h.cache.Set(symbol, priceData); err != nil {
		log.Printf("Failed to update cache for %s: %v", symbol, err)
		// continue even if cache update fails
	}

	return fmt.Sprintf("Price for %s refreshed", symbol), 200, nil
}

// RefreshAssetsByTier refreshes the prices for assets based on their tier
// high: top 20 assets
// medium: next 100 assets
// low: remaining 880 assets, using a round-robin strategy
// This function is called by the Lambda function
// to refresh the prices of assets in a specific tier
// It uses a round-robin strategy for low-tier assets
// to ensure that all assets are refreshed periodically
func (h *RefreshHandler) RefreshAssetsByTier(ctx context.Context, tier string) error {
	assets := GetAssetsByTier(tier)

	for _, asset := range assets {
		message, _, err := h.RefreshPrice(asset)
		if err != nil {
			log.Printf("Error refreshing price for %s: %v", asset, err)
		} else {
			log.Printf("Successfully refreshed price for %s: %s", asset, message)
		}
	}

	return nil
}

// GetAssetsByTier returns a list of asset symbols based on the specified tier
func GetAssetsByTier(tier string) []string {
	switch tier {
	case "high":
		// the top 20 high-frequency assets
		assets := make([]string, 20)
		for i := 0; i < 20; i++ {
			assets[i] = fmt.Sprintf("asset%d", i+1)
		}
		return assets

	case "medium":
		// the next 100 medium-frequency assets
		assets := make([]string, 100)
		for i := 0; i < 100; i++ {
			assets[i] = fmt.Sprintf("asset%d", i+21)
		}
		return assets

	case "low":
		// the remaining 880 low-frequency assets
		batchMutex.Lock()
		defer batchMutex.Unlock()

		batchSize := 100
		// 880 low-priority assets (asset121 to asset1000)
		totalAssets := 880
		totalBatches := (totalAssets + batchSize - 1) / batchSize // Round up

		// update the current batch
		currentLowTierBatch = (currentLowTierBatch + 1) % totalBatches

		// calculate the start and end index for the current batch
		startIndex := 121 + (currentLowTierBatch * batchSize)
		endIndex := startIndex + batchSize
		if endIndex > 1001 {
			endIndex = 1001
		}

		// generate the asset symbols for the current batch
		assets := make([]string, 0, endIndex-startIndex)
		for i := startIndex; i < endIndex; i++ {
			assets = append(assets, fmt.Sprintf("asset%d", i))
		}
		return assets

	default:
		return []string{}
	}
}

// IsValidAsset checks if the asset symbol is valid
func IsValidAsset(symbol string) bool {
	// simple validation: check if the symbol starts with "asset" and is followed by digits
	symbolLower := strings.ToLower(symbol)
	if !strings.HasPrefix(symbolLower, "asset") {
		return false
	}

	numStr := symbolLower[5:]
	// check if the remaining part is a number
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}
