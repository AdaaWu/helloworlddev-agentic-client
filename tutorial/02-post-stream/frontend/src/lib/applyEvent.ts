import { assertNever, type AgentEvent, type AgentStep, type StepStatus } from '../types/agent';

// 把一個事件套用到步驟列表
export function applyEvent(steps: AgentStep[], ev: AgentEvent): void {
  switch (ev.event) {
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
    case 'tool_call':
      finishStreaming(steps, 'done');
      // 維持 running 直到對應的 tool_result 回來
      steps.push({
        id: ev.data.stepId,
        type: 'tool_call',
        status: 'running',
        toolCallId: ev.data.toolCallId,
        name: ev.data.name,
        args: ev.data.args,
      });
      return;
    case 'tool_result': {
      finishStreaming(steps, 'done');
      // 結果順序不一定，要用 toolCallId 配對
      const call = steps.find(s => s.type === 'tool_call' && s.toolCallId === ev.data.toolCallId);
      if (call) call.status = 'done';
      steps.push({
        id: ev.data.stepId,
        type: 'tool_result',
        status: 'done',
        toolCallId: ev.data.toolCallId,
        output: ev.data.output,
      });
      return;
    }
    case 'error':
      markRunning(steps, 'error');
      steps.push({
        id: ev.data.stepId,
        type: 'error',
        status: 'error',
        message: ev.data.message,
        retryable: ev.data.retryable,
        resumeFrom: ev.data.resumeFrom,
      });
      return;
    case 'done':
      markRunning(steps, 'done');
      return;
    default:
      assertNever(ev);
  }
}

// 只收尾文字類 step，tool_call 要等 tool_result
function finishStreaming(steps: AgentStep[], status: StepStatus): void {
  for (const step of steps) {
    if (step.status === 'running' && (step.type === 'thinking' || step.type === 'text')) {
      step.status = status;
    }
  }
}

export function markRunning(steps: AgentStep[], status: StepStatus): void {
  for (const step of steps) {
    if (step.status === 'running') step.status = status;
  }
}
