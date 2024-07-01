provider "google" {
  project = "fc24-assignment"
  region  = "europe-west1"
}

resource "google_compute_network" "vpc_network" {
  name = "fc-network"
}

resource "google_compute_firewall" "default" {
  name    = "fc-network-firewall"
  description   = "This firewall allows external connections to our instance for ssh."
  network = google_compute_network.vpc_network.name
  direction = "INGRESS"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "tcp"
    ports    = ["22", "50051"]
  }
}

resource "google_compute_instance" "default" {
  name         = "ubuntu2204-vm"
  machine_type = "e2-standard-2"
  zone         = "europe-west1-b"

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
    }
  }

  network_interface {
    network = google_compute_network.vpc_network.name
    access_config {
      // Ephemeral IP
    }
  }

  metadata_startup_script = <<-EOT
    #!/bin/bash
    apt-get update
    apt-get install -y git python3-pip
    git clone https://github.com/LeanderAK/SS24_FOGC_Project.git /opt/SS24_FOGC_Project
    cd /opt/SS24_FOGC_Project
    pip3 install -r requirements.txt
    make cloud &
  EOT
}

output "instance_ip" {
  value = google_compute_instance.default.network_interface[0].access_config[0].nat_ip
}
