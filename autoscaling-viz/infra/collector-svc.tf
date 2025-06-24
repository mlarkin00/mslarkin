resource "google_cloud_run_v2_service" "collector" {
  name     = "collector"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = google_service_account.collector-svc.email
    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }
    containers {
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

resource "google_service_account" "collector-svc" {
  account_id   = "cloud-run-collector-svc"
  display_name = "Cloud Run Collector Service"
}

output "collector-svc" {
  value = google_cloud_run_v2_service.collector.uri
}

resource "google_cloud_run_v2_service_iam_binding" "collector" {
  location = google_cloud_run_v2_service.collector.location
  name     = google_cloud_run_v2_service.collector.name
  role     = "roles/run.invoker"
  members  = ["serviceAccount:${google_service_account.pubsub-invoker.email}"]
}

