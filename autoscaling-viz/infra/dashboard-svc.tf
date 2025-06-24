resource "google_cloud_run_v2_service" "dashboard" {
  name     = "dashboard"
  location = var.region  

  template {    
    revision = null
    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }
    timeout = "60s"
    service_account = google_service_account.dashboard-svc.email
    containers {      
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      
      ports {
        container_port = 8080
        name = "http1"
      }
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
      env {
        name= "LOADER_CONCURRENCY"
        value = 2000
      }
      env {
        name= "TARGET_INSTANCES"
        value = 50
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

resource "google_service_account" "dashboard-svc" {
  account_id   = "cloud-run-dashboard-svc"
  display_name = "Cloud Run Dashboard Service"
}

resource "google_cloud_run_v2_service_iam_binding" "dashboard-public" {
  location = google_cloud_run_v2_service.dashboard.location
  name     = google_cloud_run_v2_service.dashboard.name
  role     = "roles/run.invoker"
  members  = ["allUsers"]
}

output "dashboard-svc" {
  value = google_cloud_run_v2_service.dashboard.uri
}

