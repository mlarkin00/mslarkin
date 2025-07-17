resource "google_firestore_database" "scraper-db" {
  project     = var.project_id
  name        = var.db_name
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
}

resource "google_service_account" "scraper-sa" {
  account_id                   = var.service_account
  project                      = var.project_id
  create_ignore_already_exists = true
}

resource "google_project_iam_member" "scraper-sa-iam" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.scraper-sa.email}"
}

resource "google_cloud_run_v2_worker_pool" "scraper-worker" {
  name                = var.worker_pool_name
  location            = var.region
  project             = var.project_id
  deletion_protection = false
  launch_stage        = "BETA"

  scaling {
    manual_instance_count = 1
  }

  template {
    service_account = google_service_account.scraper-sa.email
    containers {
      image = "us-west1-docker.pkg.dev/mslarkin-tf/mslarkin-docker/web-scraper:latest"
      resources {
        limits = {
          cpu    = "2"
          memory = "4Gi"
        }
      }
    }

    vpc_access {
      network_interfaces {
        network    = var.network_name
        subnetwork = var.subnet_name
        # tags = ["tag1", "tag2", "tag3"]
      }
    }
  }
}
