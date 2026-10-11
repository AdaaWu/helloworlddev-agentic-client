package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	encoder.Encode(map[string]string{
		"message": "Hello World Dev",
	})
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher := http.NewResponseController(w)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("使用者斷開")
			return
		case t := <-ticker.C:
			rawMessage := t.Format(time.RFC3339)
			message := fmt.Sprintf("event: message\ndata: %s\n\n", rawMessage)
			w.Write([]byte(message))
			flusher.Flush()
		}
	}
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /", handleRoot)
	router.HandleFunc("GET /sse", handleSSE)
	fmt.Println("run on http://localhost:8080/sse")
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server : %v\n", err)
	}
}
