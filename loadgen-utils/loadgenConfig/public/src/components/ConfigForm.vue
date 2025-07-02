<template>
  <form @submit.prevent="$emit('submit-form', localConfig)">
    <input type="hidden" v-model="localConfig.id" />
    <div class="form-group">
      <label for="targetUrl">Target URL</label>
      <input type="text" class="form-control" id="targetUrl" v-model="localConfig.targetUrl" required>
    </div>
    <div class="form-group">
      <label for="qps">QPS</label>
      <input type="number" class="form-control" id="qps" v-model.number="localConfig.qps">
    </div>
    <div class="form-group">
      <label for="duration">Duration (seconds)</label>
      <input type="number" class="form-control" id="duration" v-model.number="localConfig.duration">
    </div>
    <div class="form-group">
      <label for="targetCpu">Target CPU (%)</label>
      <input type="number" class="form-control" id="targetCpu" v-model.number="localConfig.targetCpu">
    </div>
    <button type="submit" class="btn btn-primary">{{ isEditing ? 'Update' : 'Create' }}</button>
    <button type="button" class="btn btn-secondary" @click="$emit('reset-form')" v-if="isEditing">Cancel</button>
  </form>
  <!-- <div v-if="message" class="alert mt-3" :class="{'alert-success': !error, 'alert-danger': error}">{{ message }}</div> -->
  <!-- <div v-if="message" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
    <div class="toast-header">
      <img src="..." class="rounded mr-2" alt="...">
      <strong class="mr-auto">Bootstrap</strong>
      <small>11 mins ago</small>
      <button type="button" class="ml-2 mb-1 close" data-dismiss="toast" aria-label="Close">
        <span aria-hidden="true">&times;</span>
      </button>
    </div>
    <div class="toast-body">
      {{ message }}
    </div> -->
  <!-- </div> -->
</template>

<script>
  export default {
    props: {
      config: Object,
      isEditing: Boolean,
    },
    data() {
      return {
        localConfig: { ...this.config },
      };
    },
    watch: {
      config: {
        handler(newVal) {
          this.localConfig = { ...newVal };
        },
        deep: true,
      },
    },
  };
</script>
