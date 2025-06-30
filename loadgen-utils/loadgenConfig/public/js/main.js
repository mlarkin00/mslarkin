const { createApp } = Vue

createApp({
  data() {
    return {
      config: {
        targetUrl: '',
        qps: null,
        duration: null,
        targetCpu: null
      },
      message: ''
    }
  },
  methods: {
    async submitForm() {
      try {
        const response = await fetch('/api/submit', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(this.config)
        });
        const data = await response.json();
        if (response.ok) {
          this.message = data.message;
          this.config = {
            targetUrl: '',
            qps: null,
            duration: null,
            targetCpu: null
          };
        } else {
          this.message = `Error: ${data.error}`;
        }
      } catch (error) {
        this.message = 'An unexpected error occurred.';
        console.error(error);
      }
    }
  }
}).mount('#app')
