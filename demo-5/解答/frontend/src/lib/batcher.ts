import type { AgentEvent } from '../types/agent';

type DeltaEvent = Extract<AgentEvent, { event: 'thinking' | 'text' }>;

/**
 * 模組 2 : text token 以 16ms throttle 合批更新
 *
 * 問題 : 後端每個字都是一個事件，每個事件都改一次響應式資料，畫面就要跟著更新。
 *        （Vue 會把同一個 tick 內的修改合併成一次 render，所以實際 re-render 次數 ≈ 收到的 chunk 數，
 *          LLM 吐字很快時還是遠超過螢幕的 60fps）
 * 作法 : thinking / text 的 delta 先暫存，每 16ms（約一個 frame）才真正套用一次。
 *
 * 注意 : 其他事件（tool_call、error ...）進來前要先 flush，
 *        不然 tool_call 會比前面還沒套用的文字先出現在畫面上，順序就亂了。
 */
export function createDeltaBatcher(apply: (ev: AgentEvent) => void, interval = 16) {
  let pending: DeltaEvent | null = null;
  let timer: ReturnType<typeof setTimeout> | null = null;

  const flush = (): void => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    if (pending) {
      const ev = pending;
      pending = null;
      apply(ev);
    }
  }

  const push = (ev: AgentEvent): void => {
    if (ev.event === 'thinking' || ev.event === 'text') {
      if (pending && pending.event === ev.event && pending.data.stepId === ev.data.stepId) {
        // 同一個 step 的 delta : 先接起來
        pending.data.delta += ev.data.delta;
      } else {
        // 換了 step : 先把上一批套用掉，再開新的一批（複製一份，避免改到原始事件）
        flush();
        pending = { ...ev, data: { ...ev.data } };
      }
      // 第一個 delta 進來時開始計時，16ms 後一次套用
      timer ??= setTimeout(flush, interval);
      return;
    }
    // 非文字事件 : 先 flush 保持順序，再直接套用
    flush();
    apply(ev);
  }

  // flush 也要給外面用 : 結束、取消、出錯時都要把最後一批套用掉
  return { push, flush };
}
