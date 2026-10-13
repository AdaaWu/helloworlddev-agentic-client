import type { AgentEvent } from '../types/agent';

type DeltaEvent = Extract<AgentEvent, { event: 'thinking' | 'text' }>;

// 文字 delta 先暫存，每 16ms 才套用一次
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

  return { push, flush };
}
