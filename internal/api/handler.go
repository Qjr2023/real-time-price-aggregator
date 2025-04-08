package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/storage"

	"github.com/gorilla/mux"
)

// Handler handles API requests
type Handler struct {
	fetcher         fetcher.Fetcher
	cache           cache.Cache
	storage         storage.Storage
	supportedAssets map[string]bool
}

// NewHandler creates a new API handler
func NewHandler(f fetcher.Fetcher, c cache.Cache, s storage.Storage, supportedAssets map[string]bool) *Handler {
	return &Handler{
		fetcher:         f,
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

	// Get price from cache
	priceData, err := h.cache.Get(symbolLower)
	if err != nil {
		log.Printf("Failed to get price from cache for %s: %v", symbolLower, err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if priceData == nil {
		respondWithError(w, http.StatusNotFound, "Price not found")
		return
	}

	respondWithJSON(w, http.StatusOK, priceData)
}

// RefreshPrice handles POST /refresh/{asset}
func (h *Handler) RefreshPrice(w http.ResponseWriter, r *http.Request) {
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

	// Fetch price data
	priceData, err := h.fetcher.FetchPrice(symbolLower)
	if err != nil {
		log.Printf("Failed to fetch price for %s: %v", symbolLower, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch price")
		return
	}

	// Create price record
	record := storage.ToPriceRecord(priceData)

	// Save to DynamoDB
	if err := h.storage.Save(record); err != nil {
		log.Printf("Failed to save record for %s: %v", symbolLower, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to save price to DynamoDB")
		return
	}

	// Update cache
	if err := h.cache.Set(symbolLower, priceData); err != nil {
		log.Printf("Failed to update cache for %s: %v", symbolLower, err)
	}

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
