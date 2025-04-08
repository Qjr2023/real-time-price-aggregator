package main

import (
	"log"
	"net/http"
	"real-time-price-aggregator/internal/api"
	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/storage"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

func main() {
	log.Println("Starting Real-Time Price Aggregator service...")

	// Initialize Redis client
	log.Println("Connecting to Redis...")
	redisClient := redis.NewClient(&redis.Options{
		Addr: "redis:6379", // Use service name instead of localhost
	})

	// Test Redis connection with retries
	connected := false
	for i := 0; i < 5; i++ {
		_, err := redisClient.Ping(redisClient.Context()).Result()
		if err == nil {
			connected = true
			log.Println("Successfully connected to Redis")
			break
		}
		log.Printf("Failed to connect to Redis: %v, will retry in 2 seconds...", err)
		time.Sleep(2 * time.Second)
	}

	if !connected {
		log.Fatalf("Unable to connect to Redis, terminating service")
	}

	// Initialize DynamoDB client
	log.Println("Initializing DynamoDB client...")
	dynamoClient := storage.NewDynamoDBClient()

	// Initialize Fetcher
	log.Println("Initializing price fetcher...")
	priceFetcher := fetcher.NewFetcher([]string{
		"http://exchange1:8081/mock/ticker",
		"http://exchange2:8082/mock/ticker",
		"http://exchange3:8083/mock/ticker",
	})

	// Initialize Cache and Storage
	log.Println("Initializing cache and storage...")
	priceCache := cache.NewRedisCache(redisClient)
	priceStorage := storage.NewDynamoDBStorage(dynamoClient)

	// Initialize API Handler
	log.Println("Initializing API handler...")
	handler := api.NewHandler(priceFetcher, priceCache, priceStorage)

	// Set up routes
	log.Println("Setting up routes...")
	r := mux.NewRouter()
	r.HandleFunc("/prices/{asset}", handler.GetPrice).Methods("GET")
	r.HandleFunc("/refresh/{asset}", handler.RefreshPrice).Methods("POST")

	// Add health check endpoint
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
