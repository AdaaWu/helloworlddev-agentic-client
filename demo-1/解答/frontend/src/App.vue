<script setup lang="ts">
const sse = new EventSource('http://localhost:8080/sse');
sse.addEventListener('message', (ev) => {
  const data = ev.data;
  console.log('你收到的訊息在這 : ', data);
})
</script>

<template>
</template>
