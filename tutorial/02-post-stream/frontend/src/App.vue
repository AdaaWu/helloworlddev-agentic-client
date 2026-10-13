<script setup lang="ts">
const API_BASE = 'http://localhost:8081';

const startEvent = async (): Promise<void> => {
  const response = await fetch(`${API_BASE}/sse`, {
    method: 'POST',
    headers: {
      'Accept': 'text/event-stream',
    }
  });
  if (!response.ok) {
    console.log('不 ok');
    return;
  }
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
    }
  } finally {
    reader.releaseLock();
  }
}

const handleClick = async (): Promise<void> => {
  await startEvent();
}
</script>

<template>
  <button @click="handleClick">發射！！</button>
</template>
