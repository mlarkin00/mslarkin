<script setup>
import {ref} from 'vue';
import { useGlobalStore } from '@/stores/global'
const store = useGlobalStore()

async function start(key) {
  active.value=key
  if( store.debugLayout ){
    if (key == "no-traffic") {
      store.updateCounts({ Instances: 0, Clients: 0, TotalRequests: 0, FailedRequests: 0  })
    }
    if (key == "regular") {
      store.updateCounts({ Instances: 20, Clients: 10, TotalRequests: 1_811, FailedRequests: 0 })
    }
    if (key == "surge") {
      store.updateCounts({ Instances: 500, Clients: 5_000, TotalRequests: 10_231, FailedRequests: 0 })
    }
    if (key == "insane") {
      store.updateCounts({ Instances: 1_000, Clients: 30_000, TotalRequests: 9_231_811, FailedRequests: 501 })
    }
  } else {
    if (key) {
      // Send POST request using fetch with body
      await fetch('/api/start/' + key, { method: 'POST' })
    }
  }
}

const active = ref('no-traffic')

</script>

<template>
<div>
  <button class="no-traffic" :class="{ active: (active=='no-traffic')}" @click="start('no-traffic')">No traffic</button>
  <button class="regular" :class="{ active: (active=='regular')}" @click="start('regular')">Regular traffic</button>
  <button class="surge" :class="{ active: (active=='surge')}" @click="start('surge')">Surge traffic</button>
  <button class="insane"  :class="{ active: (active=='insane')}" @click="start('insane')">Insane traffic</button>  
</div>
</template>

<style scoped>
div {
  font-size: 2em;
  display: flex;
  flex-direction: row;
  gap: 1vw;
  justify-content: space-around;  
}

button {
  border-radius: 3px;
  width: 100%;
  padding: .5vw;  
  border: 1px;
  font-size: 2.5rem;  
  cursor:pointer;
}

button.active {
  text-decoration: underline;
}

button.no-traffic {
  background-color: #669DF6;
}
button.no-traffic:hover {
  background-color: #174EA6;
  color: #F8F9FA;
}

button.regular {
  background-color: #FBBC04;
  border: 2px solid #FBBC04;
}
button.regular:hover {
  background-color: #F29900;
  color: #F8F9FA; 
}

button.surge {
  background-color: #F6AEA9;
}
button.surge:hover {
  background-color: #A50E0E;
  color: #F8F9FA;
}

button.insane {
  background-color: #C5221F;
  color: #F8F9FA;
}
button.insane:hover {
  background-color: #202124;
  color: #F8F9FA;
}

</style>


