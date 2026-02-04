<script setup>
defineProps({
  cluster: {
    type: Object,
    required: true
  }
})
</script>

<template>
  <div class="cluster-card">
    <div class="card-header">
      <h3>{{ cluster.cluster_name }}</h3>
      <span class="badge" :class="cluster.error ? 'error' : 'success'">
        {{ cluster.error ? 'Error' : 'Active' }}
      </span>
    </div>

    <div class="card-content">
      <div class="info-row">
        <span class="label">Project:</span>
        <span class="value">{{ cluster.project_id }}</span>
      </div>
      <div class="info-row">
        <span class="label">Location:</span>
        <span class="value">{{ cluster.location }}</span>
      </div>
      <div class="info-row">
        <span class="label">Nodes:</span>
        <span class="value">{{ cluster.nodes_ready }} / {{ cluster.node_count }}</span>
      </div>

      <div v-if="cluster.error" class="error-msg">
        {{ cluster.error }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.cluster-card {
  background: var(--color-background-soft);
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border: 1px solid var(--color-border);
  transition: transform 0.2s;
}

.cluster-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

h3 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--color-heading);
}

.badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: 600;
}

.badge.success {
  background-color: #e6f4ea;
  color: #1e8e3e;
}

.badge.error {
  background-color: #fce8e6;
  color: #c5221f;
}

.info-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
}

.label {
  color: var(--color-text-light);
}

.value {
  font-weight: 500;
  color: var(--color-text);
}

.error-msg {
  margin-top: 1rem;
  color: #c5221f;
  font-size: 0.85rem;
  padding: 0.5rem;
  background-color: #fce8e6;
  border-radius: 4px;
}
</style>
