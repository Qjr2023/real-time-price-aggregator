// internal/refresher/refresher.go
package refresher

import (
	"log"
	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/metrics"
	"real-time-price-aggregator/internal/storage"

	"sync"
	"time"
)

// AssetTier represents the refresh frequency tier of an asset
type AssetTier int

const (
	// HotTier assets refresh every 5 seconds
	HotTier AssetTier = iota
	// MediumTier assets refresh every 30 seconds
	MediumTier
	// ColdTier assets refresh every 5 minutes
	ColdTier
)

// RefreshInterval returns the time.Duration for a given tier
func (t AssetTier) RefreshInterval() time.Duration {
	switch t {
	case HotTier:
		return 5 * time.Second
	case MediumTier:
		return 30 * time.Second
	case ColdTier:
		return 5 * time.Minute
	default:
		return 5 * time.Minute
	}
}

// Refresher is responsible for periodically refreshing asset prices
type Refresher struct {
	fetcher       fetcher.Fetcher
	cache         cache.Cache
	storage       storage.Storage
	assetTiers    map[string]AssetTier
	stopChans     map[string]chan struct{}
	mutex         sync.Mutex
	isRunning     bool
	supportedList []string
	metrics       *metrics.MetricsService
}

// NewRefresher creates a new auto-refresher instance
func NewRefresher(
	f fetcher.Fetcher,
	c cache.Cache,
	s storage.Storage,
	supportedList []string,
	m *metrics.MetricsService,
) *Refresher {
	return &Refresher{
		fetcher:       f,
		cache:         c,
		storage:       s,
		assetTiers:    make(map[string]AssetTier),
		stopChans:     make(map[string]chan struct{}),
		supportedList: supportedList,
		metrics:       m,
	}
}

// AssignTiers assigns refresh tiers to assets based on their popularity
// Top 20 assets are hot, next 180 are medium, the rest are cold
func (r *Refresher) AssignTiers() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// For simplicity, we'll just use the order in the supportedList to determine "popularity"
	// In a real system, you might use trading volume or other metrics
	for i, asset := range r.supportedList {
		if i < 20 {
			r.assetTiers[asset] = HotTier
		} else if i < 200 {
			r.assetTiers[asset] = MediumTier
		} else {
			r.assetTiers[asset] = ColdTier
		}
	}
	log.Printf("Assigned tiers: %d hot, %d medium, %d cold",
		min(20, len(r.supportedList)),
		min(180, max(0, len(r.supportedList)-20)),
		max(0, len(r.supportedList)-200))
}

// Start begins the auto-refresh processes for all assets
func (r *Refresher) Start() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isRunning {
		return
	}

	log.Println("Starting auto-refresh service")
	r.isRunning = true

	// Start a refresh goroutine for each asset
	for _, asset := range r.supportedList {
		tier := r.assetTiers[asset]
		stop := make(chan struct{})
		r.stopChans[asset] = stop

		go r.refreshLoop(asset, tier, stop)
	}
}

// Stop halts all refresh processes
func (r *Refresher) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isRunning {
		return
	}

	log.Println("Stopping auto-refresh service")

	// Signal all refresh goroutines to stop
	for asset, stop := range r.stopChans {
		close(stop)
		delete(r.stopChans, asset)
	}

	r.isRunning = false
}

// refreshLoop periodically refreshes the price for a single asset
func (r *Refresher) refreshLoop(asset string, tier AssetTier, stop <-chan struct{}) {
	ticker := time.NewTicker(tier.RefreshInterval())
	defer ticker.Stop()

	// Initial refresh
	r.refreshAsset(asset)

	for {
		select {
		case <-ticker.C:
			r.refreshAsset(asset)
		case <-stop:
			return
		}
	}
}

// refreshAsset fetches the latest price for an asset and updates cache and storage
func (r *Refresher) refreshAsset(asset string) {
	// acquire lock to prevent concurrent access
	tier := r.assetTiers[asset]
	var tierString string
	switch tier {
	case HotTier:
		tierString = "hot"
	case MediumTier:
		tierString = "medium"
	case ColdTier:
		tierString = "cold"
	}

	// Fetch the latest price
	priceData, err := r.fetcher.FetchPrice(asset)
	if err != nil {
		r.metrics.RecordRefreshError(tierString)
		log.Printf("Failed to refresh price for %s: %v", asset, err)
		return
	}

	// Update cache
	if err := r.cache.Set(asset, priceData, tierString); err != nil {
		log.Printf("Failed to update cache for %s: %v", asset, err)
	}

	// Update storage
	record := storage.ConvertPriceDataToRecord(priceData)
	if err := r.storage.Save(record); err != nil {
		log.Printf("Failed to update storage for %s: %v", asset, err)
	}

	// Record the refresh operation
	r.metrics.RecordRefresh(tierString, "auto")
	log.Printf("Refreshed price for %s: %.2f", asset, priceData.Price)
}

// GetAssetTier returns the refresh tier for a given asset
func (r *Refresher) GetAssetTier(asset string) AssetTier {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.assetTiers[asset]
}

// ForceRefresh triggers an immediate refresh for a specific asset
// This can be used when a user requests data for an infrequently updated asset
func (r *Refresher) ForceRefresh(asset string) error {
	// Check if asset is supported
	found := false
	for _, a := range r.supportedList {
		if a == asset {
			found = true
			break
		}
	}
	if !found {
		return fetcher.ErrAssetNotSupported
	}

	// acquire lock to prevent concurrent access
	tier := r.assetTiers[asset]
	var tierString string
	switch tier {
	case HotTier:
		tierString = "hot"
	case MediumTier:
		tierString = "medium"
	case ColdTier:
		tierString = "cold"
	}

	// Fetch the latest price
	priceData, err := r.fetcher.FetchPrice(asset)
	if err != nil {
		r.metrics.RecordRefreshError(tierString)
		return err
	}

	// update cache
	if err := r.cache.Set(asset, priceData, tierString); err != nil {
		log.Printf("Failed to update cache for %s: %v", asset, err)
	}

	// update storage
	record := storage.ConvertPriceDataToRecord(priceData)
	if err := r.storage.Save(record); err != nil {
		log.Printf("Failed to update storage for %s: %v", asset, err)
	}

	// record the refresh operation
	r.metrics.RecordRefresh(tierString, "force")
	return nil
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// max returns the larger of x or y
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// refresher.go
func (r *Refresher) GetAllAssetTiers() map[string]AssetTier {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 返回副本以避免并发问题
	result := make(map[string]AssetTier)
	for k, v := range r.assetTiers {
		result[k] = v
	}
	return result
}
