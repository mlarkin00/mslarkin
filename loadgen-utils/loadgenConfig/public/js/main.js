// This script initializes a Vue.js application for managing load generation configurations.
const { createApp } = Vue;

createApp({
  // The data function returns the initial state of the component's data.
  data() {
    return {
      // The config object holds the data for the form.
      config: {
        id: null, // The ID of the configuration, used when editing.
        targetUrl: "", // The URL to be targeted by the load generator.
        qps: null, // The number of queries per second.
        duration: null, // The duration of the load test in seconds.
        targetCpu: null, // The target CPU utilization for the load test.
      },
      // The configs array holds the list of all load generation configurations.
      configs: [],
      // The message string is used to display feedback to the user (e.g., success or error messages).
      message: "",
      // The error boolean is a flag to indicate if the last operation resulted in an error.
      error: false,
      // The isEditing boolean is a flag to indicate if the form is being used to edit an existing configuration.
      isEditing: false,
    };
  },
  // The methods object contains all the methods used in the Vue application.
  methods: {
    /**
     * Submits the form data to the server to create a new configuration or update an existing one.
     * It sends a POST request to /api/submit for new configurations
     * and a PUT request to /api/update/:id for existing configurations.
     */
    async submitForm() {
      try {
        const url = this.isEditing
          ? `/api/update/${this.config.id}`
          : "/api/submit";
        const method = this.isEditing ? "PUT" : "POST";

        const response = await fetch(url, {
          method: method,
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(this.config),
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
        this.message = "An unexpected error occurred.";
        this.error = true;
        console.error(error);
      }
    },
    /**
     * Loads all the existing load generation configurations from the server.
     * It sends a GET request to /api/configs and populates the configs array with the results.
     */
    async loadConfigs() {
      try {
        const response = await fetch("/api/configs");
        this.configs = await response.json();
      } catch (error) {
        console.error("Error loading configs:", error);
      }
    },
    /**
     * Deletes a configuration with the specified ID.
     * It sends a DELETE request to /api/delete/:id.
     * @param {number} id - The ID of the configuration to be deleted.
     */
    async deleteConfig(id) {
      if (!confirm("Are you sure you want to delete this config?")) {
        return;
      }
      try {
        const response = await fetch(`/api/delete/${id}`, {
          method: "DELETE",
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
        this.message = "An unexpected error occurred.";
        this.error = true;
        console.error(error);
      }
    },
    /**
     * Populates the form with the data of the configuration to be edited.
     * @param {object} config - The configuration object to be edited.
     */
    editConfig(config) {
      this.isEditing = true;
      this.config = { ...config };
    },
    /**
     * Toggles the active state of a configuration.
     * It sends a PUT request to /api/toggleActive/:id.
     * @param {number} id - The ID of the configuration to be toggled.
     */
    async toggleActive(id) {
      try {
        const response = await fetch(`/api/toggleActive/${id}`, {
          method: "PUT",
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
        this.message = "An unexpected error occurred.";
        this.error = true;
        console.error(error);
      }
    },
    /**
     * Resets the form to its initial state.
     */
    resetForm() {
      this.isEditing = false;
      this.config = {
        id: null,
        targetUrl: "",
        qps: null,
        duration: null,
        targetCpu: null,
      };
    },
  },
  /**
   * The mounted lifecycle hook is called after the component has been mounted to the DOM.
   * It calls the loadConfigs method to initially populate the list of configurations.
   */
  mounted() {
    this.loadConfigs();
  },
}).mount("#app");
