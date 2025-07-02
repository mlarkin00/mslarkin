
<template>
  <div>
    <h2>Existing Configs</h2>
    <table>
      <thead>
        <tr>
          <th>Target URL</th>
          <th>QPS</th>
          <th>Duration</th>
          <th>Target CPU</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="c in configs" :key="c.id">
          <td>{{ c.targetUrl }}</td>
          <td>{{ c.qps }}</td>
          <td>{{ c.duration }}</td>
          <td>{{ c.targetCpu }}</td>
          <td>
            <button @click="$emit('edit-config', c)">Update</button>
            <button @click="$emit('delete-config', c.id)">Delete</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script>
export default {
  props: {
    configs: Array,
  },
};
</script>
