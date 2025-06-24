resource "google_artifact_registry_repository" "default" {
  location      = var.region
  repository_id = "default"
  description   = "Docker repository"
  format        = "DOCKER"
}
