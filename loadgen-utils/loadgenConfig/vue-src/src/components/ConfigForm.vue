
<template>
  <form @submit.prevent="$emit('submit-form', localConfig)">
    <input type="hidden" v-model="localConfig.id" />
    <label for="targetUrl">Target URL:</label>
    <input type="text" id="targetUrl" v-model="localConfig.targetUrl" required />
    <label for="qps">QPS:</label>
    <input type="number" id="qps" v-model.number="localConfig.qps" />
    <label for="duration">Duration (seconds):</label>
    <input type="number" id="duration" v-model.number="localConfig.duration" />
    <label for="targetCpu">Target CPU:</label>
    <input type="number" id="targetCpu" v-model.number="localConfig.targetCpu" />
    <button type="submit">{{ isEditing ? 'Update' : 'Create' }}</button>
    <button type="button" @click="$emit('reset-form')" v-if="isEditing">Cancel</button>
  </form>
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
