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
	Message  string `json:"message"`
	Scenario string `json:"scenario"`
	// 有值時從這個 step 開始重跑
	ResumeFrom string `json:"resumeFrom"`
	// fast 模擬 LLM 爆發輸出，用來展示前端 16ms throttle
	Speed string `json:"speed"`
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
// fail 只在第一次跑時送出 error 並結束
type Step struct {
	ID      string
	Type    string
	Text    string
	Payload map[string]any
}

// 模擬工具執行時間，前端 spinner 才看得到
const toolDelay = 800 * time.Millisecond

func buildScenario(name, message string) ([]Step, bool) {
	switch name {
	case "search_and_calc":
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
		}, true
	case "multi_tool":
		// 結果回來的順序與呼叫順序不同，前端要用 toolCallId 配對
		return []Step{
			{ID: "s1", Type: "thinking", Text: fmt.Sprintf("使用者說：「%s」。我同時查天氣、新聞和股價，整理成一份早報。", message)},
			{ID: "s2", Type: "tool_call", Payload: map[string]any{
				"toolCallId": "c1", "name": "weather", "args": map[string]any{"city": "台北"},
			}},
			{ID: "s3", Type: "tool_call", Payload: map[string]any{
				"toolCallId": "c2", "name": "news", "args": map[string]any{"topic": "科技", "limit": 2},
			}},
			{ID: "s4", Type: "tool_call", Payload: map[string]any{
				"toolCallId": "c3", "name": "stock", "args": map[string]any{"symbol": "2330.TW"},
			}},
			{ID: "s5", Type: "tool_result", Payload: map[string]any{
				"toolCallId": "c2", "output": []string{"AI Agent 前端框架持續升溫", "Vue 3.5 效能更新"},
			}},
			{ID: "s6", Type: "tool_result", Payload: map[string]any{
				"toolCallId": "c1", "output": map[string]any{"temp": 26, "weather": "多雲"},
			}},
			{ID: "s7", Type: "tool_result", Payload: map[string]any{
				"toolCallId": "c3", "output": map[string]any{"price": 1085, "change": "+1.2%"},
			}},
			{ID: "s8", Type: "text", Text: "早安！台北今天多雲、約 26 度。科技新聞：AI Agent 前端框架持續升溫、Vue 3.5 效能更新。台積電目前 1085 元，上漲 1.2%。"},
		}, true
	case "error_retry":
		// 第一次回答寫到一半失敗，帶 resumeFrom s4 重跑才會完成
		return []Step{
			{ID: "s1", Type: "thinking", Text: fmt.Sprintf("使用者說：「%s」。我先查美金對台幣的匯率。", message)},
			{ID: "s2", Type: "tool_call", Payload: map[string]any{
				"toolCallId": "c1", "name": "exchange_rate", "args": map[string]any{"from": "USD", "to": "TWD"},
			}},
			{ID: "s3", Type: "tool_result", Payload: map[string]any{
				"toolCallId": "c1", "output": 32.1,
			}},
			{ID: "s4", Type: "text", Text: "目前匯率約 1 美元兌 32.1 新台幣，所以 100 美元大約等於 3210 新台幣。"},
			{ID: "s5", Type: "fail", Payload: map[string]any{
				"message": "LLM 連線中斷（模擬）", "retryable": true, "resumeFrom": "s4",
			}},
		}, true
	}
	return nil, false
}

func (s *SSEWriter) RunScenario(steps []Step, resumeFrom string, delay time.Duration) error {
	// 跳過 resumeFrom 之前已完成的 step
	started := resumeFrom == ""
	for i, st := range steps {
		if !started {
			if st.ID != resumeFrom {
				continue
			}
			started = true
		}

		switch st.Type {
		case "thinking", "text":
			tokens := tokenize(st.Text, 1)
			// 第一次跑且下一步是 fail，只送一半模擬寫到一半斷線
			nextIsFail := i+1 < len(steps) && steps[i+1].Type == "fail"
			if resumeFrom == "" && nextIsFail {
				tokens = tokens[:len(tokens)/2]
			}
			for _, token := range tokens {
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
		case "fail":
			if resumeFrom != "" {
				continue
			}
			payload := map[string]any{"stepId": st.ID}
			maps.Copy(payload, st.Payload)
			return s.sendJSON("error", payload)
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
	if req.Scenario == "" {
		req.Scenario = "search_and_calc"
	}
	// 開串流前驗證，錯誤才能用 HTTP status 回應
	steps, ok := buildScenario(req.Scenario, req.Message)
	if !ok {
		http.Error(w, "unknown scenario: "+req.Scenario, http.StatusBadRequest)
		return
	}
	delay := 40 * time.Millisecond
	if req.Speed == "fast" {
		delay = 2 * time.Millisecond
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

	err = sseWriter.RunScenario(steps, req.ResumeFrom, delay)
	if err != nil {
		// 使用者取消時連線已斷，不再寫入
		if errors.Is(err, context.Canceled) {
			log.Printf("使用者取消 (scenario=%s)\n", req.Scenario)
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
