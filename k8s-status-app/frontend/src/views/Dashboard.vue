<script setup>
import { ref, onMounted, computed, watch } from 'vue';
import api from '../services/api';
import ClusterCard from '../components/ClusterCard.vue';
import WorkloadTable from '../components/WorkloadTable.vue';

const clusters = ref([]);
const loading = ref(true);
const error = ref(null);
const activeClusterName = ref(null);

const activeCluster = computed(() => {
  return clusters.value.find(c => c.cluster_name === activeClusterName.value) || null;
});

const fetchData = async () => {
  try {
    loading.value = true;
    error.value = null;
    const response = await api.getStatus();
    clusters.value = response.data;

    // Set default active cluster if none selected or current selection is gone
    if (clusters.value.length > 0) {
      const currentExists = clusters.value.some(c => c.cluster_name === activeClusterName.value);
      if (!activeClusterName.value || !currentExists) {
        activeClusterName.value = clusters.value[0].cluster_name;
      }
    }
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

    <div v-else-if="clusters.length > 0" class="cluster-tabs-container">
      <div class="tab-bar">
        <button
          v-for="cluster in clusters"
          :key="cluster.cluster_name"
          class="tab-btn"
          :class="{ active: activeClusterName === cluster.cluster_name }"
          @click="activeClusterName = cluster.cluster_name"
        >
          {{ cluster.cluster_name }}
          <span
            class="status-dot"
            :class="cluster.error ? 'error' : 'success'"
          ></span>
        </button>
      </div>

      <div v-if="activeCluster" class="cluster-content">
        <ClusterCard :cluster="activeCluster" />

        <div class="workloads-section">
          <h4>Workloads</h4>
          <WorkloadTable :workloads="activeCluster.workloads || []" />
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      No clusters found.
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

/* Tab Styles */
.cluster-tabs-container {
  display: flex;
  flex-direction: column;
}

.tab-bar {
  display: flex;
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 1.5rem;
  overflow-x: auto;
}

.tab-btn {
  padding: 0.75rem 1.5rem;
  background: none;
  border: none;
  border-bottom: 3px solid transparent;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 500;
  color: var(--color-text-light);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  white-space: nowrap;
}

.tab-btn:hover {
  background-color: var(--color-background-soft);
  color: var(--color-text);
}

.tab-btn.active {
  border-bottom-color: #1a73e8;
  color: #1a73e8;
  font-weight: 600;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.status-dot.success {
  background-color: #1e8e3e;
}

.status-dot.error {
  background-color: #c5221f;
}

.cluster-content {
  animation: fadeIn 0.3s ease;
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

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
