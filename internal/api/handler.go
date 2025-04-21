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
	// Maximum age of data before forcing a refresh (for cold tier assets)
	maxDataAge time.Duration
}

// statusRecorder 是一个自定义的 ResponseWriter，可以记录状态码
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
	return &Handler{
		fetcher:         f,
		cache:           c,
		storage:         s,
		refresher:       r,
		supportedAssets: supportedAssets,
		metrics:         m,
		maxDataAge:      5 * time.Minute, // Maximum acceptable age for cold tier data
	}
}

// WriteHeader 覆盖 http.ResponseWriter 的方法，记录状态码
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// GetPrice handles GET /prices/{asset}
// This is now a purely "Query" operation in CQRS
func (h *Handler) GetPrice(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	// 创建一个响应记录器，捕获状态码
	recorder := statusRecorder{w, http.StatusOK}

	// 函数结束时记录请求
	defer func() {
		h.metrics.RecordAPIRequest("/prices", recorder.status)
		h.metrics.ObserveAPIRequestDuration("/prices", time.Since(startTime))
	}()

	// 使用 recorder 代替 w 作为响应写入器
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

	// Get price from cache
	priceData, err := h.cache.Get(symbolLower)
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
		} else {
			// Found in storage but not in cache - convert and check age
			priceData = &types.PriceData{
				Asset:     record.Asset,
				Price:     record.Price,
				Timestamp: record.Timestamp,
			}

			// Update cache with storage data
			if err := h.cache.Set(symbolLower, priceData); err != nil {
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
	h.metrics.RecordAssetAccess(symbolLower, tierString)

	priceResponse := priceData.ToResponseWithTier(tierString)
	respondWithJSON(w, http.StatusOK, priceResponse)
}

// RefreshPrice handles POST /refresh/{asset}
// This is a "Command" operation in CQRS
func (h *Handler) RefreshPrice(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 创建一个响应记录器，捕获状态码
	recorder := statusRecorder{w, http.StatusOK}

	// 函数结束时记录请求
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

	// 获取资产的层级
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

	// 记录成功刷新
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
