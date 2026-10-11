# demo-5 前端（展示用）

## 啟動

```sh
# 先啟動後端
cd ../backend && go run .

# 再啟動前端
bun install
bun run dev
```

打開 http://localhost:5173/ ，選擇 scenario 後送出問題。

## 檔案說明

| 檔案 | 對應模組 | 內容 |
| --- | --- | --- |
| `src/types/agent.ts` | 1 | `AgentEvent` / `AgentStep` discriminated union、`assertNever` |
| `src/lib/applyEvent.ts` | 1 | 把事件套用到步驟列表的 reducer（switch + 窮舉檢查） |
| `src/lib/sse.ts` | 2 | fetch + ReadableStream、SSE parser |
| `src/lib/batcher.ts` | 2 | text / thinking delta 的 16ms 合批 |
| `src/components/AgentTimeline.vue` | 3 | 每種 step 一個 template 分支、游標、spinner、錯誤狀態 |
| `src/composables/useAgent.ts` | 4 | `AbortController` 取消、`retryFrom` 從指定 step 恢復 |
| `src/App.vue` | - | 組裝畫面、`Ctrl+C` 取消 |

## 展示重點

- `search_and_calc` : 基本流程，工具呼叫中會顯示 spinner
- `multi_tool` : 三個工具同時執行，結果用 `toolCallId` 配對
- `error_retry` : 回答寫到一半失敗，按「從 s4 重試」只重跑 s4，前面的步驟保留
- 執行中按 `Ctrl+C` 或「停止」取消（有選取文字時 `Ctrl+C` 維持複製）；在文字串流中取消會出現「從中斷處繼續」
- 速度選 `fast`，切換「16ms 合批」比較 Timeline re-render 次數
- 在 `AgentEvent` 加一個新成員，`applyEvent` 的 `assertNever` 會出現編譯錯誤
