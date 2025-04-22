package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"real-time-price-aggregator/internal/api"
	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/metrics"
	"real-time-price-aggregator/internal/refresher"
	"real-time-price-aggregator/internal/storage"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// supportedAssets holds the list of supported asset symbols
var supportedAssets map[string]bool

// loadSymbols loads supported asset symbols from a CSV file
func loadSymbols(filename string) []string {
	// Initialize the map to store supported assets
	supportedAssets = make(map[string]bool)
	supportedList := []string{}

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
		supportedList = append(supportedList, asset)
	}
	log.Printf("Loaded %d symbols", len(supportedAssets))
	return supportedList
}

func main() {
	// Load symbols from CSV
	supportedList := loadSymbols("symbols.csv")
	log.Printf("Loaded %d symbols", len(supportedAssets))

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

	// Get exchange hosts from environment variables or use defaults
	exchange1 := os.Getenv("EXCHANGE1_URL")
	if exchange1 == "" {
		exchange1 = "http://exchange1:8081/mock/ticker" // Default for local
	}

	exchange2 := os.Getenv("EXCHANGE2_URL")
	if exchange2 == "" {
		exchange2 = "http://exchange2:8082/mock/ticker" // Default for local
	}

	exchange3 := os.Getenv("EXCHANGE3_URL")
	if exchange3 == "" {
		exchange3 = "http://exchange3:8083/mock/ticker" // Default for local
	}

	// Initialize metrics services
	metricsService := metrics.NewMetricsService()
	systemMetrics := metrics.NewSystemMetrics()

	systemMetrics.StartCollecting(5 * time.Second)

	// Initialize Fetcher with environment-specific URLs
	priceFetcher := fetcher.NewFetcher([]string{
		exchange1,
		exchange2,
		exchange3,
	}, metricsService)

	// Initialize Cache and Storage
	priceCache := cache.NewRedisCache(redisClient)
	priceStorage := storage.NewDynamoDBStorage(dynamoClient, systemMetrics)

	// Initialize Refresher service
	priceRefresher := refresher.NewRefresher(
		priceFetcher,
		priceCache,
		priceStorage,
		supportedList,
		metricsService,
	)

	// Assign refresh tiers to assets based on popularity (order in CSV)
	priceRefresher.AssignTiers()

	// Start the auto-refresh service
	priceRefresher.Start()
	defer priceRefresher.Stop() // Ensure proper cleanup on shutdown

	// Initialize API Handler with the refresher
	handler := api.NewHandler(
		priceFetcher,
		priceCache,
		priceStorage,
		priceRefresher,
		supportedAssets,
		metricsService,
	)
	handler.WarmupCache()

	// Set up routes
	r := mux.NewRouter()

	// Price API endpoints
	r.HandleFunc("/prices/{asset}", handler.GetPrice).Methods("GET")
	r.HandleFunc("/refresh/{asset}", handler.RefreshPrice).Methods("POST")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// increase the GC percent to 200% for testing
	debug.SetGCPercent(200)

	// Start server
	log.Println("Starting server on port 8080...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Fatal(http.ListenAndServe(":8080", r))
}
