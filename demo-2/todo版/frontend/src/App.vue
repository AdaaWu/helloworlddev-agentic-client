<script setup lang="ts">
//const sse = new EventSource('http://localhost:8080/sse');
//sse.addEventListener('message', (ev) => {
//  const data = ev.data;
//  console.log('你收到的訊息在這 : ', data);
//})
// 以上就用不到了，不用參考了
// ======

// 我這裡寫兩個 function 是故意的，因為下個 demo 要拆，所以先寫好
const startEvent = async (): Promise<void> => {
  // TODO: 從這裡開始實作
  // 提示 : 這裡我們要用 fetch API 所以先把 fetch 搞出來
  // 提示2 : 我們常用 response.json() 去處理 json
  // 但我們這裡不會用它，要處理 response.body
  // response.body.getReader() 可以取得 ReadableStream 的東東，就說到這了
  // 提示 3 : 因為 Stream 過來是 uint8Array 也可以理解為 binary 所以我們需要 decode 它才能正常顯示
  // decode 可以用 new TextDecoder('utf-8') 處理: 參考 -> https://developer.mozilla.org/en-US/docs/Web/API/TextDecoder
  // 記得結尾要 releaseLock
  // 加油 GoGo 🏃
}

const handleClick = async (): Promise<void> => {
  await startEvent();
}
</script>

<template>
  <button @click="handleClick">發射！！</button>
</template>
