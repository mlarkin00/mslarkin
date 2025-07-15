output "model_host_external_ip" {
  description = "The external IP address to be used for model host."
  value       = google_compute_address.model-host-ip.address
}

output "network_self_link" {
  description = "The self_link of the created VPC network."
  value       = google_compute_network.ai-network.self_link
}

output "subnet_self_link" {
  description = "The self_link of the created subnet."
  value       = google_compute_subnetwork.ai-subnet.self_link
}

output "subnet" {
  description = "The subnet for the VM."
  value       = google_compute_subnetwork.ai-subnet.name
}
