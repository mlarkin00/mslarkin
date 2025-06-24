import { ref } from 'vue'
import { defineStore } from 'pinia'
import { RingBuffer } from '../ring.js'


export const useGlobalStore = defineStore('global', () => {
  const instances = ref(0)
  const clients = ref(0)
  const requests = ref(0)
  const rate = ref(0)
  const duration = ref(0.0)
  const connected = ref(false)
  const successPercentage = ref(100)
  const debugLayout = ref(false)
  const showContainers = ref(false)

  const rateBuffer = new RingBuffer(20)
  
  function getRate(rps) {
    rateBuffer.enq(rps)
    if (rateBuffer.peekEnd() == 0) {
      return 0
    } 
    return rateBuffer.avg()
  }

  function updateCounts({
    Clients: c,
    Instances: i,
    TotalRequests: tr,
    FailedRequests: fr,
    RatePerSecond: rps,
    Duration: d,
  }) {
    instances.value = Math.min(i, 1_000) // We sometimes exceed 1,000
    requests.value = tr
    clients.value = c

    rate.value = getRate(rps)
    duration.value = d

    if (tr == 0) {
      successPercentage.value = 100
    } else {
      successPercentage.value = ((tr - fr) / tr) * 100
    }
  }

  function setShowContainers(value) {
    showContainers.value = value
  }

  function setConnected(value) {
    connected.value = value
  }

  return {
    instances, clients, requests,
    connected, successPercentage, debugLayout,
    showContainers, rate, duration,
    updateCounts, setConnected, setShowContainers
  }
})
