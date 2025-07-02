<template>
  <div id="app">
    <h1>Load Generation Configuration</h1>
    <ConfigForm :config="config" :is-editing="isEditing" @submit-form="submitForm" @reset-form="resetForm" />
    <p v-if="message" :class="{ 'error-message': error, 'success-message': !error }">{{ message }}</p>
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
        Hello, world! This is a toast message.
      </div>
    </div> -->
    <h2>Existing Configs</h2>
    <ConfigList :configs="configs" @delete-config="deleteConfig" @edit-config="editConfig"
      @toggle-active="toggleActive" />
  </div>
</template>

<script>
  import ConfigForm from './components/ConfigForm.vue';
  import ConfigList from './components/ConfigList.vue';

  export default {
    components: {
      ConfigForm,
      ConfigList,
    },
    data() {
      return {
        config: {
          id: null,
          targetUrl: '',
          qps: null,
          duration: null,
          targetCpu: null,
        },
        configs: [],
        message: '',
        error: false,
        isEditing: false,
      };
    },
    methods: {
      async submitForm(configData) {
        try {
          const url = this.isEditing ? `/api/update/${configData.id}` : '/api/submit';
          const method = this.isEditing ? 'PUT' : 'POST';

          const response = await fetch(url, {
            method: method,
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify(configData),
          });
          const data = await response.json();
          if (response.ok) {
            this.message = data.message;
            this.error = false;
            this.resetForm();
            this.loadConfigs();
          } else {
            this.message = `Error: ${data.error}`;
            this.error = true;
          }
        } catch (error) {
          this.message = 'An unexpected error occurred.';
          this.error = true;
          console.error(error);
        }
      },
      async loadConfigs() {
        try {
          const response = await fetch('/api/configs');
          this.configs = await response.json();
        } catch (error) {
          console.error('Error loading configs:', error);
        }
      },
      async deleteConfig(id) {
        if (!confirm('Are you sure you want to delete this config?')) {
          return;
        }
        try {
          const response = await fetch(`/api/delete/${id}`, {
            method: 'DELETE',
          });
          const data = await response.json();
          if (response.ok) {
            this.message = data.message;
            this.error = false;
            this.loadConfigs();
          } else {
            this.message = `Error: ${data.error}`;
            this.error = true;
          }
        } catch (error) {
          this.message = 'An unexpected error occurred.';
          this.error = true;
          console.error(error);
        }
      },
      editConfig(config) {
        this.isEditing = true;
        this.config = { ...config };
      },
      async toggleActive(id) {
        try {
          const response = await fetch(`/api/toggleActive/${id}`, {
            method: 'PUT',
          });
          const data = await response.json();
          if (response.ok) {
            this.message = data.message;
            this.error = false;
            this.loadConfigs();
          } else {
            this.message = `Error: ${data.error}`;
            this.error = true;
          }
        } catch (error) {
          this.message = 'An unexpected error occurred.';
          this.error = true;
          console.error(error);
        }
      },
      resetForm() {
        this.isEditing = false;
        this.config = {
          id: null,
          targetUrl: '',
          qps: null,
          duration: null,
          targetCpu: null,
        };
      },
    },
    mounted() {
      this.loadConfigs(); // Initial load
      this.timer = setInterval(this.loadConfigs, 30000); // Load configs every 30 seconds
    },
    beforeUnmount() {
      clearInterval(this.timer);
    },
  };
</script>

<style>
  .error-message {
    color: red;
  }

  .success-message {
    color: green;
  }
</style>
