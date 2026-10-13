<script setup lang="ts">
import { shallowRef } from 'vue';

const API_BASE = 'http://localhost:8081';

interface SSERequest {
  message: string
}

interface EventOpts {
  onMessage?: (payload: string) => void
  onToolCalling?: (payload: string) => void
  onDone?: (payload: string) => void
  onError?: (payload: string) => void
}

const eventStatusList = [
  'message',
  'tool_calling',
  'done',
  'error',
] as const;

type EventStatus = typeof eventStatusList[number];

interface SSEEvent {
  id: string
  event: EventStatus
  data: string
  retry: number
}

type SSEEventKey = keyof SSEEvent;

type ResultOptions =
  { field: 'id', value: string} |
  { field: 'event', value: string} |
  { field: 'data', value: string, dataList: string[] } |
  { field: 'retry', value: string};

const inputMessage = shallowRef('');

const isEvent = (input: unknown): input is EventStatus => {
  return eventStatusList.some(item => item === input);
}

const isSSEEventKey = (input: unknown): input is SSEEventKey => {
  const sample: Record<SSEEventKey, true> = {
    id: true,
    event: true,
    data: true,
    retry: true,
  }
  return typeof input === 'string' && Object.hasOwn(sample, input);
}

const setResult = (result: SSEEvent, resultOptions: ResultOptions) => {
  switch(resultOptions.field) {
    case 'id':
      result.id = resultOptions.value;
      break;
    case 'event':
      if (isEvent(resultOptions.value)) result.event = resultOptions.value;
      break;
    case 'data':
      resultOptions.dataList.push(resultOptions.value);
      break;
    case 'retry':
      if (!resultOptions.value.trim()) return;
      const retryNum = Number(resultOptions.value);
      result.retry = !Number.isNaN(retryNum) ? retryNum : 1;
      break;
  }
}

const triggerEvent = (result: SSEEvent, opts: EventOpts) => {
  const { onMessage, onToolCalling, onDone, onError } = opts;
  const { data } = result;
  switch(result.event) {
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

const handleEvent = (textList: string[], opts: EventOpts): void => {
  for(const text of textList) {
    if(!text.trim()) continue;
    const lineList = text.split('\n');
    const dataList: string[] = [];
    const result: SSEEvent = {
      id: '',
      event: 'message',
      data: '',
      retry: 1,
    }
    for(const line of lineList) {
      const idx = line.indexOf(':');
      if (idx === -1) continue;
      const key = line.slice(0, idx);
      const rawValue = line.slice(idx+1);
      const value = rawValue.startsWith(' ') ? rawValue.slice(1) : rawValue;
      setResult(result, { field: isSSEEventKey(key) ? key : 'data', value, dataList });
    }
    result.data = dataList.join('\n');
    triggerEvent(result, opts);
  }
}

const startEvent = async (opts: EventOpts): Promise<void> => {
  if (!inputMessage.value.trim()) return;
  const req: SSERequest = {
    message: inputMessage.value,
  }
  const response = await fetch(`${API_BASE}/sse`, {
    method: 'POST',
    headers: {
      'Accept': 'text/event-stream',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(req),
  });
  if (!response.ok) {
    console.log('不 okay');
    return;
  }
  const body = response.body;
  if (!body) {
    console.log('body null');
    return;
  }
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while(true) {
      const { value, done } = await reader.read();
      if(done) return;
      buffer += decoder.decode(value, { stream: true });
      const textList = buffer.split('\n\n');
      buffer = textList.pop() ?? '';
      handleEvent(textList, opts);
    }
  } finally {
    reader.releaseLock();
    inputMessage.value = '';
  }
}

const handleClick = async (): Promise<void> => {
  if (!inputMessage.value.trim()) return;
  const opts: EventOpts = {
    onMessage: (payload: string) => {
      console.log('收到的訊息 :', payload);
    }
  }
  await startEvent(opts);
}
</script>

<template>
  <textarea v-model="inputMessage"></textarea>
  <button @click="handleClick">送出</button>
</template>
