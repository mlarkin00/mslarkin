resource "google_cloud_run_v2_service" "loader" {
  name     = "loader"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    scaling {
      min_instance_count = 0
      max_instance_count = 15
    }
    service_account = google_service_account.loader-svc.email
    containers {
      resources {
        cpu_idle = false
        limits = {
          "cpu"    = "1000m"
          "memory" = "1Gi"
        }
      }
      ports {
        container_port = 8080
        name           = "h2c"
      }
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      env {
        name  = "SQL_INSTANCE"
        value = google_sql_database_instance.main.connection_name
      }
      env {
        name  = "DB_USER"
        value = google_sql_user.app.name
      }
      env {
        name  = "DB_NAME"
        value = google_sql_database.analytics.name
      }
      env {
        name  = "DB_PASS" # TODO Use secret
        value = random_password.app-password.result
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].labels["client.knative.dev/nonce"],
      template[0].containers[0].image,
      client,
      client_version,
    ]
  }
}

resource "google_service_account" "loader-svc" {
  account_id   = "cloud-run-loader-svc"
  display_name = "Cloud Run Loader Service"
}

resource "google_cloud_run_v2_service_iam_binding" "loader-no-access" {
  location = google_cloud_run_v2_service.loader.location
  name     = google_cloud_run_v2_service.loader.name
  role     = "roles/run.invoker"
  members  = []
}

output "loader-svc" {
  value = google_cloud_run_v2_service.loader.uri
}

