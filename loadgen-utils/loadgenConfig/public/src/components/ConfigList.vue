<template>
  <div>
    <h2>Existing Configs</h2>
    <table class="table">
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
          <td v-if="c.duration < 0"></td>
          <td v-if="c.duration > 0">{{ c.duration }}</td>
          <td>{{ c.targetCpu }}</td>
          <td>
            <button class="btn btn-sm btn-primary" @click="$emit('edit-config', c)">Update</button>
            <button class="btn btn-sm btn-danger" @click="$emit('delete-config', c.id)">Delete</button>
            <button v-if="!c.active" class="btn btn-sm btn-success" @click="$emit('toggle-active', c.id)">Start</button>
            <button v-if="c.active" class="btn btn-sm btn-warning" @click="$emit('toggle-active', c.id)">Stop</button>
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
