variable "project_id" {
  description = "The GCP project ID."
  type        = string
}

variable "region" {
  description = "The region where the Firestore DB will be created."
  type        = string
}

variable "db_name" {
  description = "The name of the Firestore DB."
  type        = string
}