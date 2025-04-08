package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type PriceData struct {
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
}

func main() {
	// Initialize random seed
	// rand.Seed(time.Now().UnixNano())

	// Get port and exchange name from command line arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run mock_server.go <port> <exchange_name>")
		os.Exit(1)
	}
	port := os.Args[1]
	exchangeName := os.Args[2]

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Add a health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// Add the ticker endpoint that matches the fetcher's expectation
	r.GET("/mock/ticker/:symbol", func(c *gin.Context) {
		symbol := c.Param("symbol")
		data := simulatePrice(symbol, exchangeName)

		// Log the request and response for debugging
		log.Printf("[%s] Received request for %s, responding with: price=%.2f, volume=%.2f",
			exchangeName, symbol, data.Price, data.Volume)

		// Return the response in the format expected by the fetcher
		c.JSON(200, gin.H{
			"symbol":    symbol,
			"price":     data.Price,
			"volume":    data.Volume,
			"timestamp": data.Timestamp,
		})
	})

	// Log the server start
	log.Printf("Starting %s mock exchange server on port %s", exchangeName, port)

	// Run the server
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func simulatePrice(symbol, exchangeName string) PriceData {
	var basePrice, fluctuation float64

	// Set base prices and fluctuations based on symbol
	switch symbol {
	case "BTCUSDT":
		basePrice = 79413.33
		fluctuation = 1000.0
	case "ETHUSDT":
		basePrice = 1593.79
		fluctuation = 50.0
	case "USDTUSD":
		basePrice = 0.9995
		fluctuation = 0.001
	case "XRPUSDT":
		basePrice = 0.98
		fluctuation = 0.05
	case "BNBUSDT":
		basePrice = 556.37
		fluctuation = 10.0
	case "USDCUSD":
		basePrice = 0.9999
		fluctuation = 0.001
	case "SOLUSDT":
		basePrice = 107.54
		fluctuation = 5.0
	case "DOGEUSDT":
		basePrice = 0.1521
		fluctuation = 0.01
	case "TRXUSDT":
		basePrice = 0.2317
		fluctuation = 0.005
	case "ADAUSDT":
		basePrice = 0.5806
		fluctuation = 0.02
	default:
		basePrice = 100.0
		fluctuation = 10.0
	}

	// Adjust price based on exchange
	var adjustment float64
	switch exchangeName {
	case "exchange1":
		adjustment = 1.01 // Slightly higher price
	case "exchange2":
		adjustment = 0.99 // Slightly lower price
	case "exchange3":
		adjustment = 1.0 // Normal price
	default:
		adjustment = 1.0
	}

	// Calculate final price with random fluctuation
	avgPrice := basePrice * adjustment
	price := avgPrice + (rand.Float64()-0.5)*fluctuation*2

	// Round to 2 decimal places
	price = float64(int(price*100)) / 100

	// Generate random volume
	volume := 1_000_000_000 + rand.Float64()*9_000_000_000

	// Return the price data
	return PriceData{
		Price:     price,
		Volume:    volume,
		Timestamp: time.Now().Unix(),
	}
}
