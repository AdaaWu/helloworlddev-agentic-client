package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type SSEEvent struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  string `json:"data"`
}

type SSEWriter struct {
	ctx context.Context
	w   http.ResponseWriter
	rc  *http.ResponseController
}

func (s *SSEWriter) send(ev SSEEvent) error {
	// todo
	return nil
}

// 這個可以用於每一段時間發一則
func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// 這個可以把 SSEEvent 轉成 string
func FormatEvent(ev SSEEvent) string {
	var b strings.Builder
	newLine := "\n"
	if ev.ID != "" {
		b.WriteString("id: " + ev.ID + newLine)
	}

	if ev.Event != "" {
		b.WriteString("event: " + ev.Event + newLine)
	}
	for _, line := range strings.Split(ev.Data, "\n") {
		b.WriteString("data: " + line + newLine)
	}
	b.WriteString(newLine)
	return b.String()
}

// 這個可以把要傳的 string 化成 string array
func tokenize(text string, size int) []string {
	runes := []rune(text)
	result := make([]string, 0, len(text)/size+1)
	for i := 0; i < len(runes); i += size {
		result = append(result, string(runes[i:min(i+size, len(runes))]))
	}
	return result
}

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

	rc := http.NewResponseController(w)
	w.WriteHeader(http.StatusOK)
	rc.Flush()

	ctx := r.Context()

	// 這個用不到了 -> 我幫你註解了 😊
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
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /", handleRoot)
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
