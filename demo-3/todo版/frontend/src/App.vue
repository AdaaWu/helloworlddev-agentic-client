<script setup lang="ts">
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

type SetPayload = 
  { field: 'id', value: string} |
  { field: 'event', value: string} |
  { field: 'data', value: string, dataList: string[] } |
  { field: 'retry', value: string};


const startEvent = async (opts: EventOpts): Promise<void> => {
  const response = await fetch('http://localhost:8080/sse', {
    method: 'POST',
    headers: {
      'Accept': 'text/event-stream',
    }
  });
  const body = response.body;
  if (!body) {
    console.log('body null');
    return;
  }
  const reader = body.getReader();
  const decoder = new TextDecoder();
  try {
    while(true) {
      const { value, done } = await reader.read();
      if(done) return;
      const text = decoder.decode(value, { stream: true });
      console.log(text);
      // TODO:往這裡開始做 ...
      // 提示 : 因為這裡會收到像這樣的格式 : 
      // 舉個例子 event: message\ndata: hello cool\n\n
      // event: message\ndata: hello cool\n\nevent: message\ndata: hello wow\n\n
      // 所以要把每則訊息用 \n\n 分開
      // 並且把一整個訊息合在一起處理作為一整個 message
      // 提示 : 切割 \n\n，每個,每個 \n\n 之間有很多 \n 分割對應 event, data, id...etc
      // 我們要針對 event 是什麼去做什麼事，因為時間有限，我們這裡只做 message 所以 message 有通就 okay
      // 目的 : 我們要針對不同 event 對應不同 callback 去達成目的
      // 注意 : 有可能收到的是空字串這要判斷掉
    }
  } finally {
    reader.releaseLock();
  }
}

const handleClick = async (): Promise<void> => {
  const opts: EventOpts = {
    onMessage: (payload: string) => {
      console.log('收到的訊息 :', payload);
    }
  }
  await startEvent(opts);
}
</script>

<template>
  <button @click="handleClick">發射！！</button>
</template>
