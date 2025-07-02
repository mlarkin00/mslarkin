<template>
  <h3><small class="text-body-secondary">{{ isEditing ? 'Update Config' : 'Create New config' }}</small></h3>
  <div>
    <form @submit.prevent="$emit('submit-form', localConfig)">
      <input type="hidden" v-model="localConfig.id" />
      <div class="form-group">
        <label for="targetUrl">Target URL</label>
        <input type="text" class="form-control" id="targetUrl" v-model.trim="localConfig.targetUrl" required>
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
  </div>
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
