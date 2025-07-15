variable "network_name" {
  description = "The VPC network name."
  type        = string
}

variable "subnet_name" {
  description = "The subnet name."
  type        = string
}

variable "subnet_cidr" {
  description = "The subnet CIDR range."
  type        = string
}

variable "region" {
  description = "The region where the VPC and subnet will be created."
  type        = string
}

variable "external_ip_name" {
  description = "The name of the external IP address."
  type        = string
}