<script setup>
import { ref } from 'vue';
import api from '../services/api';

const props = defineProps({
  workloads: {
    type: Array,
    required: true
  },
  clusterInfo: {
    type: Object,
    required: false,
    default: () => ({})
  }
})

const expandedWorkloads = ref(new Set());
const podsData = ref({});
const loadingPods = ref(new Set());

const toggleExpand = async (workload) => {
  const key = `${workload.namespace}/${workload.name}`;
  if (expandedWorkloads.value.has(key)) {
    expandedWorkloads.value.delete(key);
  } else {
    expandedWorkloads.value.add(key);
    // Fetch pods if not already loaded (or maybe refresh?)
    // For now, fetch if not present or empty
    if (!podsData.value[key]) {
      await fetchPods(workload, key);
    }
  }
};

const fetchPods = async (workload, key) => {
  // We need project, location, cluster from somewhere.
  // Ideally passed as props or part of workload object if we enriched it?
  // Workload currently has: name, namespace, kind, ready, status, message via Aggregator.
  // BUT Dashboard.vue has the cluster info!
  // We need to pass cluster info to WorkloadTable.

  // NOTE: Assuming props.clusterInfo is passed now.
  if (!props.clusterInfo.project_id) {
    console.error("Missing cluster info for fetching pods");
    return;
  }

  try {
    loadingPods.value.add(key);
    const response = await api.getPods(
      props.clusterInfo.project_id,
      props.clusterInfo.location,
      props.clusterInfo.cluster_name,
      workload.namespace,
      workload.name
    );
    podsData.value[key] = response.data;
  } catch (err) {
    console.error("Failed to fetch pods", err);
    podsData.value[key] = { error: "Failed to load pods" };
  } finally {
    loadingPods.value.delete(key);
  }
};
</script>

<template>
  <div class="workload-table-container">
    <table class="workload-table">
      <thead>
        <tr>
          <th style="width: 30px"></th>
          <th>Name</th>
          <th>Namespace</th>
          <th>Kind</th>
          <th>Ready</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <template v-for="w in workloads" :key="w.namespace + '/' + w.name">
          <!-- Main Row -->
          <tr @click="toggleExpand(w)" class="workload-row" :class="{ expanded: expandedWorkloads.has(w.namespace + '/' + w.name) }">
            <td class="expand-icon">
              {{ expandedWorkloads.has(w.namespace + '/' + w.name) ? '▼' : '▶' }}
            </td>
            <td class="name-cell">{{ w.name }}</td>
            <td>{{ w.namespace }}</td>
            <td>{{ w.kind }}</td>
            <td>{{ w.ready }} / {{ w.desired }}</td>
            <td>
              <span class="status-dot" :class="w.status.toLowerCase()"></span>
              {{ w.status }}
              <span v-if="w.message" class="status-msg">({{ w.message }})</span>
            </td>
          </tr>

          <!-- Expanded Details Row -->
          <tr v-if="expandedWorkloads.has(w.namespace + '/' + w.name)" class="details-row">
            <td colspan="6">
              <div class="pods-container">
                <div v-if="loadingPods.has(w.namespace + '/' + w.name)" class="loading-pods">
                  Loading pods...
                </div>
                <div v-else-if="podsData[w.namespace + '/' + w.name]?.error" class="error-pods">
                  {{ podsData[w.namespace + '/' + w.name].error }}
                </div>
                <div v-else-if="podsData[w.namespace + '/' + w.name]?.length === 0" class="no-pods">
                  No pods found.
                </div>
                <div v-else class="pods-list">
                  <table class="pods-table">
                    <thead>
                      <tr>
                        <th>Pod Name</th>
                        <th>Phase</th>
                        <th>Node</th>
                        <th>IP</th>
                        <th>Age</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="pod in podsData[w.namespace + '/' + w.name]" :key="pod.name">
                        <td>{{ pod.name }}</td>
                        <td>
                          <span class="pod-status" :class="pod.phase.toLowerCase()">{{ pod.phase }}</span>
                        </td>
                        <td>{{ pod.node_name }}</td>
                        <td>{{ pod.pod_ip }}</td>
                        <td>{{ pod.age }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </td>
          </tr>
        </template>

        <tr v-if="workloads.length === 0">
          <td colspan="6" class="empty-state">No workloads found.</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.workload-table-container {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-background);
}

.workload-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.95rem;
}

th, td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
}

th {
  background-color: var(--color-background-mute);
  font-weight: 600;
  color: var(--color-heading);
}

.workload-row {
  cursor: pointer;
  transition: background-color 0.2s;
}

.workload-row:hover {
  background-color: var(--color-background-soft);
}

.workload-row.expanded {
  background-color: var(--color-background-soft);
  font-weight: 500;
}

.expand-icon {
  color: var(--color-text);
  font-size: 0.8rem;
  text-align: center;
}

tr:last-child td {
  border-bottom: none;
}

.name-cell {
  font-weight: 500;
  color: var(--color-heading);
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
}

.status-dot.healthy { background-color: #1e8e3e; }
.status-dot.degraded { background-color: #f9ab00; }
.status-dot.progressing { background-color: #1a73e8; }

.status-msg {
  color: var(--color-text);
  opacity: 0.8;
  font-size: 0.85rem;
  margin-left: 4px;
}

.empty-state {
  text-align: center;
  color: var(--color-text);
  opacity: 0.7;
  padding: 2rem;
}

/* Pod Details Styles */
.details-row td {
  padding: 0;
  background-color: var(--color-background-mute);
}

.pods-container {
  padding: 1rem 1rem 1rem 3rem;
  border-top: 1px solid var(--color-border);
  box-shadow: inset 0 2px 4px rgba(0,0,0,0.05);
}

.loading-pods, .no-pods, .error-pods {
  padding: 0.5rem;
  color: var(--color-text);
  opacity: 0.8;
  font-style: italic;
}

.error-pods {
  color: #c5221f;
  opacity: 1;
}

.pods-table {
  width: 100%;
  font-size: 0.85rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 4px;
}

.pods-table th {
  background-color: var(--color-background-soft);
  padding: 0.5rem;
  color: var(--color-heading);
}

.pods-table td {
  padding: 0.5rem;
  color: var(--color-text);
  border-bottom: 1px solid var(--color-border);
}

.pod-status {
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
  font-size: 0.8rem;
}

.pod-status.running { background-color: #e6f4ea; color: #1e8e3e; }
.pod-status.pending { background-color: #fef7e0; color: #f9ab00; }
.pod-status.failed { background-color: #fce8e6; color: #c5221f; }

/* Dark mode adjustments for specific badges if needed */
@media (prefers-color-scheme: dark) {
  .pod-status.running { background-color: rgba(30, 142, 62, 0.2); color: #81c995; }
  .pod-status.pending { background-color: rgba(249, 171, 0, 0.2); color: #fdd663; }
  .pod-status.failed { background-color: rgba(197, 34, 31, 0.2); color: #f28b82; }

  .status-dot.healthy { background-color: #81c995; }
  .status-dot.degraded { background-color: #fdd663; }
  .status-dot.progressing { background-color: #8ab4f8; }
}
</style>
