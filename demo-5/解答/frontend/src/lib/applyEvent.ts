import { assertNever, type AgentEvent, type AgentStep, type StepStatus } from '../types/agent';

/**
 * 把「一個事件」套用到「步驟列表」上（類似 reducer）。
 *
 * demo-4 是每種事件各寫一個 callback（onMessage / onToolCalling ...），
 * 這裡改成單一入口 + switch，好處是 :
 *   1. 每個 case 裡 ev.data 的型別都被窄化好了
 *   2. default 的 assertNever 保證每種事件都有被處理
 */
export function applyEvent(steps: AgentStep[], ev: AgentEvent): void {
  switch (ev.event) {
    case 'thinking':
    case 'text': {
      // 在這裡 ev.data 被窄化成 { stepId: string; delta: string }
      const last = steps.at(-1);
      // 同一個 step 的後續 delta : 接在後面
      if (last && last.id === ev.data.stepId && (last.type === 'thinking' || last.type === 'text')) {
        last.content += ev.data.delta;
        return;
      }
      // 新的 step 開始 : 先把前一個串流中的 step 收尾
      finishStreaming(steps, 'done');
      steps.push({ id: ev.data.stepId, type: ev.event, status: 'running', content: ev.data.delta });
      return;
    }
    case 'tool_call':
      finishStreaming(steps, 'done');
      // tool_call 會維持 running（顯示 spinner），直到對應的 tool_result 回來
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
      // 多個工具同時呼叫時（multi_tool），結果的順序不一定，要用 toolCallId 配對
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
      // 所有還在進行中的 step 都視為被錯誤中斷
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
      // 新增 AgentEvent 成員卻忘了處理 -> 這行會編譯錯誤
      assertNever(ev);
  }
}

// 把「還在串流的文字類 step」收尾（tool_call 不動，它要等 tool_result）
function finishStreaming(steps: AgentStep[], status: StepStatus): void {
  for (const step of steps) {
    if (step.status === 'running' && (step.type === 'thinking' || step.type === 'text')) {
      step.status = status;
    }
  }
}

// 把所有 running 的 step 改成指定狀態（結束、錯誤、取消時使用）
export function markRunning(steps: AgentStep[], status: StepStatus): void {
  for (const step of steps) {
    if (step.status === 'running') step.status = status;
  }
}
