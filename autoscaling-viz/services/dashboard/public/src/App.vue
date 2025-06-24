<script setup>
import { ref } from 'vue'
import Test from './components/Test.vue'
import Instances from './components/Instances.vue'
import Clients from './components/Clients.vue'
import Requests from './components/Requests.vue'

import { useGlobalStore } from '@/stores/global'
const store = useGlobalStore()

function webSocketURL() {
  const protocol = (window.location.protocol === 'https:') ? 'wss://' : 'ws://';
  const url = `${protocol}${window.location.host}/api/ws`;
  return url;
}
const instanceCount = ref({})

async function fetchConfig() {
  let resp = await fetch('/api/config', { method: 'GET' })
  const { showContainers } = await resp.json()
  store.setShowContainers(showContainers ? true: false)
}
fetchConfig()

// Reconnecting WebSocket 
async function connect() {
  const minWaitMs = 0
  const maxWaitMs = 30000
  let socket = undefined
  let wait = 0
  while (true) {
    await new Promise((resolve) => {
      socket = new WebSocket(webSocketURL());
      socket.addEventListener('open', (event) => {
        wait = minWaitMs
        store.setConnected(true)
      });

      socket.addEventListener('error', (error) => {
        console.log(error)
        setTimeout(resolve, wait)
        wait = Math.min(wait + 1000, maxWaitMs)
      })

      socket.addEventListener('close', (event) => {
        store.setConnected(true)
        instanceCount.value = {}
        setTimeout(resolve, wait)
        wait = Math.min(wait + 1000, maxWaitMs)
      });

      socket.addEventListener('message', (event) => {
        let counts = JSON.parse(event.data)        
        if (!store.debugLayout) {
          store.updateCounts(counts)
        }
      })
    });
  }
}
connect();
</script>

<template>
  
  <div class="layout"> 
    <div class="header"><Test/></div>
    <div class="footer"><Requests/></div>
    <div v-if="store.connected && !store.showContainers" class="full"><Clients/></div>
    <div v-if="store.connected && store.showContainers" class="left"><Clients/></div>
    <div v-if="store.connected && store.showContainers" class="right"><Instances/></div>    
  </div>

</template>

<style scoped>
/* CSS Grid Layout */
.layout {
  font-size: 2rem;
  display: grid;
  grid-template-rows: auto auto 1fr;
  grid-template-columns: 1fr 1fr;
  height: 100vh;
  gap: 1vw;
}

.header {
  padding-top: 1vw;
  grid-row: 1 / 2;
  grid-column: 1 / 3;  
}

.footer {
  padding: 1vw;
  background-color: #DADCE0;
  border-radius: 5px;
  padding-top: 1vw;  
  grid-row: 2 / 3;
  grid-column: 1 / 3;    
}

.full {
  padding-top: 1vw;
  grid-column: 1 / 3;
}

.left {
  padding-top: 1vw;
  grid-column: 1 / 2;
}

.right {
  padding-top: 1vw;
  grid-column: 2 / 3;
}


</style>
