<script setup>
import { useGlobalStore } from '@/stores/global'
const store = useGlobalStore()

const roundFormat = new Intl.NumberFormat('en',
  {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0
  });


</script>

<template>
  <span v-if="store.connected">
    Number of container instances: <b>{{ roundFormat.format(store.instances) }}</b><br/>
    <i>Every container instance = 1 vCPU / 512 MiB</i>
    <div class="blocks">
      <div class="item" :class="{ medium: (store.clients <= 100) }" v-for="index in store.instances"></div>
    </div>
  </span>
  <span v-else>-</span>
</template>

<style scoped>
span {
  font-size: 1.5rem
}
.blocks {
  width: 100%;
  display: flex;
  flex-wrap: wrap;
}

div.medium {
  width: 2vw;  
  margin: 0.4vw;
}

.item {
  aspect-ratio: 1/1;
  background-color: #81C995;  
  border-radius: 2px;
  text-align: center;
  width: 1vw;
  margin: 0.2vw;
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  flex-direction: row-reverse;
}
</style>