import { agentEventNames, type AgentEvent, type AgentEventName, type AgentRequest } from '../types/agent';

/**
 * 模組 2 : 混合型別 SSE stream 解析
 *
 * 跟 demo-3 / demo-4 的 parser 相比，修正了幾個 SSE 規格上的細節 :
 *   1. `:` 開頭的行是註解（常被拿來當 heartbeat），要略過
 *   2. 不認識的欄位要略過（demo-4 是 fallback 成 data）
 *   3. 換行可能是 \r\n，先統一成 \n 再切
 *   4. stream 結束時 buffer 裡可能還有最後一則，要處理掉
 *   5. data 是 JSON，解析失敗時轉成 error 事件，而不是讓整個 stream 爆掉
 */

const isAgentEventName = (input: string): input is AgentEventName => {
  return agentEventNames.some(name => name === input);
}

// 解析一則 SSE 訊息（已經用 \n\n 切好的一塊）
export function parseBlock(block: string): AgentEvent | null {
  let event = 'message'; // SSE 規格 : 沒寫 event 時預設是 message
  const dataList: string[] = [];

  for (const line of block.split('\n')) {
    // (1) 空行、註解行略過
    if (line === '' || line.startsWith(':')) continue;

    const idx = line.indexOf(':');
    const key = idx === -1 ? line : line.slice(0, idx);
    const rawValue = idx === -1 ? '' : line.slice(idx + 1);
    const value = rawValue.startsWith(' ') ? rawValue.slice(1) : rawValue;

    if (key === 'event') event = value;
    else if (key === 'data') dataList.push(value);
    // (2) id / retry 這個 demo 用不到；其他不認識的欄位依規格略過
  }

  // 不是我們認識的事件（例如 message）就不處理
  if (!isAgentEventName(event)) return null;

  try {
    const data: unknown = JSON.parse(dataList.join('\n') || '{}');
    // 這裡用 as 信任後端的格式。正式產品可以為每種事件寫 type guard，或用 zod / valibot 驗證
    return { event, data } as AgentEvent;
  } catch {
    // (5) JSON 壞掉 : 轉成一個不可重試的 error 事件
    return {
      event: 'error',
      data: { stepId: crypto.randomUUID(), message: `無法解析的事件 : ${event}`, retryable: false },
    };
  }
}

interface StreamOptions {
  // 模組 4 : 由 AbortController 提供，abort 時 fetch 和 reader.read() 都會丟出 AbortError
  signal: AbortSignal
  onEvent: (ev: AgentEvent) => void
}

/**
 * 用 fetch + ReadableStream 讀取 POST SSE（跟 demo-4 的 startEvent 一樣的骨架）
 * 差別是 : 多了 signal、改用 parseBlock、結束時處理剩餘 buffer、錯誤用 throw 往上丟
 */
export async function streamAgent(req: AgentRequest, { signal, onEvent }: StreamOptions): Promise<void> {
  const response = await fetch('http://localhost:8080/sse', {
    method: 'POST',
    headers: {
      'Accept': 'text/event-stream',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(req),
    signal, // 重點 : 把 signal 交給 fetch
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status} : ${await response.text()}`);
  }
  const body = response.body;
  if (!body) {
    throw new Error('response body 是 null');
  }

  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  const handleBlocks = (blockList: string[]) => {
    for (const block of blockList) {
      if (!block.trim()) continue;
      const ev = parseBlock(block);
      if (ev) onEvent(ev);
    }
  }

  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      // (3) 統一換行符號（這個後端只送 \n，這行是給真實世界的 server 用的）
      buffer += decoder.decode(value, { stream: true }).replace(/\r\n?/g, '\n');
      const blockList = buffer.split('\n\n');
      buffer = blockList.pop() ?? '';
      handleBlocks(blockList);
    }
    // (4) stream 結束 : 把 decoder 裡剩下的位元組和 buffer 最後一塊處理掉
    buffer += decoder.decode();
    handleBlocks([buffer]);
  } finally {
    reader.releaseLock();
  }
}
