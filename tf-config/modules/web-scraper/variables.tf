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

variable "subnet_name" {
  description = "The subnet name."
  type        = string
}

variable "service_account" {
  description = "The service account for the worker."
  type        = string
}

variable "network_name" {
  description = "The VPC network name."
  type        = string
}

variable "worker_pool_name" {
  description = "The Worker Pool name."
  type        = string
}
