<script setup>
import { ref, onMounted, computed, watch } from 'vue';
import api from '../services/api';
import ClusterCard from '../components/ClusterCard.vue';
import WorkloadTable from '../components/WorkloadTable.vue';

const clusters = ref([]);
const loading = ref(true);
const error = ref(null);
const activeClusterName = ref(null);
const activeNamespace = ref(null);
const activeType = ref('Deployment');

const activeCluster = computed(() => {
  return clusters.value.find(c => c.cluster_name === activeClusterName.value) || null;
});

// Extract unique namespaces from active cluster workloads
const namespaces = computed(() => {
  if (!activeCluster.value || !activeCluster.value.workloads) return [];
  const nsSet = new Set(activeCluster.value.workloads.map(w => w.namespace));
  const filtered = Array.from(nsSet)
    .filter(ns => !ns.startsWith('kube-') && !ns.startsWith('gke-'))
    .sort();

  if (filtered.length > 0) {
    return ['All', ...filtered];
  }
  return [];
});

// Filter workloads based on active selection
const filteredWorkloads = computed(() => {
  if (!activeCluster.value || !activeCluster.value.workloads) return [];

  return activeCluster.value.workloads.filter(w => {
    // Filter System Namespaces always (unless we want a toggle for that later, but user said "all non-system")
    if (w.namespace.startsWith('kube-') || w.namespace.startsWith('gke-')) return false;

    // Filter by Namespace (if not "All")
    if (activeNamespace.value && activeNamespace.value !== 'All' && w.namespace !== activeNamespace.value) return false;

    // Filter by Type (Kind)
    // Note: Backend sends "Deployment", "Service".
    if (activeType.value && w.kind !== activeType.value) return false;

    return true;
  });
});

const fetchData = async () => {
  try {
    loading.value = true;
    error.value = null;
    const response = await api.getStatus();
    clusters.value = response.data;

    // Set default active cluster
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

// Auto-select "All" when cluster changes or data loads
watch([activeCluster, namespaces], ([newCluster, newNamespaces]) => {
  if (newNamespaces.length > 0 && (!activeNamespace.value || !newNamespaces.includes(activeNamespace.value))) {
    activeNamespace.value = 'All';
  }
}, { immediate: true });

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
      <!-- Cluster Tabs -->
      <div class="tab-bar cluster-tabs">
        <button
          v-for="cluster in clusters"
          :key="cluster.cluster_name"
          class="tab-btn"
          :class="{ active: activeClusterName === cluster.cluster_name }"
          @click="activeClusterName = cluster.cluster_name"
        >
          {{ cluster.cluster_name }}
          <span
            class="status-dot-small"
            :class="cluster.error ? 'error' : 'success'"
          ></span>
        </button>
      </div>

      <div v-if="activeCluster" class="cluster-content">
        <ClusterCard :cluster="activeCluster" />

        <div class="workloads-section">
          <h4>Workloads</h4>

          <!-- Namespace Tabs -->
          <div class="tabs-row namespace-tabs" v-if="namespaces.length > 0">
            <span class="tab-label">Namespace:</span>
            <button
              v-for="ns in namespaces"
              :key="ns"
              class="pill-btn"
              :class="{ active: activeNamespace === ns }"
              @click="activeNamespace = ns"
            >
              {{ ns }}
            </button>
          </div>

          <!-- Type Tabs (Sub-tabs) -->
          <div class="tabs-row type-tabs" v-if="activeNamespace">
             <button
              class="sub-tab-btn"
              :class="{ active: activeType === 'Deployment' }"
              @click="activeType = 'Deployment'"
            >
              Deployments
            </button>
            <button
              class="sub-tab-btn"
              :class="{ active: activeType === 'Service' }"
              @click="activeType = 'Service'"
            >
              Services
            </button>
          </div>

          <WorkloadTable
            :workloads="filteredWorkloads"
            :clusterInfo="activeCluster"
          />
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

/* Cluster Tabs */
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

.status-dot-small {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  display: inline-block;
}
.status-dot-small.success { background-color: #1e8e3e; }
.status-dot-small.error { background-color: #c5221f; }

.cluster-content {
  animation: fadeIn 0.3s ease;
}

.workloads-section {
  margin-top: 2rem;
  padding-left: 0.5rem;
}

h4 {
  margin-bottom: 1rem;
  color: var(--color-heading);
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0.5rem;
}

/* Namespace & Type Tabs */
.tabs-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
  overflow-x: auto;
  padding-bottom: 0.5rem;
}

.tab-label {
  font-weight: 500;
  color: var(--color-text-light);
  margin-right: 0.5rem;
}

.pill-btn {
  padding: 0.4rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: 20px;
  background-color: var(--color-background);
  color: var(--color-text);
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.2s;
  white-space: nowrap;
}

.pill-btn:hover {
  background-color: var(--color-background-soft);
  border-color: #1a73e8;
}

.pill-btn.active {
  background-color: #e8f0fe;
  color: #1a73e8;
  border-color: #1a73e8;
  font-weight: 500;
}

/* Dark mode pill adjustments */
@media (prefers-color-scheme: dark) {
  .pill-btn.active {
    background-color: rgba(26, 115, 232, 0.2);
  }
}

.type-tabs {
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 0; /* Attach to table visually */
  padding-left: 0.5rem;
}

.sub-tab-btn {
  padding: 0.6rem 1.2rem;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 0.95rem;
  color: var(--color-text-light);
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.sub-tab-btn:hover {
  color: var(--color-text);
  background-color: var(--color-background-soft);
}

.sub-tab-btn.active {
  color: #1a73e8;
  border-bottom-color: #1a73e8;
  font-weight: 500;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
