package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/storage"
	"real-time-price-aggregator/internal/types"

	"github.com/gorilla/mux"
)

// Handler handles API requests
type Handler struct {
	// fetcher         fetcher.Fetcher
	cache           cache.Cache
	storage         storage.Storage
	supportedAssets map[string]bool
}

// NewHandler creates a new API handler
func NewHandler(c cache.Cache, s storage.Storage, supportedAssets map[string]bool) *Handler {
	return &Handler{
		cache:           c,
		storage:         s,
		supportedAssets: supportedAssets,
	}
}

// GetPrice handles GET /prices/{asset}
func (h *Handler) GetPrice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["asset"]
	if symbol == "" {
		RespondWithError(w, r, http.StatusBadRequest, "Asset symbol is required")
		return
	}

	// Convert to lowercase for case-insensitive comparison
	symbolLower := strings.ToLower(symbol)
	// Check if asset is supported (in CSV)
	if !h.supportedAssets[symbolLower] {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid asset symbol")
		return
	}

	// Get price from cache
	priceData, err := h.cache.Get(symbolLower)
	if err != nil {
		log.Printf("Failed to get price from cache for %s: %v", symbolLower, err)
		RespondWithError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Cache miss, try DynamoDB
	if priceData == nil {
		record, err := h.storage.Get(symbolLower)
		if err != nil {
			log.Printf("Failed to get price from DynamoDB for %s: %v", symbolLower, err)
			RespondWithError(w, r, http.StatusInternalServerError, "Internal server error")
			return
		}
		if record == nil {
			RespondWithError(w, r, http.StatusNotFound, "Asset not found")
			return
		}
		// Convert record to PriceData
		priceData = &types.PriceData{
			Asset:     record.Asset,
			Price:     record.Price,
			Timestamp: record.Timestamp,
		}
		// Update cache
		if err := h.cache.Set(symbolLower, priceData); err != nil {
			log.Printf("Failed to update cache for %s: %v", symbolLower, err)
		}
	}

	// Convert to response format with formatted timestamp
	priceResponse := priceData.ToResponse()
	respondWithJSON(w, http.StatusOK, priceResponse)
}

// RespondWithError sends an error response with the specified status code and message
func RespondWithError(w http.ResponseWriter, r *http.Request, status int, message string) {
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
