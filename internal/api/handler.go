package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/metrics"
	"real-time-price-aggregator/internal/refresher"
	"real-time-price-aggregator/internal/storage"
	"real-time-price-aggregator/internal/types"

	"github.com/panjf2000/ants/v2"

	"github.com/gorilla/mux"
)

// Handler handles API requests
type Handler struct {
	fetcher         fetcher.Fetcher
	cache           cache.Cache
	storage         storage.Storage
	refresher       *refresher.Refresher
	supportedAssets map[string]bool
	metrics         *metrics.MetricsService
	pool            *ants.Pool
	// Maximum age of data before forcing a refresh (for cold tier assets)
	maxDataAge time.Duration
}

// statusRecorder is a custom http.ResponseWriter to capture the status code
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// NewHandler creates a new API handler
func NewHandler(
	f fetcher.Fetcher,
	c cache.Cache,
	s storage.Storage,
	r *refresher.Refresher,
	supportedAssets map[string]bool,
	m *metrics.MetricsService,
) *Handler {
	pool, _ := ants.NewPool(100) // Create a pool with 100 goroutines
	return &Handler{
		fetcher:         f,
		cache:           c,
		storage:         s,
		refresher:       r,
		supportedAssets: supportedAssets,
		metrics:         m,
		maxDataAge:      5 * time.Minute, // Maximum acceptable age for cold tier data
		pool:            pool,
	}
}

// WriteHeader captures the status code
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// GetPrice handles GET /prices/{asset}
// This is now a purely "Query" operation in CQRS
func (h *Handler) GetPrice(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	// create a response recorder to capture the status code
	recorder := statusRecorder{w, http.StatusOK}

	// when the function exits, record the request
	defer func() {
		h.metrics.RecordAPIRequest("/prices", recorder.status)
		h.metrics.ObserveAPIRequestDuration("/prices", time.Since(startTime))
	}()

	// Extract the asset symbol from the URL
	vars := mux.Vars(r)
	symbol := vars["asset"]
	if symbol == "" {
		respondWithError(&recorder, http.StatusBadRequest, "Asset symbol is required")
		return
	}

	// Convert to lowercase for case-insensitive comparison
	symbolLower := strings.ToLower(symbol)

	// Check if asset is supported (in CSV)
	if !h.supportedAssets[symbolLower] {
		respondWithError(w, http.StatusBadRequest, "Invalid asset symbol")
		return
	}

	tier := h.refresher.GetAssetTier(symbolLower)
	var tierString string
	switch tier {
	case refresher.HotTier:
		tierString = "hot"
	case refresher.MediumTier:
		tierString = "medium"
	case refresher.ColdTier:
		tierString = "cold"
	default:
		tierString = "medium" // 默认为中等层级
	}
	// Check if asset is supported
	var priceData *types.PriceData
	var err error
	priceData, err = h.cache.Get(symbolLower)
	if err != nil {
		log.Printf("Failed to get price from cache for %s: %v", symbolLower, err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Check if we need to trigger a refresh
	needsRefresh := false

	// Cache miss - try to get from storage and trigger refresh
	if priceData == nil {
		h.metrics.RecordCacheMiss()
		// Try to get from storage
		record, err := h.storage.Get(symbolLower)
		if err != nil {
			log.Printf("Failed to get price from storage for %s: %v", symbolLower, err)
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		if record == nil {
			// Neither in cache nor storage - trigger refresh
			needsRefresh = true
			if err := h.cache.Set(symbolLower, priceData, tierString); err != nil {
				log.Printf("Failed to update cache from storage for %s: %v", symbolLower, err)
			}
		} else {
			// Found in storage but not in cache - convert and check age
			priceData = &types.PriceData{
				Asset:     record.Asset,
				Price:     record.Price,
				Timestamp: record.Timestamp,
			}

			// Update cache with storage data
			if err := h.cache.Set(symbolLower, priceData, tierString); err != nil {
				log.Printf("Failed to update cache from storage for %s: %v", symbolLower, err)
			}

			// Check if data is stale
			dataAge := time.Since(time.Unix(record.Timestamp, 0))
			if dataAge > h.maxDataAge {
				needsRefresh = true
			}
		}
	} else {
		// Cache hit - check if data is stale for a cold tier asset
		// We don't need to check for hot/medium tier assets as they're auto-refreshed
		h.metrics.RecordCacheHit()
		tier := h.refresher.GetAssetTier(symbolLower)
		if tier == refresher.ColdTier {
			dataAge := time.Since(time.Unix(priceData.Timestamp, 0))
			if dataAge > h.maxDataAge {
				needsRefresh = true
			}
		}
	}

	// If we need fresh data, trigger a refresh
	if needsRefresh {
		// For cold tier assets or missing data, force an immediate refresh
		err := h.refresher.ForceRefresh(symbolLower)
		if err != nil {
			log.Printf("Failed to force refresh for %s: %v", symbolLower, err)
			if priceData == nil {
				// If we have no data at all, return an error
				respondWithError(w, http.StatusNotFound, "Asset data not available")
				return
			}
			// If we have stale data, continue with it
		} else {
			// Refresh succeeded, get fresh data from cache
			priceData, err = h.cache.Get(symbolLower)
			if err != nil || priceData == nil {
				log.Printf("Failed to get fresh data for %s after refresh: %v", symbolLower, err)
				// Fall back to previous data if available
				if priceData == nil {
					respondWithError(w, http.StatusNotFound, "Asset data not available")
					return
				}
			}
		}
	}

	// Convert to response format with formatted timestamp and time ago
	h.metrics.RecordAssetAccess(symbolLower, tierString)

	priceResponse := priceData.ToResponseWithTier(tierString)
	respondWithJSON(w, http.StatusOK, priceResponse)
}

func (h *Handler) WarmupCache() {
	log.Println("Starting cache warmup...")

	// 收集热门资产列表
	hotAssets := []string{}
	for asset, tier := range h.refresher.GetAllAssetTiers() {
		if tier == refresher.HotTier {
			hotAssets = append(hotAssets, asset)
		}
	}

	if len(hotAssets) == 0 {
		log.Println("No hot assets found for cache warmup")
		return
	}

	log.Printf("Warming up cache with %d hot assets", len(hotAssets))

	// 批量获取数据
	records, err := h.storage.BatchGet(hotAssets)
	if err != nil {
		log.Printf("Cache warmup failed: %v", err)
		return
	}

	// 填充缓存
	for asset, record := range records {
		priceData := &types.PriceData{
			Asset:     record.Asset,
			Price:     record.Price,
			Timestamp: record.Timestamp,
		}

		if err := h.cache.Set(asset, priceData, "hot"); err != nil {
			log.Printf("Failed to warm up cache for %s: %v", asset, err)
		}
	}

	log.Printf("Cache warmed up with %d hot assets", len(records))
}

// RefreshPrice handles POST /refresh/{asset}
// This is a "Command" operation in CQRS
func (h *Handler) RefreshPrice(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// create a response recorder to capture the status code
	recorder := statusRecorder{w, http.StatusOK}

	// when the function exits, record the request
	defer func() {
		h.metrics.RecordAPIRequest("/refresh", recorder.status)
		h.metrics.ObserveAPIRequestDuration("/refresh", time.Since(startTime))
	}()

	vars := mux.Vars(r)
	symbol := vars["asset"]
	if symbol == "" {
		respondWithError(w, http.StatusBadRequest, "Asset symbol is required")
		return
	}

	// Convert to lowercase for case-insensitive comparison
	symbolLower := strings.ToLower(symbol)

	// Check if asset exists in CSV
	if !h.supportedAssets[symbolLower] {
		respondWithError(w, http.StatusNotFound, "Asset not found")
		return
	}

	// Check if asset is supported
	tier := h.refresher.GetAssetTier(symbolLower)
	var tierString string
	switch tier {
	case refresher.HotTier:
		tierString = "hot"
	case refresher.MediumTier:
		tierString = "medium"
	case refresher.ColdTier:
		tierString = "cold"
	}

	// Force a refresh through the refresher service
	err := h.refresher.ForceRefresh(symbolLower)
	if err != nil {
		h.metrics.RecordRefreshError(tierString)
		log.Printf("Failed to refresh price for %s: %v", symbolLower, err)
		respondWithError(&recorder, http.StatusInternalServerError, "Failed to refresh price")
		return
	}

	// Update cache and storage
	h.metrics.RecordRefresh(tierString, "manual")

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Price for " + symbol + " refreshed",
	})
}

// respondWithError sends an error response with the specified status code and message
func respondWithError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"msg": message})
}

// respondWithJSON sends a JSON response with the specified status code and payload
func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
