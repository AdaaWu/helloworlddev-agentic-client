package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	// id 在整個 stream 內遞增
	seq int
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

// event 名稱對應前端 AgentEvent 的 discriminant
func (s *SSEWriter) sendJSON(event string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.seq++
	return s.send(SSEEvent{
		ID:    strconv.Itoa(s.seq),
		Event: event,
		Data:  string(b),
	})
}

// thinking、text 逐字送 Text；tool_call、tool_result 一次送 Payload
type Step struct {
	ID      string
	Type    string
	Text    string
	Payload map[string]any
}

// 模擬工具執行時間，前端 spinner 才看得到
const toolDelay = 800 * time.Millisecond

func buildScenario(message string) []Step {
	return []Step{
		{ID: "s1", Type: "thinking", Text: fmt.Sprintf("使用者說：「%s」。我先搜尋北北基的人口，再用計算機加總。", message)},
		{ID: "s2", Type: "tool_call", Payload: map[string]any{
			"toolCallId": "c1", "name": "search", "args": map[string]any{"q": "北北基 人口"},
		}},
		{ID: "s3", Type: "tool_result", Payload: map[string]any{
			"toolCallId": "c1", "output": map[string]any{"台北市": 2490000, "新北市": 4050000, "基隆市": 360000},
		}},
		{ID: "s4", Type: "tool_call", Payload: map[string]any{
			"toolCallId": "c2", "name": "calculator", "args": map[string]any{"expr": "2490000 + 4050000 + 360000"},
		}},
		{ID: "s5", Type: "tool_result", Payload: map[string]any{
			"toolCallId": "c2", "output": 6900000,
		}},
		{ID: "s6", Type: "text", Text: "根據搜尋結果，台北市約 249 萬人、新北市約 405 萬人、基隆市約 36 萬人，北北基合計約 690 萬人。"},
	}
}

func (s *SSEWriter) RunScenario(steps []Step, delay time.Duration) error {
	for _, st := range steps {
		switch st.Type {
		case "thinking", "text":
			for _, token := range tokenize(st.Text, 1) {
				err := s.sendJSON(st.Type, map[string]any{"stepId": st.ID, "delta": token})
				if err != nil {
					return err
				}
				if err := sleep(s.ctx, delay); err != nil {
					return err
				}
			}
		case "tool_call", "tool_result":
			payload := map[string]any{"stepId": st.ID}
			maps.Copy(payload, st.Payload)
			if err := s.sendJSON(st.Type, payload); err != nil {
				return err
			}
			if err := sleep(s.ctx, toolDelay); err != nil {
				return err
			}
		}
	}
	return s.sendJSON("done", map[string]any{})
}

func handleSSEOption(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	decoder := json.NewDecoder(r.Body)
	var req SSERequest
	err := decoder.Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")

	rc := http.NewResponseController(w)

	w.WriteHeader(http.StatusOK)
	rc.Flush()

	sseWriter := &SSEWriter{
		ctx: r.Context(),
		w:   w,
		rc:  rc,
	}

	err = sseWriter.RunScenario(buildScenario(req.Message), 40*time.Millisecond)
	if err != nil {
		// 使用者取消時連線已斷，不再寫入
		if errors.Is(err, context.Canceled) {
			log.Println("使用者取消")
			return
		}
		sseWriter.sendJSON("error", map[string]any{
			"stepId":    "server-error",
			"message":   err.Error(),
			"retryable": false,
		})
	}
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", handleRoot)
	router.HandleFunc("OPTIONS /sse", handleSSEOption)
	router.HandleFunc("POST /sse", handleSSE)
	fmt.Println("run on http://localhost:8081/sse")
	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server : %v\n", err)
	}
}
