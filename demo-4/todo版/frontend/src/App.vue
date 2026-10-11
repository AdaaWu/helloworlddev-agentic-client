<script setup lang="ts">
  import { shallowRef, ref } from 'vue';
  interface EventCallback {
    onMessage?: (data: string) => void;
    onToolCalling?: (data: string) => void;
    onDone?: (data: string) => void;
    onError?: (data: string) => void;
  }

  interface SSERequest {
    message: string
  }

  const eventStatusList = [
    'message',
    'tool_calling',
    'done',
    'error'
  ] as const;
  type EventStatus = typeof eventStatusList[number];

  interface SSEEvent {
    id: string
    event: EventStatus
    data: string
    retry: number
  }

  type SetPayload = { field: 'id', value: string } |
    { field: 'event', value: string } |
    { field: 'data', value: string, dataList: string[] } |
    { field: 'retry', value: string };

  interface LLMResult {
    type: 'user' | 'ai',
    message: string
  }

  // state::
  const inputMessage = shallowRef('');
  const messageList = ref<LLMResult[]>([]);
  // methods::

  const isEvent = (input: unknown): input is EventStatus => {
    return eventStatusList.some(item => item === input);
  }
  const isInField = (input: unknown): input is keyof SSEEvent => {
    const sample: Record<keyof SSEEvent, true> = {
      id: true,
      event: true,
      data: true,
      retry: true,
    }
    return typeof input === 'string' && Object.hasOwn(sample, input);
  }

  const setPayloadAction = (result: SSEEvent, payload: SetPayload): void => {
    switch(payload.field) {
      case 'id':
        result.id = payload.value;
        break;
      case 'event':
        if (isEvent(payload.value)) result.event = payload.value;
        break;
      case 'data':
        payload.dataList.push(payload.value);
        break;
      case 'retry':
        if (!payload.value.trim()) return;
        result.retry = !Number.isNaN(+payload.value) ? +payload.value : 1;
        break;
    }
  }
  const triggerEvent = (sseEvent: SSEEvent, callbacks: EventCallback) => {
    const { onMessage, onToolCalling, onDone, onError } = callbacks;
    const { event, data } = sseEvent;
    switch(event) {
      case 'message':
        onMessage?.(data);
        break;
      case 'tool_calling':
        onToolCalling?.(data);
        break;
      case 'done':
        onDone?.(data);
        break;
      case 'error':
        onError?.(data);
        break;
    }
  }
  const handleEvent = (textList: string[], callbacks: EventCallback): void => {
    for(const text of textList) {
      if (!text.trim()) continue;
      const lineList = text.split('\n');
      const result: SSEEvent = {
        id: '',
        event: 'message',
        data: '',
        retry: 1,
      }
      const dataList: string[] = [];
      for (const line of lineList) {
        const idx = line.indexOf(':');
        if (idx === -1) continue;
        const key = line.slice(0, idx);
        const rawValue = line.slice(idx + 1);
        const value = rawValue.startsWith(' ') ? rawValue.slice(1) : rawValue;
        setPayloadAction(result, { field: isInField(key) ? key : 'data', value, dataList});
      }
      result.data = dataList.join('\n');
      triggerEvent(result, callbacks);
    }
  }

  const startEvent = async (callbacks: EventCallback): Promise<void> => {
    if (!inputMessage.value.trim()) return;
    const req: SSERequest = {
      message: inputMessage.value,
    }
    
    const response = await fetch('http://localhost:8080/sse', {
      method: 'POST',
      headers: {
        'Accept': "text/event-stream",
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(req),
    });

    const body = response.body;
    if (!body) {
      console.error('body null');
      return;
    }
    const reader = body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    try {
      while(true) {
        const { value, done } = await reader.read();
        if (done) {
          return;
        }
        buffer += decoder.decode(value, { stream: true });
        const textList = buffer.split('\n\n');
        buffer = textList.pop() ?? '';
        handleEvent(textList, callbacks);
      }
    } finally {
      reader.releaseLock();
      inputMessage.value = '';
    }
  }
  const handleClick = async (): Promise<void> => {
    if (!inputMessage.value.trim()) return;
    messageList.value.push({
      type: 'user',
      message: inputMessage.value,
    })
    const callbacks: EventCallback = {
      onMessage: (data) => {
        const currentMessageList = messageList.value;
        const lastMessage = currentMessageList[currentMessageList.length - 1];
        if (lastMessage.type === 'ai') {
          lastMessage.message += data;
          return;
        }
        currentMessageList.push({
          type: 'ai',
          message: data,
        })
      },
      onDone: () => {
        console.log('%c回應完成', 'color: lightgreen');
      }
    }

    await startEvent(callbacks);
  }
</script>

<template>
  <h1>demo-4 測試畫面</h1>
  <div class="input-block">
    <textarea class="input-text" v-model="inputMessage"></textarea>
    <div>
      <button class="input-button" @click="handleClick">送出</button>
    </div>
  </div>
  <div v-show="messageList.length > 0" class="chat">
    <div v-for="(messageObject, index) in messageList" 
      :key="index" 
      class="chat-block" 
      :class="{ 'chat-ai': messageObject.type === 'ai' }" 
    >
      <div 
        class="chat-role" 
        :class="[messageObject.type === 'ai' ? 'chat-role-ai' : 'chat-role-user']"
      >
        {{ messageObject.type === 'ai' ? 'AI 的回應' : '你的回應' }}
      </div>
      {{ messageObject.message }}
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
.input-block > div {
  display: flex;
  justify-content: flex-end;
}
.input-text:focus {
  box-shadow: 0 0 0 4px #A9CBE0;
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

.chat-ai {
  align-self: start;
  background-color: #DFEBD6;
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

.chat-role-ai {
  background-color: #B5D3A0;
}
</style>


