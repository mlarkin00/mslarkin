resource "google_sql_database_instance" "main" {
  database_version    = "POSTGRES_15"
  deletion_protection = true
  name                = "sql"
  project             = var.project_id
  region              = var.region

  settings {
    availability_type           = "ZONAL"
    deletion_protection_enabled = true
    disk_autoresize             = true
    disk_autoresize_limit       = 0
    disk_size                   = 10
    disk_type                   = "PD_SSD"
    edition                     = "ENTERPRISE"
    pricing_plan                = "PER_USE"
    tier                        = "db-custom-2-8192"

    backup_configuration {
      enabled = false
    }

    maintenance_window { # Sunday, 00:00 – 01:00 GMT+2
      day  = 6           # Saturday
      hour = 22
    }

    database_flags {
      name  = "cloudsql.iam_authentication"
      value = "on"
    }

    database_flags {
      name  = "max_connections"
      value = "20000"
    }

    insights_config {
      query_insights_enabled = true
    }
  }

  timeouts {}
}

resource "random_password" "app-password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

output "sql-password" {
  value     = random_password.app-password
  sensitive = true
}

output "sql-connection-name" {
  value = google_sql_database_instance.main.connection_name
}

resource "google_sql_user" "app" {
  name     = "app"
  instance = google_sql_database_instance.main.name
  type     = "BUILT_IN"
  password = random_password.app-password.result
}

resource "google_sql_database" "analytics" {
  name            = "analytics"
  instance        = google_sql_database_instance.main.name
  deletion_policy = "ABANDON"
}

resource "google_project_iam_binding" "sql_clients" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  members = [
    "serviceAccount:${google_service_account.loader-svc.email}",
    "serviceAccount:${google_service_account.collector-svc.email}",
    "serviceAccount:${google_service_account.dashboard-svc.email}"
  ]
}
