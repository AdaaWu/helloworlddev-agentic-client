package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	encoder.Encode(map[string]string{
		"message": "Hello World Dev",
	})
}

func HandleSSEOption(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rc := http.NewResponseController(w)
	w.WriteHeader(http.StatusOK)
	rc.Flush()

	// ctx := r.Context()

	// 這段不需要了
	// ticker := time.NewTicker(1 * time.Second)
	// defer ticker.Stop()
	//
	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		fmt.Println("使用者斷開")
	// 		return
	// 	case t := <-ticker.C:
	// 		rawMessage := t.Format(time.RFC3339)
	// 		message := fmt.Sprintf("event: message\ndata: %s\n\n", rawMessage)
	// 		w.Write([]byte(message))
	// 		rc.Flush()
	// 	}
	// }
	// TODO: 從這開始做吧😃
	// 記得這裡要接 prompt 的 request 唷
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /", handleRoot)
	router.HandleFunc("OPTIONS /sse", HandleSSEOption)
	router.HandleFunc("POST /sse", handleSSE) // 我幫你改成 POST 了 😊
	fmt.Println("run on http://localhost:8080/sse")
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server : %v\n", err)
	}
}
