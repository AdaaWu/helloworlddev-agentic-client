# Agentic AI 前端 Workshop 逐步教學

- 每一步對應一個 commit，點「看程式碼變更」可看該步 diff
- ⓪～③ 在 `tutorial/01-get-sse/`，④ 之後在 `tutorial/02-post-stream/`
- 後端 port `:8081`，前端 `bun run dev` 開在 `:5173`

## 目錄

- [教學⓪ 後端建立 Go module](#e32050b)
- [教學⓪ 前端建立 Vite Vue TS 專案](#99965bb)
- [教學① 後端 hello world](#7e219a7)
- [教學② 後端 GET SSE 每秒推送時間](#48cce44)
- [教學③ 前端 EventSource 接收 GET SSE](#894574e)
- [教學④ 前置 複製 01 作為 POST 版起點](#34c8278)
- [教學④ 後端 SSE 路由改為 POST](#e6c72e7)
- [教學④ 後端補 OPTIONS 預檢](#a384372)
- [教學④ 前端改用 fetch 讀取 POST 串流](#cc4592e)
- [教學⑤ 前端 SSE parser 依事件分派](#1ea923b)
- [教學⑥ 後端 SSEWriter 逐字送出 AgentStep](#958e656)
- [教學⑥ 前端以 JSON body 送出輸入訊息](#75d54a2)
- [教學⑥ 前端聊天畫面逐字顯示回應](#3a4d5e7)
- [教學⑦-1 後端改送 AgentStep 事件與 JSON data](#04e9173)
- [教學⑦-2 後端 mock LLM 劇本選擇與吐字速度](#ce1ccc0)
- [教學⑦-3 後端 error 帶 retryable 並可從指定 step 重跑](#bd62da3)
- [教學⑦-4 前端 AgentEvent 與 AgentStep 型別](#6ff024e)
- [教學⑦-5 前端符合 SSE 規格的串流 parser](#edea60c)
- [教學⑦-6 前端 applyEvent 把事件套用成步驟](#84706f8)
- [教學⑦-7 前端文字 delta 以 16ms 合批](#5cdb9de)
- [教學⑦-8 前端 useAgent 管理請求與步驟狀態](#9e8ab8a)
- [教學⑦-9 前端 AgentTimeline 元件與畫面組裝](#c5533eb)
- [教學⑦-10 前端停止、Ctrl+C 與從指定 step 重試](#7259b9c)

---

<a id="e32050b"></a>

## 教學⓪ 後端建立 Go module

- 目標：建立 Go module 與空的 main 當起點
- 重點：go mod init tutorial，go 版本對齊解答 1.27.1
- 驗收：go vet ./... 與 go build 皆通過
- 卡點：Go 低於 1.27 會自動下載 toolchain，網路慢會卡住

**關鍵程式碼**

```sh
cd tutorial/01-get-sse/backend
go mod init tutorial
```

`tutorial/01-get-sse/backend/go.mod`

```text
module tutorial

go 1.27.1
```

`tutorial/01-get-sse/backend/main.go`

```go
package main

func main() {
}
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/e32050b4cc4c037aa4b0cdeb2958e44a83e19663)

---

<a id="99965bb"></a>

## 教學⓪ 前端建立 Vite Vue TS 專案

- 目標：用 create-vite 的 vue-ts 樣板建立前端
- 重點：刪掉 HelloWorld 元件與 style.css
- 重點：App.vue 只留空的 script setup 與 template
- 驗收：bun install 後 bun run build 通過
- 卡點：資料夾沒有 node_modules，要先 bun install 才能 dev

**關鍵程式碼**

```sh
cd tutorial/01-get-sse
bun create vite frontend --template vue-ts
cd frontend && bun install
```

`tutorial/01-get-sse/frontend/src/App.vue`

```vue
<script setup lang="ts">
</script>

<template>
</template>
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/99965bb016db56c80d382ddded6f6b3d7ac49dad)

---

<a id="7e219a7"></a>

## 教學① 後端 hello world

- 目標：GET / 回 JSON 訊息 Hello World Dev
- 重點：Go 1.22 起路由可寫 method 與路徑，如 GET /{$}
- 重點：先設 Content-Type 再 Encode，Encode 失敗回 500
- 驗收：curl -i localhost:8081/ 回 200 與 JSON
- 驗收：curl -i localhost:8081/abc 回 404
- 卡點：{$} 只匹配根路徑，少了它會吃掉所有路徑
- 卡點：本機 8080 被占用，全程改用 8081

**關鍵程式碼**

`tutorial/01-get-sse/backend/main.go`

```go
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
```

```go
func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", handleRoot)
	fmt.Println("run on http://localhost:8081")
	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}
	// ...
}
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/7e219a73fbc739f9412086a9705d04a093722d6f)

---

<a id="48cce44"></a>

## 教學② 後端 GET SSE 每秒推送時間

- 目標：GET /sse 每秒推送 RFC3339 時間
- 重點：text/event-stream、no-cache、keep-alive、CORS 四個 header
- 重點：WriteHeader 後先 Flush 一次，連線立即建立
- 重點：每次 Write 後 Flush，資料才會即時送出
- 重點：select 同時等 ctx.Done 與 ticker，斷線即 return
- 驗收：curl -N localhost:8081/sse 每秒多一筆 data
- 驗收：curl 中斷後後端印出「使用者斷開」
- 卡點：結尾要兩個換行，不然 EventSource 不觸發
- 卡點：curl 沒加 -N 會被 buffer 擋住看不出串流

**關鍵程式碼**

`tutorial/01-get-sse/backend/main.go`

```go
func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rc := http.NewResponseController(w)
	w.WriteHeader(http.StatusOK)
	rc.Flush()
	// ...
}
```

```go
	for {
		select {
		case <-ctx.Done():
			fmt.Println("使用者斷開")
			return
		case t := <-ticker.C:
			rawMessage := t.Format(time.RFC3339)
			message := fmt.Sprintf("event: message\ndata: %s\n\n", rawMessage)
			w.Write([]byte(message))
			rc.Flush()
		}
	}
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/48cce444f46ad1d6d9ecd50f718a24413815a170)

---

<a id="894574e"></a>

## 教學③ 前端 EventSource 接收 GET SSE

- 目標：瀏覽器用 EventSource 收後端每秒推送的時間
- 重點：後端網址集中成 API_BASE 常數
- 重點：addEventListener('message') 取 ev.data 印到 console
- 驗收：後端與 bun run dev 都開，Console 每秒一筆時間
- 驗收：Network 點 sse，EventStream 分頁逐筆可見
- 卡點：畫面空白是正常，要開 DevTools 看 Console
- 卡點：CORS 報錯代表後端少了 Allow-Origin

**關鍵程式碼**

`tutorial/01-get-sse/frontend/src/App.vue`

```ts
const API_BASE = 'http://localhost:8081';

const sse = new EventSource(`${API_BASE}/sse`);
sse.addEventListener('message', (ev) => {
  const data = ev.data;
  console.log('你收到的訊息在這 : ', data);
})
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/894574e85b33afe0bd9077859dc0822bdb004acd)

---

<a id="34c8278"></a>

## 教學④ 前置 複製 01 作為 POST 版起點

- 從 01 複製作為 POST 版起點

**關鍵程式碼**

無程式碼變更，見指令

```sh
cp -R tutorial/01-get-sse tutorial/02-post-stream
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/34c8278a98e0722d2fcb68b889c8b8f1d0d37cd5)

---

<a id="e6c72e7"></a>

## 教學④ 後端 SSE 路由改為 POST

- 目標：/sse 改用 POST，之後才能帶 prompt
- 重點：只改路由的 method，handler 內容不動
- 驗收：curl -N -X POST localhost:8081/sse 每秒一筆
- 驗收：curl -i localhost:8081/sse 回 405
- 卡點：改成 POST 後 EventSource 只能 GET，前端會失效

**關鍵程式碼**

`tutorial/02-post-stream/backend/main.go`

```go
func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", handleRoot)
	router.HandleFunc("POST /sse", handleSSE)
	// ...
}
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/e6c72e78a6981a468d89f06c2a880178bc485781)

---

<a id="a384372"></a>

## 教學④ 後端補 OPTIONS 預檢

- 目標：OPTIONS /sse 回 204 與 CORS 允許清單
- 重點：Allow-Methods 為 OPTIONS, POST
- 重點：Allow-Headers 為 Content-Type
- 重點：此時前端還不會觸發預檢，⑥ 帶 JSON 才用到
- 驗收：curl -i -X OPTIONS localhost:8081/sse 回 204
- 卡點：前端帶 application/json 時瀏覽器先送 OPTIONS

**關鍵程式碼**

`tutorial/02-post-stream/backend/main.go`

```go
func handleSSEOption(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}
```

```go
	router.HandleFunc("OPTIONS /sse", handleSSEOption)
	router.HandleFunc("POST /sse", handleSSE)
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/a3843726f916be7fa29e427edbd44bd00f4dd27e)

---

<a id="cc4592e"></a>

## 教學④ 前端改用 fetch 讀取 POST 串流

- 目標：按鈕觸發 POST，逐段印出原始串流文字
- 重點：移除 EventSource，它只能發 GET
- 重點：先檢查 response.ok 與 response.body
- 重點：body.getReader 搭配 TextDecoder 逐段解碼
- 重點：decode 帶 stream: true，跨 chunk 中文不亂碼
- 重點：finally 內 releaseLock 釋放 reader
- 驗收：按「發射！！」後 Console 每秒印出 event 與 data 兩行
- 卡點：done 為 true 沒 return 會無限迴圈

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/App.vue`

```ts
  const response = await fetch(`${API_BASE}/sse`, {
    method: 'POST',
    headers: {
      'Accept': 'text/event-stream',
    }
  });
  if (!response.ok) {
    console.log('不 ok');
    return;
  }
  // ...
```

```ts
  const reader = body.getReader();
  const decoder = new TextDecoder();
  try {
    while(true) {
      const { value, done } = await reader.read();
      if(done) return;
      const text = decoder.decode(value, { stream: true });
      console.log(text);
    }
  } finally {
    reader.releaseLock();
  }
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/cc4592eb1022562a605e64a72e9257928a1e76eb)

---

<a id="1ea923b"></a>

## 教學⑤ 前端 SSE parser 依事件分派

- 目標：把原始串流切成一筆筆事件，依 event 呼叫 callback
- 重點：eventStatusList 用 as const 推出 EventStatus
- 重點：isEvent、isSSEEventKey 型別守衛擋掉未知值
- 重點：buffer 累加後以 \n\n 切塊，pop 把半塊放回 buffer
- 重點：indexOf(':') 只切第一個冒號，value 只去一個開頭空白
- 重點：多行 data 先收進 dataList，最後 join('\n')
- 重點：switch 依 event 分派到 onMessage 等 callback
- 驗收：Console 每秒印「收到的訊息 : 時間」，無 event、data 前綴
- 驗收：時間的 +08:00 完整保留
- 卡點：用 split(':') 會把時間切壞，用 trim 會吃掉內容空白
- 卡點：沒有 buffer 時一個 chunk 可能只有半筆事件

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/App.vue`

```ts
      buffer += decoder.decode(value, { stream: true });
      const textList = buffer.split('\n\n');
      buffer = textList.pop() ?? '';
      handleEvent(textList, opts);
```

```ts
    for(const line of lineList) {
      const idx = line.indexOf(':');
      if (idx === -1) continue;
      const key = line.slice(0, idx);
      const rawValue = line.slice(idx+1);
      const value = rawValue.startsWith(' ') ? rawValue.slice(1) : rawValue;
      setResult(result, { field: isSSEEventKey(key) ? key : 'data', value, dataList });
    }
    result.data = dataList.join('\n');
    triggerEvent(result, opts);
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/1ea923b91f9e68a36df15acc6a895689e5c5d882)

---

<a id="958e656"></a>

## 教學⑥ 後端 SSEWriter 逐字送出 AgentStep

- 目標：讀 body 的 message，逐字回傳並以 done 收尾
- 重點：formatEvent 有值才寫 id、event，多行 data 逐行輸出
- 重點：SSEWriter 包裝 write 加 Flush 與 send
- 重點：tokenize 先轉 []rune 再切，中文才不會切壞
- 重點：sleep 用 timer 搭配 ctx.Done，斷線立即中止
- 重點：先 decode body，失敗回 400，成功才 WriteHeader(200)
- 重點：移除 ticker 迴圈，改呼叫 SendAgentStep
- 驗收：POST {"message":"嗨"} 逐字收到，最後 event: done
- 驗收：body 壞掉或沒 body 都回 400
- 卡點：先 WriteHeader(200) 再驗證，400 就送不出去
- 卡點：sleep 沒用 s.ctx，斷線後仍會繼續跑

**關鍵程式碼**

`tutorial/02-post-stream/backend/main.go`

```go
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
		// ...
	}
	// ...
}
```

```go
func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	decoder := json.NewDecoder(r.Body)
	var req SSERequest
	err := decoder.Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// ...
	err = sseWriter.SendAgentStep("message", message, 80*time.Millisecond)
	// ...
}
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/958e6561fa9393af8c2fd5b95535b3f59f1821b1)

---

<a id="75d54a2"></a>

## 教學⑥ 前端以 JSON body 送出輸入訊息

- 目標：textarea 輸入的訊息用 POST JSON 送給後端
- 重點：inputMessage 用 shallowRef 綁 v-model
- 重點：fetch 補 Content-Type JSON 與 JSON.stringify body
- 重點：空白訊息不送出，串流結束清空輸入框
- 驗收：輸入中文送出，Console 逐字印出收到的訊息
- 驗收：Network 先看到 OPTIONS 再看到 POST
- 卡點：帶 application/json 會觸發預檢，後端要有 OPTIONS

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/App.vue`

```ts
  if (!inputMessage.value.trim()) return;
  const req: SSERequest = {
    message: inputMessage.value,
  }
  const response = await fetch(`${API_BASE}/sse`, {
    method: 'POST',
    headers: {
      'Accept': 'text/event-stream',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(req),
  });
```

```vue
<template>
  <textarea v-model="inputMessage"></textarea>
  <button @click="handleClick">送出</button>
</template>
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/75d54a2047871418fcddcf6d7d323b64813b2e33)

---

<a id="3a4d5e7"></a>

## 教學⑥ 前端聊天畫面逐字顯示回應

- 目標：使用者訊息與 AI 回應以聊天框呈現，AI 逐字長出
- 重點：messageList 用 ref 存 user 與 ai 訊息
- 重點：onMessage 最後一則是 ai 就接字，否則新增一則
- 重點：onDone 在 Console 印出回應完成
- 重點：v-for 渲染訊息，依 type 切換樣式
- 驗收：輸入中文送出，AI 回應逐字出現且無亂碼
- 卡點：每個 token 都 push 新訊息，畫面會一字一框

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/App.vue`

```ts
    onMessage: (payload: string) => {
      const currentMessageList = messageList.value;
      const lastMessage = currentMessageList[currentMessageList.length - 1];
      if (lastMessage.type === 'ai') {
        lastMessage.message += payload;
        return;
      }
      currentMessageList.push({
        type: 'ai',
        message: payload,
      })
    },
```

```vue
    <div v-for="(messageObject, index) in messageList"
      :key="index"
      class="chat-block"
      :class="{ 'chat-ai': messageObject.type === 'ai' }"
    >
      <!-- ... -->
      {{ messageObject.message }}
    </div>
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/3a4d5e7d5f3f47875028ef594be4a2c84881553c)

---

<a id="04e9173"></a>

## 教學⑦-1 後端改送 AgentStep 事件與 JSON data

- 目標：event 改成 thinking、tool_call、tool_result、text、error、done
- 重點：sendJSON 把 payload 轉 JSON 放進 data，id 整條 stream 遞增
- 重點：Step 劇本固定跑 search_and_calc
- 重點：thinking、text 逐字送 delta，工具事件一次送出並等 800ms
- 重點：使用者取消不再寫入，其他錯誤送不可重試的 error
- 驗收：curl 依序看到 thinking、tool_call、tool_result、text、done
- 卡點：取消後還寫 error 事件，連線已斷只會再噴錯

**關鍵程式碼**

`tutorial/02-post-stream/backend/main.go`

```go
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
```

```go
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
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/04e91735a619b62679655abb5be0ca277a827f84)

---

<a id="ce1ccc0"></a>

## 教學⑦-2 後端 mock LLM 劇本選擇與吐字速度

- 目標：request 帶 scenario 與 speed，選劇本並調整吐字速度
- 重點：buildScenario 依名稱回傳劇本，新增 multi_tool
- 重點：multi_tool 三個工具結果亂序回來，靠 toolCallId 對應
- 重點：scenario 空值預設 search_and_calc，未知名稱回 400
- 重點：開串流前先驗證，錯誤才能用 HTTP status 回應
- 重點：speed fast 每字 2ms，其他 40ms
- 驗收：multi_tool 收到三個 tool_call 後結果依 c2、c1、c3 回來
- 驗收：scenario 亂填回 400 unknown scenario
- 卡點：先 WriteHeader(200) 再驗證劇本，400 送不出去

**關鍵程式碼**

`tutorial/02-post-stream/backend/main.go`

```go
func buildScenario(name, message string) ([]Step, bool) {
	switch name {
	case "search_and_calc":
		// ...
	case "multi_tool":
		// 結果回來的順序與呼叫順序不同，前端要用 toolCallId 配對
		// ...
	}
	return nil, false
}
```

```go
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
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/ce1ccc0d700d869c0b5eafa0c4de68404d0aaa7c)

---

<a id="bd62da3"></a>

## 教學⑦-3 後端 error 帶 retryable 並可從指定 step 重跑

- 目標：error_retry 劇本中途失敗，帶 resumeFrom 從該 step 重跑
- 重點：request 新增 resumeFrom，RunScenario 跳過之前的 step
- 重點：第一次跑遇到 fail，前一段文字只送一半再送 error
- 重點：error 帶 retryable 與 resumeFrom，告訴前端從哪重試
- 重點：帶 resumeFrom 重跑時 fail 直接 continue
- 驗收：error_retry 收到 retryable true、resumeFrom s4 的 error
- 驗收：帶 resumeFrom s4 只送 s4 文字並以 done 收尾
- 卡點：重跑沒跳過 fail，會一直失敗無法完成

**關鍵程式碼**

`tutorial/02-post-stream/backend/main.go`

```go
	// 跳過 resumeFrom 之前已完成的 step
	started := resumeFrom == ""
	for i, st := range steps {
		if !started {
			if st.ID != resumeFrom {
				continue
			}
			started = true
		}
		// ...
			// 第一次跑且下一步是 fail，只送一半模擬寫到一半斷線
			nextIsFail := i+1 < len(steps) && steps[i+1].Type == "fail"
			if resumeFrom == "" && nextIsFail {
				tokens = tokens[:len(tokens)/2]
			}
```

```go
		case "fail":
			if resumeFrom != "" {
				continue
			}
			payload := map[string]any{"stepId": st.ID}
			maps.Copy(payload, st.Payload)
			return s.sendJSON("error", payload)
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/bd62da3e3d4fe627b5215db911c9fa33a6e88cc8)

---

<a id="6ff024e"></a>

## 教學⑦-4 前端 AgentEvent 與 AgentStep 型別

- 目標：用型別描述後端事件與畫面步驟兩層狀態
- 重點：AgentEvent 以 event 為判別欄位，data 依事件不同
- 重點：AgentStep 以 type 為判別欄位，delta 累積成 content
- 重點：agentEventNames 給 runtime 檢查，satisfies 對齊型別
- 重點：AgentRequest 帶 scenario、speed、resumeFrom
- 重點：assertNever 做 switch 窮舉檢查
- 驗收：bun run build 綠燈
- 卡點：事件與步驟共用一個型別，delta 無法累積成一個 step

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/types/agent.ts`

```ts
// 後端事件：event 是判別欄位，data 是 JSON
export type AgentEvent =
  | { event: 'thinking'; data: { stepId: string; delta: string } }
  | { event: 'tool_call'; data: { stepId: string; toolCallId: string; name: string; args: Record<string, unknown> } }
  | { event: 'tool_result'; data: { stepId: string; toolCallId: string; output: unknown } }
  | { event: 'text'; data: { stepId: string; delta: string } }
  | { event: 'error'; data: { stepId: string; message: string; retryable: boolean; resumeFrom?: string } }
  | { event: 'done'; data: Record<string, never> };
```

```ts
// 畫面步驟：多個 delta 事件累積成一個 step
export type AgentStep =
  | (BaseStep & { type: 'thinking'; content: string })
  | (BaseStep & { type: 'tool_call'; toolCallId: string; name: string; args: Record<string, unknown> })
  | (BaseStep & { type: 'tool_result'; toolCallId: string; output: unknown })
  | (BaseStep & { type: 'text'; content: string })
  | (BaseStep & { type: 'error'; message: string; retryable: boolean; resumeFrom?: string });

// 放在 switch default，漏寫 case 會編譯錯誤
export function assertNever(x: never): never {
  throw new Error(`未處理的型別 : ${JSON.stringify(x)}`);
}
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/6ff024edff8c0d173846efa9b7234e3c48f84be0)

---

<a id="edea60c"></a>

## 教學⑦-5 前端符合 SSE 規格的串流 parser

- 目標：把 POST SSE 串流解析成 AgentEvent
- 重點：parseBlock 略過註解行與不認識的欄位
- 重點：不在 agentEventNames 的事件回傳 null 不處理
- 重點：data JSON 壞掉轉成不可重試的 error 事件
- 重點：streamAgent 把 signal 交給 fetch，HTTP 失敗用 throw 往上丟
- 重點：\r\n 先統一成 \n，結束時處理剩餘 buffer
- 驗收：bun run build 綠燈
- 卡點：最後一則沒有 \n\n 結尾，沒處理剩餘 buffer 會漏掉

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/lib/sse.ts`

```ts
  if (!isAgentEventName(event)) return null;

  try {
    const data: unknown = JSON.parse(dataList.join('\n') || '{}');
    // 信任後端格式，正式產品可改用 type guard 或 schema 驗證
    return { event, data } as AgentEvent;
  } catch {
    return {
      event: 'error',
      data: { stepId: crypto.randomUUID(), message: `無法解析的事件 : ${event}`, retryable: false },
    };
  }
```

```ts
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      // 其他 server 可能送 \r\n，先統一成 \n
      buffer += decoder.decode(value, { stream: true }).replace(/\r\n?/g, '\n');
      const blockList = buffer.split('\n\n');
      buffer = blockList.pop() ?? '';
      handleBlocks(blockList);
    }
    // 最後一則可能沒有 \n\n 結尾
    buffer += decoder.decode();
    handleBlocks([buffer]);
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/edea60c2f36bf7796cd653068f8315432f2eefe7)

---

<a id="84706f8"></a>

## 教學⑦-6 前端 applyEvent 把事件套用成步驟

- 目標：單一入口 switch 把事件轉成畫面步驟
- 重點：同 stepId 的 delta 接到上一個 step 的 content
- 重點：新 step 開始時先把串流中的文字 step 收尾
- 重點：tool_call 維持 running，tool_result 用 toolCallId 配對收尾
- 重點：error、done 把所有 running 的 step 改成對應狀態
- 重點：default 用 assertNever 保證每種事件都有處理
- 驗收：bun run build 綠燈
- 卡點：用 steps.at(-1) 找 tool_call，multi_tool 亂序會配錯

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/lib/applyEvent.ts`

```ts
    case 'thinking':
    case 'text': {
      const last = steps.at(-1);
      // 同一個 step 的後續 delta 接在後面
      if (last && last.id === ev.data.stepId && (last.type === 'thinking' || last.type === 'text')) {
        last.content += ev.data.delta;
        return;
      }
      finishStreaming(steps, 'done');
      steps.push({ id: ev.data.stepId, type: ev.event, status: 'running', content: ev.data.delta });
      return;
    }
```

```ts
    case 'tool_result': {
      finishStreaming(steps, 'done');
      // 結果順序不一定，要用 toolCallId 配對
      const call = steps.find(s => s.type === 'tool_call' && s.toolCallId === ev.data.toolCallId);
      if (call) call.status = 'done';
      // ...
    }
    // ...
    default:
      assertNever(ev);
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/84706f857c9f0f1a112068015ea5f4986fb45cd1)

---

<a id="5cdb9de"></a>

## 教學⑦-7 前端文字 delta 以 16ms 合批

- 目標：LLM 吐字很快時，每 16ms 才更新一次畫面
- 重點：同一個 step 的 delta 先接進 pending
- 重點：第一個 delta 進來才開始計時，到時 flush 套用
- 重點：換 step 或非文字事件進來前先 flush
- 重點：flush 也開放給外部，結束與取消時用
- 驗收：bun run build 綠燈
- 卡點：非文字事件沒先 flush，tool_call 會跑到文字前面

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/lib/batcher.ts`

```ts
  const push = (ev: AgentEvent): void => {
    if (ev.event === 'thinking' || ev.event === 'text') {
      if (pending && pending.event === ev.event && pending.data.stepId === ev.data.stepId) {
        pending.data.delta += ev.data.delta;
      } else {
        flush();
        // 複製一份，合併時不改到原始事件
        pending = { ...ev, data: { ...ev.data } };
      }
      timer ??= setTimeout(flush, interval);
      return;
    }
    // 非文字事件先 flush，順序才不會亂
    flush();
    apply(ev);
  }
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/5cdb9de1aca72550323347f7357bdd825cf037b3)

---

<a id="9e8ab8a"></a>

## 教學⑦-8 前端 useAgent 管理請求與步驟狀態

- 目標：把串流、合批、套用事件包成一個 composable
- 重點：steps 與 isRunning 給畫面用，ask 清空後送出
- 重點：每次請求一個 AbortController，新請求先 abort 舊的
- 重點：onEvent 檢查 controller !== current，忽略舊請求事件
- 重點：throttle 開啟走 batcher，關閉時直接套用
- 重點：網路或 HTTP 錯誤轉成可重試的 error step
- 驗收：bun run build 綠燈
- 卡點：沒擋舊請求，連按送出兩次的事件會混在一起

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/composables/useAgent.ts`

```ts
  async function run(req: AgentRequest): Promise<void> {
    controller?.abort();
    const current = new AbortController();
    controller = current;
    isRunning.value = true;

    const onEvent = (ev: AgentEvent): void => {
      // 已被新請求取代的舊事件一律忽略
      if (controller !== current) return;
      if (toValue(options.throttle)) {
        batcher.push(ev);
      } else {
        batcher.flush();
        applyEvent(steps.value, ev);
      }
    }
    // ...
```

```ts
    } catch (e) {
      if (controller !== current) return;
      batcher.flush();
      markRunning(steps.value, 'error');
      steps.value.push({
        id: crypto.randomUUID(),
        type: 'error',
        status: 'error',
        message: e instanceof Error ? e.message : String(e),
        retryable: true,
      });
    }
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/9e8ab8a253f643696450b0fb1f9eff54910f53ce)

---

<a id="c5533eb"></a>

## 教學⑦-9 前端 AgentTimeline 元件與畫面組裝

- 目標：依步驟種類顯示 Timeline，可選劇本、速度與合批
- 重點：AgentTimeline 用 v-if 判斷 step.type，template 內型別收窄
- 重點：thinking 進行中展開，tool_call 顯示 spinner，text 閃爍游標
- 重點：onUpdated 發出 rendered，App 累計 re-render 次數
- 重點：App 改用 useAgent，Enter 送出，執行中鎖住選項
- 重點：頁面語系改 zh-Hant，標題改 demo-5
- 驗收：三個 scenario 都能跑完，multi_tool 工具結果配對正確
- 驗收：speed fast 時關閉合批，re-render 次數明顯變多
- 卡點：template 無法寫 assertNever，新 step 型別要靠 applyEvent 提醒

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/components/AgentTimeline.vue`

```vue
      <details v-if="step.type === 'thinking'" class="thinking" :open="step.status === 'running'">
        <summary class="step-label label-thinking">
          {{ step.status === 'running' ? '思考中…' : '思考過程' }}
        </summary>
        <p class="thinking-content">{{ step.content }}</p>
      </details>

      <div v-else-if="step.type === 'tool_call'">
        <div class="step-label label-tool">
          呼叫工具 : {{ step.name }}
          <span v-if="step.status === 'running'" class="spinner" aria-label="執行中"></span>
        </div>
        <pre class="code">{{ toJSON(step.args) }}</pre>
      </div>
      <!-- ... -->
```

`tutorial/02-post-stream/frontend/src/App.vue`

```ts
  const { steps, isRunning, ask } = useAgent({ throttle });

  // methods::
  const handleSubmit = (): void => {
    if (!inputMessage.value.trim() || isRunning.value) return;
    question.value = inputMessage.value;
    renderCount.value = 0;
    ask({
      message: inputMessage.value,
      scenario: scenario.value,
      speed: speed.value,
    });
    inputMessage.value = '';
  }
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/c5533ebbf1b08a15df8e6ecf9d71f00639f0f5bb)

---

<a id="7259b9c"></a>

## 教學⑦-10 前端停止、Ctrl+C 與從指定 step 重試

- 目標：執行中可取消，錯誤或中斷後從指定 step 繼續
- 重點：stop 呼叫 abort，後端 r.Context() 跟著被 cancel
- 重點：abort 走 aborted 狀態與 isAborted，不當成錯誤
- 重點：retryFrom 保留該 step 之前的結果，帶 resumeFrom 重送
- 重點：找不到 step 時用 lastRequest 從頭重跑
- 重點：error 依 retryable 顯示重試按鈕，aborted 顯示從中斷處繼續
- 重點：執行中攔截 Ctrl+C，有選取文字時保留複製
- 驗收：error_retry 按「從 s4 重試」只重跑 s4 並完成
- 驗收：執行中按停止或 Ctrl+C，後端印出使用者取消
- 卡點：AbortError 當成一般錯誤，取消會顯示成發生錯誤

**關鍵程式碼**

`tutorial/02-post-stream/frontend/src/composables/useAgent.ts`

```ts
  // abort 後 fetch 中斷，後端 r.Context() 也會被 cancel
  function stop(): void {
    controller?.abort();
  }

  // 保留指定 step 之前的結果，找不到 step 就從頭重跑
  function retryFrom(stepId?: string): void {
    if (!lastRequest || isRunning.value) return;
    const idx = stepId ? steps.value.findIndex(s => s.id === stepId) : -1;
    if (idx === -1) {
      ask({ ...lastRequest, resumeFrom: undefined });
      return;
    }
    steps.value = steps.value.slice(0, idx).filter(s => s.type !== 'error');
    void run({ ...lastRequest, resumeFrom: stepId });
  }
```

`tutorial/02-post-stream/frontend/src/App.vue`

```ts
  // 執行中才攔截 Ctrl+C，有選取文字時保留複製
  const handleKeydown = (e: KeyboardEvent): void => {
    if (!e.ctrlKey || e.key.toLowerCase() !== 'c') return;
    if (!isRunning.value) return;
    if (window.getSelection()?.toString()) return;
    e.preventDefault();
    stop();
  }
```

[看程式碼變更](https://github.com/AdaaWu/helloworlddev-agentic-client/commit/7259b9c8fe4e42c1bb1d768cd061dc09fe5c3bb0)

