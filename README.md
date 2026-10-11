# 為 Agentic AI 設計前端狀態機與即時回饋介面 (Hello world Dev workshop)

(本 workshop 為文藝復興活動，需要手動撰寫程式碼，以達到學習目的)
註：文藝復興活動 -> 不靠 AI 補全，從手寫開始

## demo-0 : 小試身手
> objective : 製作一個 hello world dev 的 web service

## demo-1 : 製作 sse 的服務並且與前端接起來
> objective : 

1. Go 的部分(後端) : 加入 get SSE 的服務 提供 RFC3339 的當下時間每一秒回傳
2. Vue 的部分(前端) : 製作 `EventSource` 接起後端的 message

## demo-2 : ：改用 fetch 讀取 ReadableStream（後端改為 POST SSE）

> objective :  

Vue 的部分(前端) : EventSource -> Fetch 並且 console 出 後端的 stream 結果

## demo3/4 : 解析 SSE 文字格式、依 event 分派 callback

### demo-3 : 處理前端收到的訊息

> objective :   

Vue 的部分(前端) : 整理改為 fetch 的資料處理問題

### demo-4 : 處理後端收到的訊息

> objective :   

Golang 的部分(後端) : 後端改用 `AgentStep` 的方式把訊息傳出去

## demo-5 : 完整的 Agent Timeline（展示用，只有解答）

> objective : 把前面的內容組合成一個完整的 Agent 前端

1. Golang 的部分(後端) : mock LLM 三種 scenario（`search_and_calc`、`multi_tool`、`error_retry`），事件改成 `thinking / tool_call / tool_result / text / error / done`，data 改成 JSON，支援 `resumeFrom` 從指定 step 重跑
2. Vue 的部分(前端) :
    - `AgentEvent` / `AgentStep` discriminated union + `assertNever` 窮舉檢查
    - 符合 SSE 規格的 parser（註解行、未知欄位、`\r\n`、結尾剩餘 buffer）
    - text token 以 16ms throttle 合批更新（畫面上可以切換開關、比較 re-render 次數）
    - `AgentTimeline` 元件 : 每種 step 一個 template 分支、閃爍游標、工具 spinner、錯誤狀態
    - `AbortController` : 「停止」按鈕或 `Ctrl+C` 取消問答，依 `retryable` 決定能否重試，並從指定 step 恢復

```sh
curl -N http://localhost:8080/sse -X POST -d '{"message":"Hi","scenario":"error_retry"}'
curl -N http://localhost:8080/sse -X POST -d '{"message":"Hi","scenario":"error_retry","resumeFrom":"s4"}'
```

## 常見指令


> 前端部分

```sh
bunx create-vite@latest --template vue-ts <專案名稱>
# 這會把 vue 的框架弄出來，我們會在 src/App.vue 上面實作
```
這代表啟動前端的專案

> 後端的部分: 這會初始化後端 golang 專案

```sh
go mod init <專案名稱>
touch main.go # 創建一個 main.go 的檔案，我們會在 main.go 上面實作
```

測試後端的部分

(demo-3 之後)
```sh
curl -N http://localhost:8080/sse -X POST -d '{"message": "Hi" }'
```
