<template>
  <div id="app">
    <h1>Load Generation Configuration</h1>
    <ConfigForm :config="config" :is-editing="isEditing" @submit-form="submitForm" @reset-form="resetForm" />
    <div>
      <p v-if="message" :class="{ 'error-message': error, 'success-message': !error }">{{ message }}</p>
    </div>
    <div class="toast-container position-fixed bottom-0 end-0 p-3">
      <div v-if="message" class="toast show" :class="{ 'bg-danger': error, 'bg-success': !error }" role="alert"
        aria-live="assertive" aria-atomic="true">
        <div class="toast-header">
          <strong class="me-auto">{{ isEditing ? 'Config Updated' : 'Config Created' }}</strong>
          <button type="button" class="btn-close" data-bs-dismiss="toast" aria-label="Close"></button>
        </div>
        <div class="toast-body">
          {{ message }}
        </div>
      </div>
      <h2>Existing Configs</h2>
      <ConfigList :configs="configs" @delete-config="deleteConfig" @edit-config="editConfig"
        @toggle-active="toggleActive" />
    </div>
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
    computed: {
      isPerpetual() {
        return this.config.duration < 0 ? true : false;
      }
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
