<script setup lang="ts">
  /**
   * 模組 3 : 即時回饋 UI 元件
   *
   * 每種 step type 對應一個獨立的 template 分支。
   * v-if / v-else-if 用 step.type 判斷後，vue-tsc（Volar）一樣會窄化型別，
   * 所以在 tool_call 分支裡才能用 step.name、step.args，在 text 分支裡才能用 step.content。
   *
   * 注意 : template 裡沒辦法寫 assertNever，窮舉檢查交給 applyEvent 那層負責。
   */
  import { computed, onUpdated } from 'vue';
  import type { AgentStep } from '../types/agent';

  const props = defineProps<{
    steps: AgentStep[]
  }>();

  const emit = defineEmits<{
    // 從指定 step 重跑
    retry: [fromStepId: string | undefined]
    // 每次 re-render 都通知外層，用來展示 throttle 的效果
    rendered: []
  }>();

  // 被取消時，第一個 aborted 的 step 就是「從中斷處繼續」的起點
  const firstAbortedStep = computed(() => props.steps.find(step => step.status === 'aborted'));

  const toJSON = (value: unknown): string => JSON.stringify(value, null, 2);

  onUpdated(() => emit('rendered'));
</script>

<template>
  <ol class="timeline">
    <li
      v-for="step in steps"
      :key="step.id"
      class="step"
      :class="[`step-${step.type}`, `is-${step.status}`]"
    >
      <!-- thinking : 進行中自動展開，結束後收合 -->
      <details v-if="step.type === 'thinking'" class="thinking" :open="step.status === 'running'">
        <summary class="step-label label-thinking">
          {{ step.status === 'running' ? '思考中…' : '思考過程' }}
        </summary>
        <p class="thinking-content">{{ step.content }}</p>
      </details>

      <!-- tool_call : 工具名稱 + 參數，執行中顯示 spinner -->
      <div v-else-if="step.type === 'tool_call'">
        <div class="step-label label-tool">
          呼叫工具 : {{ step.name }}
          <span v-if="step.status === 'running'" class="spinner" aria-label="執行中"></span>
        </div>
        <pre class="code">{{ toJSON(step.args) }}</pre>
      </div>

      <!-- tool_result : 預設收合，需要時再展開 -->
      <details v-else-if="step.type === 'tool_result'">
        <summary class="step-label label-tool">工具結果 ({{ step.toolCallId }})</summary>
        <pre class="code">{{ toJSON(step.output) }}</pre>
      </details>

      <!-- text : AI 的回答，進行中顯示閃爍游標 -->
      <div v-else-if="step.type === 'text'">
        <div class="step-label label-ai">AI 的回應</div>
        <p class="text-content">{{ step.content }}<span v-if="step.status === 'running'" class="cursor" aria-hidden="true"></span></p>
      </div>

      <!-- error : 依 retryable 決定能不能重試 -->
      <div v-else-if="step.type === 'error'" role="alert">
        <div class="step-label label-error">發生錯誤</div>
        <p>{{ step.message }}</p>
        <button
          v-if="step.retryable"
          class="retry-button"
          @click="emit('retry', step.resumeFrom)"
        >
          {{ step.resumeFrom ? `從 ${step.resumeFrom} 重試` : '重新開始' }}
        </button>
        <p v-else class="hint">這個錯誤無法重試</p>
      </div>

      <p v-if="step.status === 'aborted'" class="hint">已中止</p>
    </li>
  </ol>

  <div v-if="firstAbortedStep" class="resume">
    <button class="retry-button" @click="emit('retry', firstAbortedStep.id)">
      從中斷處（{{ firstAbortedStep.id }}）繼續
    </button>
  </div>
</template>

<style scoped>
.timeline {
  display: flex;
  flex-direction: column;
  row-gap: 1rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.step {
  padding: 1rem 1.25rem;
  box-sizing: border-box;
  background-color: #FFFFFF;
  border: 3px solid #2E2E2E;
  border-radius: 1.25rem;
  line-height: 1.7;
  letter-spacing: 0.03em;
}

.step-text {
  background-color: #DFEBD6;
}

.step-error {
  background-color: #FBE3DF;
}

.is-aborted {
  opacity: 0.6;
  border-style: dashed;
}

.step-label {
  display: inline-flex;
  align-items: center;
  column-gap: 0.5rem;
  width: fit-content;
  margin-bottom: 0.6rem;
  padding: 0.15rem 0.9rem;
  border: 2.5px solid #2E2E2E;
  border-radius: 999px;
  font-size: 0.9rem;
  letter-spacing: 0.08em;
}

summary.step-label {
  cursor: pointer;
}

.label-thinking {
  background-color: #F7F5EC;
}

.label-tool {
  background-color: #A9CBE0;
}

.label-ai {
  background-color: #B5D3A0;
}

.label-error {
  background-color: #E8968A;
}

.thinking-content {
  margin: 0;
  color: #6B6B6B;
  font-style: italic;
  white-space: pre-wrap;
}

.text-content {
  margin: 0;
  white-space: pre-wrap;
}

.code {
  margin: 0;
  padding: 0.75rem 1rem;
  background-color: #F7F5EC;
  border: 2px solid #2E2E2E;
  border-radius: 0.75rem;
  font-size: 0.9rem;
  overflow-x: auto;
}

.hint {
  margin: 0.5rem 0 0;
  color: #6B6B6B;
  font-size: 0.9rem;
}

/* 閃爍游標 : steps(1) 讓它「跳」著閃，而不是淡入淡出 */
.cursor {
  display: inline-block;
  width: 0.5em;
  height: 1.1em;
  margin-left: 2px;
  vertical-align: text-bottom;
  background-color: #2E2E2E;
  animation: blink 1s steps(1) infinite;
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}

.spinner {
  display: inline-block;
  width: 0.85em;
  height: 0.85em;
  border: 2px solid #2E2E2E;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.resume {
  display: flex;
  justify-content: center;
  margin-top: 1rem;
}

.retry-button {
  padding: 0.4rem 1.5rem;
  outline: none;
  border: 3px solid #2E2E2E;
  border-radius: 999px;
  background-color: #E8968A;
  color: #2E2E2E;
  font-size: 1rem;
  letter-spacing: 0.08em;
  cursor: pointer;
  transition: transform 0.1s ease, background-color 0.15s ease;
}

.retry-button:hover {
  background-color: #E07F71;
}

.retry-button:active {
  transform: translateY(2px);
}
</style>
