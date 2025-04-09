package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: mock_server <port> <exchange_name>")
		os.Exit(1)
	}

	port := os.Args[1]
	exchangeName := os.Args[2]

	http.HandleFunc("/mock/ticker/", func(w http.ResponseWriter, r *http.Request) {
		// Extract symbol from URL
		symbol := r.URL.Path[len("/mock/ticker/"):]
		if symbol == "" {
			http.Error(w, `{"error":"symbol is required"}`, http.StatusBadRequest)
			return
		}

		price := 50.0 + rand.Float64()*50.0            // Random price between 50 and 100
		volume := 1000000.0 + rand.Float64()*9000000.0 // Random volume between 1M and 10M
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"symbol":"%s","price":%.2f,"volume":%.2f,"timestamp":%d}`, symbol, price, volume, time.Now().Unix())
	})

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Mock server (%s) running on port %s\n", exchangeName, port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Failed to start mock server: %v\n", err)
		os.Exit(1)
	}
}
