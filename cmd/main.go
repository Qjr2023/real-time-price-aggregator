package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"real-time-price-aggregator/internal/api"
	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/storage"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// supportedAssets holds the list of supported asset symbols
var supportedAssets map[string]bool

// loadSymbols loads supported asset symbols from a CSV file
func loadSymbols(filename string) {
	// Initialize the map to store supported assets
	supportedAssets = make(map[string]bool)
	// Open the CSV file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open symbols file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read symbols file: %v", err)
	}

	for _, record := range records[1:] { // Skip header
		asset := strings.ToLower(record[0])
		supportedAssets[asset] = true
	}
	log.Printf("Loaded %d symbols", len(supportedAssets))
}

func main() {
	// Load symbols from CSV
	loadSymbols("symbols.csv")

	// Get Redis connection info from environment variables or use defaults
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379" // Default for local development
	}

	// Initialize Redis client with appropriate address
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test Redis connection
	if _, err := redisClient.Ping(redisClient.Context()).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize DynamoDB client
	dynamoClient := storage.NewDynamoDBClient()

	// Initialize Cache and Storage
	priceCache := cache.NewRedisCache(redisClient)
	priceStorage := storage.NewDynamoDBStorage(dynamoClient)

	// Initialize API Handler
	handler := api.NewHandler(priceCache, priceStorage, supportedAssets)

	// Set up routes
	r := mux.NewRouter()
	r.HandleFunc("/prices/{asset}", handler.GetPrice).Methods("GET")

	// add health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// add system info endpoint
	r.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"service":  "Price Aggregator API",
			"version":  "2.0",
			"features": []string{"Get cached price data", "Automatic tiered refresh via Lambda"},
			"note":     "Price refreshing is now handled by AWS Lambda with tiered refresh strategy",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(info)
	}).Methods("GET")

	// Start server
	log.Println("Starting server on port 8080...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
