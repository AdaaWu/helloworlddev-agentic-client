<script setup lang="ts">
  import { onUpdated } from 'vue';
  import type { AgentStep } from '../types/agent';

  defineProps<{
    steps: AgentStep[]
  }>();

  const emit = defineEmits<{
    // 每次 re-render 通知外層，用來比較 throttle 效果
    rendered: []
  }>();

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
      <details v-if="step.type === 'thinking'" class="thinking" :open="step.status === 'running'">
        <summary class="step-label label-thinking">
          {{ step.status === 'running' ? '思考中…' : '思考過程' }}
        </summary>
        <p class="thinking-content">{{ step.content }}</p>
      </details>

      <div v-else-if="step.type === 'tool_call'">
        <div class="step-label label-tool">
          呼叫工具 : {{ step.name }}
          <span v-if="step.status === 'running'" class="spinner" aria-label="執行中"></span>
        </div>
        <pre class="code">{{ toJSON(step.args) }}</pre>
      </div>

      <details v-else-if="step.type === 'tool_result'">
        <summary class="step-label label-tool">工具結果 ({{ step.toolCallId }})</summary>
        <pre class="code">{{ toJSON(step.output) }}</pre>
      </details>

      <div v-else-if="step.type === 'text'">
        <div class="step-label label-ai">AI 的回應</div>
        <p class="text-content">{{ step.content }}<span v-if="step.status === 'running'" class="cursor" aria-hidden="true"></span></p>
      </div>

      <div v-else-if="step.type === 'error'" role="alert">
        <div class="step-label label-error">發生錯誤</div>
        <p>{{ step.message }}</p>
      </div>
    </li>
  </ol>
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


/* steps(1) 讓游標跳著閃 */
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




</style>
