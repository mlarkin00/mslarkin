const { createApp } = Vue

createApp({
  data() {
    return {
      config: {
        id: null,
        targetUrl: '',
        qps: null,
        duration: null,
        targetCpu: null
      },
      configs: [],
      message: '',
      error: false,
      isEditing: false
    }
  },
  methods: {
    async submitForm() {
      try {
        const url = this.isEditing ? `/api/update/${this.config.id}` : '/api/submit';
        const method = this.isEditing ? 'PUT' : 'POST';

        const response = await fetch(url, {
          method: method,
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(this.config)
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
          method: 'DELETE'
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
    resetForm() {
      this.isEditing = false;
      this.config = {
        id: null,
        targetUrl: '',
        qps: null,
        duration: null,
        targetCpu: null
      };
    }
  },
  mounted() {
    this.loadConfigs();
  }
}).mount('#app')
