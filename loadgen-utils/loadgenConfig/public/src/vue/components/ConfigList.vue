<template>
  <div>
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
      <tbody class="table-group-divider">
        <template v-for="c in configs" :key="c.id">
          <tr v-if="c.isPerpetual = false" :class="{ 'table-primary': c.active }">
            <td v-if="c.active">
              <div class="spinner-border text-primary"></div>
            </td>
            <td>{{ c.targetUrl }}</td>
            <td>{{ c.qps }}</td>
            <td v-if="!c.isPerpetual">{{ c.duration }}</td>
            <td>{{ c.targetCpu }}</td>
            <td>
              <button class="btn btn-sm btn-primary" @click="$emit('edit-config', c)">Update</button>
              <button class="btn btn-sm btn-danger" @click="$emit('delete-config', c.id)">Delete</button>
              <button v-if="!c.active" class="btn btn-sm btn-success"
                @click="$emit('toggle-active', c.id)">Start</button>
              <button v-if="c.active" class="btn btn-sm btn-warning" @click="$emit('toggle-active', c.id)">Stop</button>
            </td>
          </tr>
        </template>
      </tbody>
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
      <tbody class="table-group-divider">
        <template v-for="c in configs" :key="c.id">
          <tr v-if="c.isPerpetual" :class="{ 'table-primary': c.active }">
            <td>{{ c.targetUrl }}</td>
            <td>{{ c.qps }}</td>
            <td>{{ c.targetCpu }}</td>
            <td>
              <button class="btn btn-sm btn-primary" @click="$emit('edit-config', c)">Update</button>
              <button class="btn btn-sm btn-danger" @click="$emit('delete-config', c.id)">Delete</button>
              <button v-if="!c.active" class="btn btn-sm btn-success"
                @click="$emit('toggle-active', c.id)">Start</button>
              <button v-if="c.active" class="btn btn-sm btn-warning" @click="$emit('toggle-active', c.id)">Stop</button>
            </td>
          </tr>
        </template>
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
