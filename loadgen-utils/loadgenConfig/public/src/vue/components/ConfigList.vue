<template>
  <h3><small class="text-body-secondary">Existing Configs</small></h3>
  <div class='g-3'>
    <h4><small class="text-body-secondary">Timed Configs</small></h4>
    <table class="table table-striped">
      <thead>
        <tr>
          <th scope="col"></th>
          <th scope="col">Target URL</th>
          <th scope="col">QPS</th>
          <th scope="col">Duration</th>
          <th scope="col">Target CPU</th>
          <th scope="col">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="tc in timedConfigs" :key="tc.id" :class="{ 'table-primary': tc.active }">
          <td>
            <div v-if="tc.active" class="spinner-border text-primary"></div>
          </td>
          <td>{{ tc.targetUrl }}</td>
          <td>{{ tc.qps }}</td>
          <td>{{ tc.duration }}</td>
          <td>{{ tc.targetCpu }}</td>
          <td>
            <button class="btn btn-sm btn-primary" @click="$emit('edit-config', tc)">Update</button>
            <button class="btn btn-sm btn-danger" @click="$emit('delete-config', tc.id)">Delete</button>
            <button v-if="!tc.active" class="btn btn-sm btn-success"
              @click="$emit('toggle-active', tc.id)">Start</button>
            <button v-if="tc.active" class="btn btn-sm btn-warning" @click="$emit('toggle-active', tc.id)">Stop</button>
          </td>
        </tr>
      </tbody>
      <h4><small class="text-body-secondary">Perpetual Configs</small></h4>
    </table>
    <table class="table table-striped">
      <thead>
        <tr>
          <th scope="col">Target URL</th>
          <th scope="col">QPS</th>
          <th scope="col">Target CPU</th>
          <th scope="col">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="pc in perpetualConfigs" :key="pc.id" :class="{ 'table-primary': pc.active }">
          <td>{{ pc.targetUrl }}</td>
          <td>{{ pc.qps }}</td>
          <td>{{ pc.targetCpu }}</td>
          <td>
            <button class="btn btn-sm btn-primary" @click="$emit('edit-config', pc)">Update</button>
            <button class="btn btn-sm btn-danger" @click="$emit('delete-config', pc.id)">Delete</button>
            <button v-if="!pc.active" class="btn btn-sm btn-success"
              @click="$emit('toggle-active', pc.id)">Start</button>
            <button v-if="pc.active" class="btn btn-sm btn-warning" @click="$emit('toggle-active', pc.id)">Stop</button>
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
    computed: {
      perpetualConfigs() {
        return this.configs.filter((config) => config.duration < 0);
      },
      timedConfigs() {
        return this.configs.filter((config) => config.duration >= 0);
      },
    },
  };
</script>
