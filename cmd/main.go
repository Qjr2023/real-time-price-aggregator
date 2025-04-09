package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"os"
	"strings"

	"real-time-price-aggregator/internal/api"
	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/storage"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// Supported assets (loaded from CSV)
var supportedAssets map[string]bool

func loadSymbols(filename string) {
	supportedAssets = make(map[string]bool)
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

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	// Test Redis connection
	if _, err := redisClient.Ping(redisClient.Context()).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize DynamoDB client
	dynamoClient := storage.NewDynamoDBClient()

	// Initialize Fetcher
	priceFetcher := fetcher.NewFetcher([]string{
		"http://exchange1:8081/mock/ticker",
		"http://exchange2:8082/mock/ticker",
		"http://exchange3:8083/mock/ticker",
	})

	// Initialize Cache and Storage
	priceCache := cache.NewRedisCache(redisClient)
	priceStorage := storage.NewDynamoDBStorage(dynamoClient)

	// Initialize API Handler
	handler := api.NewHandler(priceFetcher, priceCache, priceStorage, supportedAssets)

	// Set up routes
	r := mux.NewRouter()
	r.HandleFunc("/prices/{asset}", handler.GetPrice).Methods("GET")
	r.HandleFunc("/refresh/{asset}", handler.RefreshPrice).Methods("POST")
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Start server
	log.Println("Starting server on port 8080...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
