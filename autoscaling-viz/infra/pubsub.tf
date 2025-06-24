resource "google_pubsub_topic" "analytics" {
  name = "instances"
}

resource "google_service_account" "pubsub-invoker" {
  account_id   = "cloud-run-pubsub-invoker"
  display_name = "Cloud Run Pub/Sub Invoker"
}

resource "google_project_service_identity" "pubsub_agent" {
  provider = google-beta
  project  = var.project_id
  service  = "pubsub.googleapis.com"
}

resource "google_project_iam_binding" "project_token_creator" {
  project = var.project_id
  role    = "roles/iam.serviceAccountTokenCreator"
  members = ["serviceAccount:${google_project_service_identity.pubsub_agent.email}"]
}

resource "google_pubsub_subscription" "instances" {
  name  = "instances"
  topic = google_pubsub_topic.analytics.name
  push_config {
    push_endpoint = "${google_cloud_run_v2_service.collector.uri}/instances"
    no_wrapper {
      write_metadata = true
    }
    oidc_token {
      service_account_email = google_service_account.pubsub-invoker.email
    }
    attributes = {
      x-goog-version = "v1"
    }
  }
}