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

resource "google_compute_firewall" "ai-network-ssh" {
  name        = "ai-network-ssh-allow"
  network     = google_compute_network.ai-network.id
  description = "Creates firewall rule allowing SSH on the ai-network"

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["allow-ssh"]
}

resource "google_compute_firewall" "ai-network-https" {
  name        = "ai-network-https-allow"
  network     = google_compute_network.ai-network.id
  description = "Creates firewall rule allowing HTTPS on the ai-network"

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["allow-https"]
}
