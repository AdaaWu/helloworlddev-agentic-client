package main

// demo-5（展示用）: mock LLM + 三種 scenario + 從指定 step 恢復
//
// 跟 demo-4 的差異 :
//  1. SSE 的 event 名稱改成 AgentStep 的種類 : thinking / tool_call / tool_result / text / error / done
//  2. data 從純文字改成 JSON，前端才能拿到 stepId、toolCallId、retryable 這些結構化欄位
//  3. request 多了 scenario（要跑哪個劇本）、resumeFrom（從哪個 step 重跑）、speed（吐字速度）
//  4. 使用者中途取消（前端 AbortController）時，r.Context() 會被 cancel，這裡就停止送出
//
// 測試 :
//   curl -N http://localhost:8080/sse -X POST -d '{"message":"Hi","scenario":"search_and_calc"}'
//   curl -N http://localhost:8080/sse -X POST -d '{"message":"Hi","scenario":"error_retry"}'
//   curl -N http://localhost:8080/sse -X POST -d '{"message":"Hi","scenario":"error_retry","resumeFrom":"s4"}'

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
	// 要跑哪個劇本 : search_and_calc / multi_tool / error_retry
	Scenario string `json:"scenario"`
	// 不是空字串時，代表前端要「從這個 step 開始重跑」，前面的 step 直接跳過
	ResumeFrom string `json:"resumeFrom"`
	// normal : 每個字 40ms（看得清楚）; fast : 每個字 2ms（模擬真實 LLM 爆發輸出，用來展示前端 16ms throttle）
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
	// SSE 的 id 在整個 stream 內遞增（demo-4 是每次從 0 開始）
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

// sendJSON : 把 payload 轉成 JSON 放進 data，event 名稱就是前端 AgentEvent 的 discriminant
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

// ===== mock LLM 劇本 =====

// Step : 劇本裡的一步
//   - thinking / text : Text 會被切成一個字一個字送出（delta）
//   - tool_call / tool_result : Payload 一次送出
//   - fail : 只有第一次（沒有 resumeFrom）會觸發，送出 error 後結束這次 stream
type Step struct {
	ID      string
	Type    string
	Text    string
	Payload map[string]any
}

// 模擬工具執行要花的時間，讓前端的 spinner 看得到
const toolDelay = 800 * time.Millisecond

func buildScenario(name, message string) ([]Step, bool) {
	switch name {
	case "search_and_calc":
		// 基本流程 : 思考 -> 搜尋 -> 拿到結果 -> 計算 -> 拿到結果 -> 回答
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
		// 一次呼叫三個工具，結果陸續回來 : 前端要用 toolCallId 把 tool_call 跟 tool_result 配對
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
		// 第一次 : 回答寫到一半就失敗，送出 retryable 的 error，並告訴前端從 s4 重跑
		// i二次（帶 resumeFrom: "s4"）: 從 s4 開始完整跑完，fail 這步會被跳過
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

// RunScenario : 照著劇本一步一步送出 SSE
func (s *SSEWriter) RunScenario(steps []Step, resumeFrom string, delay time.Duration) error {
	// 有 resumeFrom 時，先跳過前面已經完成的 step
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
			// 下一步是 fail 而且是第一次跑 : 只送一半，模擬「寫到一半斷掉」
			nextIsFail := i+1 < len(steps) && steps[i+1].Type == "fail"
			if resumeFrom == "" && nextIsFail {
				tokens = tokens[:len(tokens)/2]
			}
			for _, token := range tokens {
				err := s.sendJSON(st.Type, map[string]any{"stepId": st.ID, "delta": token})
				if err != nil {
					return err
				}
				// sleep 會監聽 ctx.Done() : 使用者取消時這裡會回傳 context.Canceled
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
			// 重跑時不再失敗
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

func HandleSSEOption(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func HandleSSE(w http.ResponseWriter, r *http.Request) {
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
	// 還沒開始 stream 前就檢查，錯誤可以用一般的 HTTP status 回應
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
		// 使用者主動取消（前端 abort）: 連線已經斷了，不要再寫任何東西
		if errors.Is(err, context.Canceled) {
			log.Printf("使用者取消 (scenario=%s)\n", req.Scenario)
			return
		}
		// 其他錯誤 : 用 error 事件告訴前端，這種錯誤不給重試
		sseWriter.sendJSON("error", map[string]any{
			"stepId":    "server-error",
			"message":   err.Error(),
			"retryable": false,
		})
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
