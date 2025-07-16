resource "google_compute_instance" "model_host_vm" {
  project      = var.project_id
  zone         = var.zone
  name         = var.vm_name
  machine_type = var.machine_type
  hostname     = var.hostname
  tags         = ["allow-ssh"]

  boot_disk {
    initialize_params {
      image = var.boot_disk_image
      size  = var.boot_disk_size_gb
      type  = "pd-ssd" # Often preferred for performance with ML workloads
    }
    mode = "READ_WRITE"
  }

  guest_accelerator {
    type  = var.gpu_type
    count = 1
  }

  network_interface {
    subnetwork = "ai-subnet"
    access_config {
      nat_ip = var.external_ip
    }
  }

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "TERMINATE"
    preemptible         = false
    provisioning_model  = "STANDARD"
  }

  metadata = {
    enable-oslogin      = "TRUE"
    enable-osconfig     = "TRUE"
    enable-os-inventory = "TRUE"
    startup-script      = "#!/bin/bash docker run -p 8080:8080 -d us-west1-docker.pkg.dev/mslarkin-tf/mslarkin-docker/vllm-backend:latest"
  }


  service_account {
    email = var.service_account_email
    # Full access to all Cloud APIs
    scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }

  labels = {
    goog-ops-agent-policy = "v2-x86-template-1-4-0"
  }
}

module "ops_agent_policy" {
  source        = "github.com/terraform-google-modules/terraform-google-cloud-operations/modules/ops-agent-policy"
  project       = "mslarkin-tf"
  zone          = "us-west1-b"
  assignment_id = "goog-ops-agent-v2-x86-template-1-4-0-us-west1-b"
  agents_rule = {
    package_state = "installed"
    version       = "latest"
  }
  instance_filter = {
    all = false
    inclusion_labels = [{
      labels = {
        goog-ops-agent-policy = "v2-x86-template-1-4-0"
      }
    }]
  }
}
