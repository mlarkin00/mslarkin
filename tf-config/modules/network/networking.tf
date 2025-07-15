resource "google_compute_network" "ai-network" {
  name                    = var.network_name
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "ai-subnet" {
  name          = var.subnet_name
  ip_cidr_range = var.subnet_cidr
  region        = var.region
  network       = google_compute_network.ai-network.id
}

resource "google_compute_address" "model-host-ip" {
  name         = var.external_ip_name
  region       = var.region
  address_type = "EXTERNAL"
}
