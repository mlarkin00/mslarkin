resource "google_firestore_database" "scraper-db" {
  project     = var.project_id
  name        = var.db_name
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
}