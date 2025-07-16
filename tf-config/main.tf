terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0" # Use an appropriate version
    }
  }

  backend "gcs" {
    bucket = "mslarkin-tf-state"
    # prefix = "terraform/state"
  }
}

locals {
  project_id = "mslarkin-tf"
  region     = "us-west1"
  zone       = "us-west1-b"
}

provider "google" {
  project = local.project_id
  region  = local.region
  zone    = local.zone
}

module "network" {
  source           = "./modules/network"
  project_id       = local.project_id
  network_name     = "ai-network"
  subnet_name      = "ai-subnet"
  subnet_cidr      = "10.0.0.0/20"
  region           = local.region
  external_ip_name = "model-host-ip"
}

module "model_host_vm" {
  source = "./modules/model_host"

  project_id            = local.project_id
  vm_name               = "model-host"
  zone                  = local.zone
  machine_type          = "n1-standard-8"
  gpu_type              = "nvidia-tesla-t4"
  hostname              = "model-host.mslarkin-tf"
  service_account_email = "model-host-sa@mslarkin-tf.iam.gserviceaccount.com"
  external_ip           = module.network.model_host_external_ip
  subnetwork            = module.network.subnet

  # This is the specific image family for Deep Learning VM with CUDA 12.4 M129
  # It's crucial to get the exact image name. You can list available images with:
  # gcloud compute images list --project deeplearning-platform-release --filter="name~'cuda-12-4-m129'"
  # The exact image name might change over time, so verify this.
  boot_disk_image   = "projects/ml-images/global/images/c0-deeplearning-common-cu124-v20250325-debian-11-py310" # Placeholder, see note below
  boot_disk_size_gb = 200
}

module "scraper-db" {
  source = "./modules/web-scraper"

  project_id = local.project_id
  region     = local.region
  db_name    = "scraper-db"
}