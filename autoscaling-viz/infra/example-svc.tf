resource "google_cloud_run_v2_service" "example" {
  name     = "example"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    scaling {
      min_instance_count = 0
      max_instance_count = 1000
    }
    max_instance_request_concurrency = 40
    timeout                          = "10s"
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      ports {
        container_port = 8080
        name           = "h2c"
      }
      env {
        name  = "PUB_SUB_TOPIC"
        value = google_pubsub_topic.analytics.name
      }
      env {
        name  = "RESPONSE_DELAY"
        value = "1"
      }
    }

  }
  lifecycle {
    ignore_changes = [
      template[0].labels["client.knative.dev/nonce"],
      template[0].containers[0].image,
      template[0].revision,
      client,
      client_version,
    ]
  }
}

output "example-svc" {
  value = google_cloud_run_v2_service.example.uri
}

resource "google_cloud_run_v2_service_iam_member" "example-public" {
  location = google_cloud_run_v2_service.example.location
  name     = google_cloud_run_v2_service.example.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

