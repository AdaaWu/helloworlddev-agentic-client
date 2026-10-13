package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)

	w.Header().Set("Content-Type", "application/json")
	err := encoder.Encode(map[string]string{
		"message": "Hello World Dev",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", handleRoot)
	fmt.Println("run on http://localhost:8081")
	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server : %v\n", err)
	}
}
