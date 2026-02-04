<script setup>
defineProps({
  workloads: {
    type: Array,
    required: true
  }
})
</script>

<template>
  <div class="workload-table-container">
    <table class="workload-table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>Kind</th>
          <th>Ready</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="w in workloads" :key="w.namespace + '/' + w.name">
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
        <tr v-if="workloads.length === 0">
          <td colspan="5" class="empty-state">No workloads found.</td>
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
}

th {
  background-color: var(--color-background-mute);
  font-weight: 600;
  color: var(--color-heading);
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
  color: var(--color-text-light);
  font-size: 0.85rem;
  margin-left: 4px;
}

.empty-state {
  text-align: center;
  color: var(--color-text-light);
  padding: 2rem;
}
</style>
