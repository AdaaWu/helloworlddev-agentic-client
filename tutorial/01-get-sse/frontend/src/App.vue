<script setup lang="ts">
const API_BASE = 'http://localhost:8081';

const sse = new EventSource(`${API_BASE}/sse`);
sse.addEventListener('message', (ev) => {
  const data = ev.data;
  console.log('你收到的訊息在這 : ', data);
})
</script>

<template>
</template>
