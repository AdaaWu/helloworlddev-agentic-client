import { ref, shallowRef, toValue, type MaybeRefOrGetter } from 'vue';
import type { AgentEvent, AgentRequest, AgentStep } from '../types/agent';
import { applyEvent, markRunning } from '../lib/applyEvent';
import { createDeltaBatcher } from '../lib/batcher';
import { streamAgent } from '../lib/sse';

interface UseAgentOptions {
  // 是否啟用 16ms 合批（畫面上可以切換，比較 re-render 次數）
  throttle: MaybeRefOrGetter<boolean>
}

/**
 * 模組 4 : Abort + Retry
 *
 * 把「跟 Agent 對話」的狀態和動作包成一個 composable :
 *   - steps     : 畫面上的步驟
 *   - isRunning : 是否執行中（決定顯示「送出」還是「停止」）
 *   - ask       : 問一個新問題
 *   - stop      : 用 AbortController 取消（停止按鈕 / Ctrl+C）
 *   - retryFrom : 從指定 step 恢復執行，保留之前完成的 step，而不是重頭來過
 */
export function useAgent(options: UseAgentOptions) {
  const steps = ref<AgentStep[]>([]);
  const isRunning = shallowRef(false);
  const isAborted = shallowRef(false);

  // AbortController 不需要是響應式的，用一般變數就好
  let controller: AbortController | null = null;
  // 記住上一次的 request，retry 時要用同樣的 message / scenario
  let lastRequest: AgentRequest | null = null;

  const batcher = createDeltaBatcher(ev => applyEvent(steps.value, ev));

  async function run(req: AgentRequest): Promise<void> {
    // 如果上一個請求還在跑，先取消它
    controller?.abort();
    const current = new AbortController();
    controller = current;
    lastRequest = req;
    isRunning.value = true;
    isAborted.value = false;

    const onEvent = (ev: AgentEvent): void => {
      // 舊的請求（已經被新的取代）送來的事件一律忽略
      if (controller !== current) return;
      if (toValue(options.throttle)) {
        batcher.push(ev);
      } else {
        batcher.flush();
        applyEvent(steps.value, ev);
      }
    }

    try {
      await streamAgent(req, { signal: current.signal, onEvent });
    } catch (e) {
      if (controller !== current) return;
      batcher.flush();
      // 被 abort 時 fetch / reader.read() 會丟出 AbortError，這不是「錯誤」，是使用者的選擇
      if (current.signal.aborted) {
        markRunning(steps.value, 'aborted');
        isAborted.value = true;
        return;
      }
      // 真正的錯誤（後端沒開、網路斷線 ...）: 顯示成 error step，允許從頭重試
      markRunning(steps.value, 'error');
      steps.value.push({
        id: crypto.randomUUID(),
        type: 'error',
        status: 'error',
        message: e instanceof Error ? e.message : String(e),
        retryable: true,
      });
    } finally {
      if (controller === current) {
        batcher.flush();
        controller = null;
        isRunning.value = false;
      }
    }
  }

  function ask(req: AgentRequest): void {
    steps.value = [];
    run(req);
  }

  // 取消 : 只要呼叫 abort()，fetch 會中斷，後端的 r.Context() 也會跟著被 cancel
  function stop(): void {
    controller?.abort();
  }

  /**
   * 從指定 step 恢復 :
   *   1. 把這個 step（含）之後的步驟都移除，之前完成的保留
   *   2. 帶著 resumeFrom 重新送出，後端會跳過前面的 step
   * stepId 找不到時（例如網路錯誤，後端根本沒開始）就從頭重跑
   */
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

  return { steps, isRunning, isAborted, ask, stop, retryFrom };
}
