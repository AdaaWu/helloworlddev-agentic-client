import { ref, shallowRef, toValue, type MaybeRefOrGetter } from 'vue';
import type { AgentEvent, AgentRequest, AgentStep } from '../types/agent';
import { applyEvent, markRunning } from '../lib/applyEvent';
import { createDeltaBatcher } from '../lib/batcher';
import { streamAgent } from '../lib/sse';

interface UseAgentOptions {
  throttle: MaybeRefOrGetter<boolean>
}

export function useAgent(options: UseAgentOptions) {
  const steps = ref<AgentStep[]>([]);
  const isRunning = shallowRef(false);
  const isAborted = shallowRef(false);

  // 不需要響應式，用一般變數
  let controller: AbortController | null = null;
  // retry 沿用同樣的 message 與 scenario
  let lastRequest: AgentRequest | null = null;

  const batcher = createDeltaBatcher(ev => applyEvent(steps.value, ev));

  async function run(req: AgentRequest): Promise<void> {
    controller?.abort();
    const current = new AbortController();
    controller = current;
    lastRequest = req;
    isRunning.value = true;
    isAborted.value = false;

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

    try {
      await streamAgent(req, { signal: current.signal, onEvent });
    } catch (e) {
      if (controller !== current) return;
      batcher.flush();
      // abort 是使用者的選擇，不當成錯誤
      if (current.signal.aborted) {
        markRunning(steps.value, 'aborted');
        isAborted.value = true;
        return;
      }
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

  return { steps, isRunning, isAborted, ask, stop, retryFrom };
}
