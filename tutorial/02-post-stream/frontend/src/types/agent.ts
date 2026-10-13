// 後端事件：event 是判別欄位，data 是 JSON
export type AgentEvent =
  | { event: 'thinking'; data: { stepId: string; delta: string } }
  | { event: 'tool_call'; data: { stepId: string; toolCallId: string; name: string; args: Record<string, unknown> } }
  | { event: 'tool_result'; data: { stepId: string; toolCallId: string; output: unknown } }
  | { event: 'text'; data: { stepId: string; delta: string } }
  | { event: 'error'; data: { stepId: string; message: string; retryable: boolean; resumeFrom?: string } }
  | { event: 'done'; data: Record<string, never> };

export type AgentEventName = AgentEvent['event'];

// 型別只存在編譯期，runtime 檢查要另一份清單
export const agentEventNames = [
  'thinking',
  'tool_call',
  'tool_result',
  'text',
  'error',
  'done',
] as const satisfies readonly AgentEventName[];

export type StepStatus = 'running' | 'done' | 'error' | 'aborted';

interface BaseStep {
  id: string
  status: StepStatus
}

// 畫面步驟：多個 delta 事件累積成一個 step
export type AgentStep =
  | (BaseStep & { type: 'thinking'; content: string })
  | (BaseStep & { type: 'tool_call'; toolCallId: string; name: string; args: Record<string, unknown> })
  | (BaseStep & { type: 'tool_result'; toolCallId: string; output: unknown })
  | (BaseStep & { type: 'text'; content: string })
  | (BaseStep & { type: 'error'; message: string; retryable: boolean; resumeFrom?: string });

export const scenarioList = ['search_and_calc', 'multi_tool', 'error_retry'] as const;
export type Scenario = typeof scenarioList[number];
export type Speed = 'normal' | 'fast';

export interface AgentRequest {
  message: string
  scenario: Scenario
  speed: Speed
  resumeFrom?: string
}

// 放在 switch default，漏寫 case 會編譯錯誤
export function assertNever(x: never): never {
  throw new Error(`未處理的型別 : ${JSON.stringify(x)}`);
}
