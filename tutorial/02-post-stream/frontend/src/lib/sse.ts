import { agentEventNames, type AgentEvent, type AgentEventName, type AgentRequest } from '../types/agent';

const API_BASE = 'http://localhost:8081';

const isAgentEventName = (input: string): input is AgentEventName => {
  return agentEventNames.some(name => name === input);
}

// 解析一塊已用 \n\n 切好的 SSE 訊息
export function parseBlock(block: string): AgentEvent | null {
  // SSE 規格：沒寫 event 時預設 message
  let event = 'message';
  const dataList: string[] = [];

  for (const line of block.split('\n')) {
    // : 開頭是註解行，常當 heartbeat
    if (line === '' || line.startsWith(':')) continue;

    const idx = line.indexOf(':');
    const key = idx === -1 ? line : line.slice(0, idx);
    const rawValue = idx === -1 ? '' : line.slice(idx + 1);
    const value = rawValue.startsWith(' ') ? rawValue.slice(1) : rawValue;

    if (key === 'event') event = value;
    else if (key === 'data') dataList.push(value);
    // 其他欄位依規格略過
  }

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
}

interface StreamOptions {
  // abort 時 fetch 與 reader.read() 都會丟出 AbortError
  signal: AbortSignal
  onEvent: (ev: AgentEvent) => void
}

export async function streamAgent(req: AgentRequest, { signal, onEvent }: StreamOptions): Promise<void> {
  const response = await fetch(`${API_BASE}/sse`, {
    method: 'POST',
    headers: {
      'Accept': 'text/event-stream',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(req),
    signal,
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
      // 其他 server 可能送 \r\n，先統一成 \n
      buffer += decoder.decode(value, { stream: true }).replace(/\r\n?/g, '\n');
      const blockList = buffer.split('\n\n');
      buffer = blockList.pop() ?? '';
      handleBlocks(blockList);
    }
    // 最後一則可能沒有 \n\n 結尾
    buffer += decoder.decode();
    handleBlocks([buffer]);
  } finally {
    reader.releaseLock();
  }
}
