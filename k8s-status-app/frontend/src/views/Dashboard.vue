<script setup>
import { ref, onMounted } from 'vue';
import api from '../services/api';
import ClusterCard from '../components/ClusterCard.vue';
import WorkloadTable from '../components/WorkloadTable.vue';

const clusters = ref([]);
const loading = ref(true);
const error = ref(null);

const fetchData = async () => {
  try {
    loading.value = true;
    error.value = null;
    const response = await api.getStatus();
    clusters.value = response.data;
  } catch (err) {
    error.value = "Failed to fetch cluster status. Is backend running?";
    console.error(err);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchData();
  // Poll every 10 seconds
  setInterval(fetchData, 10000);
});
</script>

<template>
  <div class="dashboard">
    <div class="header">
      <h1>Cluster Status</h1>
      <button @click="fetchData" class="refresh-btn" :disabled="loading">
        {{ loading ? 'Refreshing...' : 'Refresh' }}
      </button>
    </div>

    <div v-if="error" class="error-banner">
      {{ error }}
    </div>

    <div v-if="loading && clusters.length === 0" class="loading-state">
      Loading cluster data...
    </div>

    <div v-else class="cluster-list">
      <div v-for="cluster in clusters" :key="cluster.cluster_name" class="cluster-section">
        <ClusterCard :cluster="cluster" />

        <div class="workloads-section">
          <h4>Workloads</h4>
          <WorkloadTable :workloads="cluster.workloads || []" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 1rem;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

h1 {
  font-size: 1.8rem;
  font-weight: 600;
  color: var(--color-heading);
}

.refresh-btn {
  padding: 0.5rem 1rem;
  background-color: #1a73e8;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
}

.refresh-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.error-banner {
  background-color: #fce8e6;
  color: #c5221f;
  padding: 1rem;
  border-radius: 6px;
  margin-bottom: 1.5rem;
  border: 1px solid #f6aea9;
}

.cluster-section {
  margin-bottom: 3rem;
  background: var(--color-background);
  padding: 1rem;
  border-radius: 8px;
}

.workloads-section {
  margin-top: 1.5rem;
  padding-left: 0.5rem;
}

h4 {
  margin-bottom: 1rem;
  color: var(--color-heading);
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0.5rem;
}
</style>
