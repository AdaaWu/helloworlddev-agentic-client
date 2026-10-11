package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)

	w.Header().Set("Content-Type", "application/json")
	err := encoder.Encode(map[string]string{
		"message": "Hello World Dev",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type SSERequest struct {
	Message string `json:"message"`
}

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

func formatEvent(ev SSEEvent) string {
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

func tokenize(text string, size int) []string {
	runes := []rune(text)
	result := make([]string, 0, len(runes)/size+1)
	for i := 0; i < len(runes); i += size {
		result = append(result, string(runes[i:min(i+size, len(runes))]))
	}
	return result
}

func (s *SSEWriter) write(msg string) error {
	_, err := io.WriteString(s.w, msg)
	if err != nil {
		return err
	}
	return s.rc.Flush()
}

func (s *SSEWriter) send(ev SSEEvent) error {
	return s.write(formatEvent(ev))
}

func (s *SSEWriter) SendAgentStep(event, message string, delay time.Duration) error {
	tokens := tokenize(message, 1)
	for i, token := range tokens {
		sseEvent := SSEEvent{
			ID:    strconv.Itoa(i),
			Event: event,
			Data:  token,
		}
		err := s.send(sseEvent)
		if err != nil {
			return err
		}
		err = sleep(s.ctx, delay)
		if err != nil {
			return err
		}
	}
	sseEvent := SSEEvent{
		ID:    strconv.Itoa(len(tokens)),
		Event: "done",
		Data:  "finish!",
	}
	return s.send(sseEvent)
}

func HandleSSEOption(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func HandleSSE(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var req SSERequest
	err := decoder.Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")

	rc := http.NewResponseController(w)

	w.WriteHeader(http.StatusOK)
	rc.Flush()

	ctx := r.Context()
	sseWriter := &SSEWriter{
		ctx: ctx,
		w:   w,
		rc:  rc,
	}
	message := fmt.Sprintf("收到的訊息 : %s", req.Message)

	err = sseWriter.SendAgentStep("message", message, 80*time.Millisecond)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("event: error\ndata: %s\n\n", err.Error())))
		rc.Flush()
		return
	}
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /", HandleRoot)
	router.HandleFunc("OPTIONS /sse", HandleSSEOption)
	router.HandleFunc("POST /sse", HandleSSE)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	fmt.Println("on http://localhost:8080/sse")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server : %v\n", err)
	}
}
