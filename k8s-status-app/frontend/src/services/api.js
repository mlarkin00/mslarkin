import axios from 'axios';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  },
});

export default {
  getStatus() {
    return apiClient.get("/status");
  },
  getPods(project, location, cluster, namespace, workload) {
    return apiClient.get("/pods", {
      params: { project, location, cluster, namespace, workload },
    });
  },
};
