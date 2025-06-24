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
    Number of load-generating clients: <b>{{ roundFormat.format(store.clients) }}</b><br/>
    <i>A client sends requests in an infinite loop</i>
    <div class="blocks">      
      <div class="item" :class="{ medium: (store.clients <= 1_000), large: (store.clients <= 100), small: (store.clients <= 5_000), }" v-for="index in store.clients"></div>
    </div>
  </span>
  <span v-else>-</span>
</template>

<style scoped>
span {
  font-size: 1.5rem;
}
.blocks {
  width: 100%;
  display: flex;
  flex-wrap: wrap;
}

div.medium {
  width: 1vw;  
  margin: 0.2vw;
}

div.small {
  width: .5vw;  
  margin: 0.1vw;
}
div.large {
  width: 2vw;  
  margin: 0.4vw;
}

.item {
  aspect-ratio: 1/1;
  background-color: #F28B82;  
  border-radius: 40px;
  text-align: center;
  width: 0.1vw;
  margin: 1px;
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  flex-direction: row-reverse;
}
</style>