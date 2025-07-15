output "instance_name" {
  description = "The name of the created VM instance."
  value       = google_compute_instance.model_host_vm.name
}

output "instance_self_link" {
  description = "The self_link of the created VM instance."
  value       = google_compute_instance.model_host_vm.self_link
}