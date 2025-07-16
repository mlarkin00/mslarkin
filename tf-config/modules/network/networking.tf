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

resource "google_compute_firewall" "ai-network-icmp" {
  name        = "ai-network-icmp-allow"
  network     = google_compute_network.ai-network.id
  description = "Creates firewall rule allowing ICMP on the ai-network"

  allow {
    protocol = "icmp"
  }

  source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "ai-network-rdp" {
  name        = "ai-network-rdp-allow"
  network     = google_compute_network.ai-network.id
  description = "Creates firewall rule allowing RDP on the ai-network"

  allow {
    protocol = "tcp"
    ports    = ["3389"]
  }

  source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "ai-network-internal" {
  name        = "ai-network-internal-allow"
  network     = google_compute_network.ai-network.id
  description = "Creates firewall rule allowing internal traffic on the ai-network"

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "icmp"
  }

  source_ranges = ["0.0.0.0/0"]
}
