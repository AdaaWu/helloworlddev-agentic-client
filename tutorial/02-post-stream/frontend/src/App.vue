<script setup lang="ts">
  import { onBeforeUnmount, onMounted, shallowRef } from 'vue';
  import AgentTimeline from './components/AgentTimeline.vue';
  import { useAgent } from './composables/useAgent';
  import { scenarioList, type Scenario, type Speed } from './types/agent';

  // state::
  const inputMessage = shallowRef('');
  const question = shallowRef('');
  const scenario = shallowRef<Scenario>('search_and_calc');
  const speed = shallowRef<Speed>('normal');
  const throttle = shallowRef(true);
  const renderCount = shallowRef(0);

  const { steps, isRunning, isAborted, ask, stop, retryFrom } = useAgent({ throttle });

  // methods::
  const handleSubmit = (): void => {
    if (!inputMessage.value.trim() || isRunning.value) return;
    question.value = inputMessage.value;
    renderCount.value = 0;
    ask({
      message: inputMessage.value,
      scenario: scenario.value,
      speed: speed.value,
    });
    inputMessage.value = '';
  }

  const handleRetry = (fromStepId: string | undefined): void => {
    renderCount.value = 0;
    retryFrom(fromStepId);
  }

  // 執行中才攔截 Ctrl+C，有選取文字時保留複製
  const handleKeydown = (e: KeyboardEvent): void => {
    if (!e.ctrlKey || e.key.toLowerCase() !== 'c') return;
    if (!isRunning.value) return;
    if (window.getSelection()?.toString()) return;
    e.preventDefault();
    stop();
  }

  onMounted(() => window.addEventListener('keydown', handleKeydown));
  onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown));
</script>

<template>
  <h1>demo-5 測試畫面</h1>
  <div class="input-block">
    <div class="options">
      <label>
        scenario
        <select v-model="scenario" class="select" :disabled="isRunning">
          <option v-for="item in scenarioList" :key="item" :value="item">{{ item }}</option>
        </select>
      </label>
      <label>
        速度
        <select v-model="speed" class="select" :disabled="isRunning">
          <option value="normal">normal（40ms / 字）</option>
          <option value="fast">fast（2ms / 字）</option>
        </select>
      </label>
      <label class="checkbox">
        <input v-model="throttle" type="checkbox" :disabled="isRunning" />
        16ms 合批
      </label>
    </div>
    <textarea
      v-model="inputMessage"
      class="input-text"
      placeholder="輸入問題，例如：北北基總共有多少人？"
      @keydown.enter.exact.prevent="handleSubmit"
    ></textarea>
    <div class="actions">
      <span class="hint">{{ isRunning ? '執行中，按 Ctrl+C 或「停止」可以取消' : 'Enter 送出' }}</span>
      <button v-if="!isRunning" class="input-button" @click="handleSubmit">送出</button>
      <button v-else class="input-button stop-button" @click="stop">停止</button>
    </div>
  </div>

  <div v-show="question" class="chat">
    <div class="chat-block">
      <div class="chat-role chat-role-user">你的回應</div>
      {{ question }}
    </div>
    <div class="chat-agent">
      <div class="status">
        <span v-if="isRunning" class="status-tag">執行中</span>
        <span v-else-if="isAborted" class="status-tag status-aborted">已取消</span>
        <span class="render-count" data-testid="render-count">Timeline re-render : {{ renderCount }} 次</span>
      </div>
      <AgentTimeline :steps="steps" @retry="handleRetry" @rendered="renderCount++" />
    </div>
  </div>
</template>

<style scoped>
:global(body) {
  background-color: #F7F5EC;
  color: #2E2E2E;
}

.input-block {
  display: flex;
  flex-direction: column;
  row-gap: 1.25rem;
  width: 50vw;
  padding: 1.5rem;
  box-sizing: border-box;
  background-color: #DFEBD6;
  border: 3px solid #2E2E2E;
  border-radius: 1.75rem;
}
.options {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 1rem;
  letter-spacing: 0.05em;
}
.select {
  margin-left: 0.4rem;
  padding: 0.3rem 0.75rem;
  background-color: #FFFFFF;
  color: #2E2E2E;
  font-size: 0.95rem;
  border: 2.5px solid #2E2E2E;
  border-radius: 999px;
  outline: none;
}
.checkbox {
  display: inline-flex;
  align-items: center;
  column-gap: 0.4rem;
}
.input-text {
  padding: 1rem 1.25rem;
  height: 20vh;
  box-sizing: border-box;
  background-color: #FFFFFF;
  color: #2E2E2E;
  font-size: 1.05rem;
  line-height: 1.6;
  letter-spacing: 0.03em;
  border: 3px solid #2E2E2E;
  border-radius: 1.25rem;
  outline: none;
  resize: vertical;
}
.input-text:focus {
  box-shadow: 0 0 0 4px #A9CBE0;
}
.actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  column-gap: 1rem;
}
.hint {
  color: #6B6B6B;
  font-size: 0.9rem;
}
.input-button {
  padding: 0.5rem 1.75rem;
  outline: none;
  border: 3px solid #2E2E2E;
  border-radius: 999px;
  background-color: #E8968A;
  color: #2E2E2E;
  font-size: 1.05rem;
  letter-spacing: 0.1em;
  cursor: pointer;
  transition: transform 0.1s ease, background-color 0.15s ease;
}
.input-button:hover {
  background-color: #E07F71;
}
.input-button:active {
  transform: translateY(2px);
}
.stop-button {
  background-color: #A9CBE0;
}
.stop-button:hover {
  background-color: #8FB8D3;
}
.chat {
  display: flex;
  flex-direction: column;
  row-gap: 1.25rem;
  width: 80vw;
  margin-top: 2rem;
  padding: 1.75rem 1.5rem;
  box-sizing: border-box;
  background-color: #F7F5EC;
  border: 3px solid #2E2E2E;
  border-radius: 1.75rem;
  align-items: end;
}
.chat-block {
  width: 60%;
  box-sizing: border-box;
  background-color: #FFFFFF;
  border: 3px solid #2E2E2E;
  border-radius: 1.25rem;
  padding: 1rem 1.25rem;
  line-height: 1.7;
  letter-spacing: 0.03em;
  white-space: pre-wrap;
}
.chat-agent {
  align-self: start;
  width: 60%;
}
.status {
  display: flex;
  align-items: center;
  column-gap: 0.75rem;
  margin-bottom: 0.75rem;
}
.status-tag {
  padding: 0.15rem 0.9rem;
  background-color: #B5D3A0;
  border: 2.5px solid #2E2E2E;
  border-radius: 999px;
  font-size: 0.9rem;
  letter-spacing: 0.08em;
}
.status-aborted {
  background-color: #E8968A;
}
.render-count {
  color: #6B6B6B;
  font-size: 0.9rem;
}
.chat-role {
  width: fit-content;
  margin-bottom: 0.6rem;
  padding: 0.15rem 0.9rem;
  border: 2.5px solid #2E2E2E;
  border-radius: 999px;
  color: #2E2E2E;
  font-size: 0.9rem;
  letter-spacing: 0.08em;
}
.chat-role-user {
  background-color: #E8968A;
}
</style>
