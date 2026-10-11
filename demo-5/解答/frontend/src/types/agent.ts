/**
 * 模組 1 : Agent 執行狀態建模
 *
 * 這裡有兩層型別 :
 *   - AgentEvent : 後端 SSE 傳過來的「事件」（wire format）
 *   - AgentStep  : 畫面上要顯示的「步驟」（UI state）
 *
 * 為什麼要分兩層？
 *   thinking / text 會被切成很多個 delta 一個一個送來，
 *   但畫面上它們要累積成「一個」step，所以事件 ≠ 步驟。
 *
 * 兩層都是 discriminated union :
 *   AgentEvent 用 `event` 當判別欄位、AgentStep 用 `type` 當判別欄位，
 *   在 switch 裡判斷之後，TypeScript 會自動把型別窄化，
 *   例如 case 'tool_call' 裡面才能拿到 name / args，拿錯欄位會直接編譯錯誤。
 */

// ===== 後端事件（SSE 的 event 名稱 = discriminant，data 是 JSON） =====
export type AgentEvent =
  | { event: 'thinking'; data: { stepId: string; delta: string } }
  | { event: 'tool_call'; data: { stepId: string; toolCallId: string; name: string; args: Record<string, unknown> } }
  | { event: 'tool_result'; data: { stepId: string; toolCallId: string; output: unknown } }
  | { event: 'text'; data: { stepId: string; delta: string } }
  | { event: 'error'; data: { stepId: string; message: string; retryable: boolean; resumeFrom?: string } }
  | { event: 'done'; data: Record<string, never> };

export type AgentEventName = AgentEvent['event'];

// 給 parser 做 runtime 檢查用（型別只存在編譯期，runtime 還是要有一份清單）
export const agentEventNames = [
  'thinking',
  'tool_call',
  'tool_result',
  'text',
  'error',
  'done',
] as const satisfies readonly AgentEventName[];

// ===== 畫面上的步驟 =====
// running : 進行中（文字還在吐、工具還在跑）
// aborted : 使用者按了停止 / Ctrl+C
export type StepStatus = 'running' | 'done' | 'error' | 'aborted';

interface BaseStep {
  id: string
  status: StepStatus
}

export type AgentStep =
  | (BaseStep & { type: 'thinking'; content: string })
  | (BaseStep & { type: 'tool_call'; toolCallId: string; name: string; args: Record<string, unknown> })
  | (BaseStep & { type: 'tool_result'; toolCallId: string; output: unknown })
  | (BaseStep & { type: 'text'; content: string })
  | (BaseStep & { type: 'error'; message: string; retryable: boolean; resumeFrom?: string });

// ===== 送給後端的 request =====
export const scenarioList = ['search_and_calc', 'multi_tool', 'error_retry'] as const;
export type Scenario = typeof scenarioList[number];
export type Speed = 'normal' | 'fast';

export interface AgentRequest {
  message: string
  scenario: Scenario
  speed: Speed
  // 有值代表「從這個 step 開始重跑」
  resumeFrom?: string
}

/**
 * 窮舉檢查 : 放在 switch 的 default。
 * 如果 union 新增了一個成員卻忘了寫 case，傳進來的值就不是 never，編譯器會報錯。
 * （可以試試在 AgentEvent 加一個 { event: 'citation'; data: {} } 看看會發生什麼事）
 */
export function assertNever(x: never): never {
  throw new Error(`未處理的型別 : ${JSON.stringify(x)}`);
}
