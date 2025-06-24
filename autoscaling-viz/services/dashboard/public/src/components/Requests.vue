<script setup>
import { useGlobalStore } from '@/stores/global'
const store = useGlobalStore()

const precisionFormat = new Intl.NumberFormat('en',
{
    minimumFractionDigits: 0,
    maximumFractionDigits: 2
});

const roundFormat = new Intl.NumberFormat('en',
{
    minimumFractionDigits: 0,
    maximumFractionDigits: 0
});

async function reset() {
  // Send POST request using fetch with body
  let resp = await fetch('/api/reset', { method: 'POST' })
  await resp.json()
}

</script>

<template>
  <span @dblclick=reset() v-if="store.connected">
    <table>
    <tr>
      <td>Number of requests:</td><td class="value">{{ roundFormat.format(store.requests) }}</td>
    </tr>
    <tr>
      <td>Success percentage (HTTP 200):</td>
      <td v-if="store.requests == 0" class="value">-</td>
      <td v-else class="value">{{ precisionFormat.format(store.successPercentage) }}%</td>
    </tr>
    <!-- <tr>
      <td>Requests per second:</td>
      <td class="value">{{roundFormat.format(store.rate)}}</td>      
    </tr> -->
    <tr>
      <td>Latency (sampled):</td>
      <td v-if="store.rate == 0" class="value">-</td>
      <td v-else class="value">{{roundFormat.format(store.duration)}} milliseconds</td>      
    </tr>
  </table>
  </span>
  <span v-else>-</span>
</template>

<style scoped>

table {
  border-spacing: 1vw;
}
td.value {
  font-weight: bold;
}

</style>