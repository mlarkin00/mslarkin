<script setup>
import { ref } from 'vue'
import { useGlobalStore } from '@/stores/global'
const store = useGlobalStore()

const roundFormat = new Intl.NumberFormat('en',
{
    minimumFractionDigits: 0,
    maximumFractionDigits: 0
});

const precisionFormat = new Intl.NumberFormat('en',
{
    minimumFractionDigits: 3,
    maximumFractionDigits: 3
});

</script>

<template>
    <h3>Total requests</h3>
    <span v-if="store.serviceKey && store.testIDs[store.serviceKey]">
        <table>
            <tr>
                <td>Total number of requests:</td>            
                <td v-if="store.testIDs[store.serviceKey].Requests.TotalRequests">
                    {{ roundFormat.format(store.testIDs[store.serviceKey].Requests.TotalRequests) }}
                </td>
                <td v-else>
                    -
                </td>
            </tr>
            <tr>
                <td>Percentage of failed requests:</td>
                <td>{{ store.testIDs[store.serviceKey].Requests.StatusCodes }}</td>
            </tr>
        </table>
    </span>
    <span v-else>-</span>
    <h3>Instance scaling</h3>
    <span v-if="store.serviceKey && store.testIDs[store.serviceKey]">
        <table>
            <tr>
                <td>Time to scale to {{ store.testIDs[store.serviceKey].Scaling.TargetInstances }} instances (seconds):</td>                
                <td v-if="store.testIDs[store.serviceKey].Scaling.TimeToTarget.Valid">{{ precisionFormat.format(
                    store.testIDs[store.serviceKey].Scaling.TimeToTarget.Float64/1000
                 )}}</td>
                 <td v-else>-</td>
            </tr>
        </table>
    </span>
    <span v-else>-</span>
</template>

<style scoped>
h3 {
    cursor: pointer;
}
</style>