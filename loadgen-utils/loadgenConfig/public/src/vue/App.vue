<template>
  <div id="app">
    <h3>Load Generation Configuration</h3>
    <ConfigForm :config="config" :is-editing="isEditing" @submit-form="submitForm" @reset-form="resetForm" />
    <div class="toast-container align-items-center border-0 g-2">
      <div id="actionToast" class="toast" :class="{ 'text-bg-danger': error, 'text-bg-primary': !error }" role="alert"
        data-bs-autohide="true" data-bs-delay='5000' aria-live="assertive" aria-atomic="true">
        <div class="d-flex">
          <div class="toast-body">
            {{ message }}
          </div>
          <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"
            aria-label="Close"></button>
        </div>
      </div>
    </div>
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

      const actionToast = document.getElementById('actionToast');
      const toast = new bootstrap.Toast.getOrCreateInstance(actionToast);
      if (this.message) {
        toast.show();
      }
    },
    beforeUnmount() {
      clearInterval(this.timer);
    },
  }
</script>

<style>
  .error-message {
    color: red;
  }

  .success-message {
    color: green;
  }
</style>
