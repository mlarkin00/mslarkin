output "model_host_external_ip" {
  description = "The external IP address to be used for model host."
  value       = google_compute_address.model-host-ip.address
}
