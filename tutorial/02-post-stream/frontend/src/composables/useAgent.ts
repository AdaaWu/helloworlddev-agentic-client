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

  // 不需要響應式，用一般變數
  let controller: AbortController | null = null;

  const batcher = createDeltaBatcher(ev => applyEvent(steps.value, ev));

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

    try {
      await streamAgent(req, { signal: current.signal, onEvent });
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

  return { steps, isRunning, ask };
}
