package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/storage"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/mux"
)

// Handler handles API requests
type Handler struct {
	fetcher fetcher.Fetcher // Using interface instead of pointer
	cache   cache.Cache     // Using interface instead of pointer
	storage storage.Storage // Using interface instead of pointer
}

// NewHandler creates a new API handler
func NewHandler(fetcher fetcher.Fetcher, cache cache.Cache, storage storage.Storage) *Handler {
	return &Handler{fetcher: fetcher, cache: cache, storage: storage}
}

// GetPrice handles GET /prices/{asset} requests
func (h *Handler) GetPrice(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["asset"]
	log.Printf("Received request for asset: %s", symbol)

	// Check cache
	cached, err := h.cache.Get(symbol)
	if err == nil {
		log.Printf("Cache hit for %s: %f", symbol, cached.Price)
		h.writeJSON(w, map[string]interface{}{
			"asset":        symbol,
			"price":        cached.Price,
			"last_updated": time.Unix(cached.Timestamp, 0).Format("2006-01-02 15:04:05"),
		})
		return
	}
	log.Printf("Cache miss for %s: %v", symbol, err)

	// Cache miss, fetch new data
	priceData, err := h.fetcher.FetchPrice(symbol)
	if err != nil {
		log.Printf("Error fetching price for %s: %v", symbol, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Fetched new price for %s: %f", symbol, priceData.Price)

	// Update cache
	cacheErr := h.cache.Set(symbol, cache.CachedPrice{
		Price:     priceData.Price,
		Timestamp: priceData.Timestamp,
	})
	if cacheErr != nil {
		log.Printf("Warning: Failed to update cache for %s: %v", symbol, cacheErr)
	} else {
		log.Printf("Updated cache for %s", symbol)
	}

	// Store in DynamoDB
	record := storage.PriceRecord{
		Asset:     symbol,
		Timestamp: priceData.Timestamp,
		Price:     priceData.Price,
		UpdatedAt: time.Now().Unix(),
	}

	log.Printf("Saving record to DynamoDB: %+v", record)
	saveErr := h.storage.Save(record)
	if saveErr != nil {
		log.Printf("Warning: Failed to save to DynamoDB for %s: %v", symbol, saveErr)
	} else {
		log.Printf("Saved to DynamoDB for %s", symbol)
	}

	// Return result
	h.writeJSON(w, map[string]interface{}{
		"asset":        symbol,
		"price":        priceData.Price,
		"last_updated": time.Unix(priceData.Timestamp, 0).Format("2006-01-02 15:04:05"),
	})
}

// RefreshPrice handles POST /refresh/{asset} requests
func (h *Handler) RefreshPrice(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["asset"]
	log.Printf("Received refresh request for asset: %s", symbol)

	// Add defer to catch panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in RefreshPrice: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	// Force fetch new data
	priceData, err := h.fetcher.FetchPrice(symbol)
	if err != nil {
		log.Printf("Error refreshing price for %s: %v", symbol, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Fetched price data: %+v", priceData)

	testRecord := storage.PriceRecord{
		Asset:     "TEST_BTCUSDT",
		Timestamp: time.Now().Unix(),
		Price:     12345.67,
		UpdatedAt: time.Now().Unix(),
	}
	log.Printf("Saving test record: %+v", testRecord)
	testErr := h.storage.Save(testRecord)
	if testErr != nil {
		log.Printf("Failed to save test record: %v", testErr)
	} else {
		log.Printf("Successfully saved test record")
	}

	// Update cache
	cacheErr := h.cache.Set(symbol, cache.CachedPrice{
		Price:     priceData.Price,
		Timestamp: priceData.Timestamp,
	})
	if cacheErr != nil {
		log.Printf("Warning: Failed to update cache during refresh for %s: %v", symbol, cacheErr)
	} else {
		log.Printf("Updated cache during refresh for %s", symbol)
	}

	// Store in DynamoDB
	record := storage.PriceRecord{
		Asset:     symbol,
		Timestamp: priceData.Timestamp,
		Price:     priceData.Price,
		UpdatedAt: time.Now().Unix(),
	}

	log.Printf("Saving refresh record to DynamoDB: %+v", record)
	saveErr := h.storage.Save(record)
	if saveErr != nil {
		log.Printf("Warning: Failed to save refresh to DynamoDB for %s: %v", symbol, saveErr)
	} else {
		log.Printf("Saved refresh to DynamoDB for %s", symbol)
		time.Sleep(5 * time.Second) // 等待 1 秒确保保存完成

		// 验证保存
		dbStorage, ok := h.storage.(*storage.DynamoDBStorage)
		if !ok {
			log.Printf("Error: Storage is not DynamoDBStorage")
			return
		}

		resp, err := dbStorage.GetClient().GetItem(&dynamodb.GetItemInput{
			TableName: aws.String("prices"),
			Key: map[string]*dynamodb.AttributeValue{
				"asset":     {S: aws.String(symbol)},
				"timestamp": {N: aws.String(fmt.Sprintf("%d", priceData.Timestamp))},
			},
		})
		if err != nil {
			log.Printf("Failed to verify saved record for %s: %v", symbol, err)
		} else if resp.Item == nil {
			log.Printf("Verification failed: No record found for %s with timestamp %d", symbol, priceData.Timestamp)
		} else {
			log.Printf("Verified: Record found for %s: %+v", symbol, resp.Item)
		}
	}

	h.writeJSON(w, map[string]string{
		"message": fmt.Sprintf("Price for %s refreshed", symbol),
	})
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
