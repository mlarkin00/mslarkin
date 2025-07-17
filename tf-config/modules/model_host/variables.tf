variable "project_id" {
  description = "The GCP project ID."
  type        = string
}

variable "vm_name" {
  description = "The name of the Compute Engine VM."
  type        = string
}

variable "hostname" {
  description = "The hostname of the Compute Engine VM."
  type        = string
}

variable "zone" {
  description = "The zone where the VM will be created."
  type        = string
}

variable "machine_type" {
  description = "The machine type for the VM."
  type        = string
}

variable "gpu_type" {
  description = "The type of GPU to attach to the VM."
  type        = string
}

variable "boot_disk_image" {
  description = "The boot disk image for the VM."
  type        = string
}

variable "boot_disk_size_gb" {
  description = "The size of the boot disk in GB."
  type        = number
}

variable "service_account" {
  description = "The service account for the VM."
  type        = string
}

variable "external_ip" {
  description = "The external IP address for the VM."
  type        = string
}

variable "subnet_name" {
  description = "The subnet name."
  type        = string
}
