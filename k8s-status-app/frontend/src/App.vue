<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <div class="app-container">
    <aside class="sidebar">
      <div class="logo">
        <h2>GKE Status</h2>
      </div>
      <nav>
        <RouterLink to="/" class="nav-item" active-class="active">Dashboard</RouterLink>
        <!-- Add more links if needed -->
      </nav>
    </aside>

    <main class="main-content">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.app-container {
  display: flex;
  min-height: 100vh;
}

.sidebar {
  width: 250px;
  background-color: var(--color-background-soft);
  border-right: 1px solid var(--color-border);
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
}

.logo h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 2rem;
  color: #1a73e8;
}

.nav-item {
  display: block;
  padding: 0.75rem 1rem;
  color: var(--color-text);
  text-decoration: none;
  border-radius: 6px;
  margin-bottom: 0.5rem;
  transition: background-color 0.2s;
}

.nav-item:hover {
  background-color: var(--color-background-mute);
}

.nav-item.active {
  background-color: #e8f0fe;
  color: #1a73e8;
  font-weight: 500;
}

.main-content {
  flex: 1;
  padding: 2rem;
  background-color: var(--color-background);
}
</style>
