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

	http.HandleFunc("/mock/ticker", func(w http.ResponseWriter, r *http.Request) {
		price := 50.0 + rand.Float64()*50.0 // 随机价格 50-100
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"symbol":"mock","price":%.2f,"time":%d}`, price, time.Now().Unix())
	})

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Mock server (%s) running on port %s\n", exchangeName, port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Failed to start mock server: %v\n", err)
		os.Exit(1)
	}
}
