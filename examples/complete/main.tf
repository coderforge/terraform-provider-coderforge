# -----------------------------------------------------------------------------
# A complete environment: a container service behind a port forward, a machine
# with a data disk beside it, and configuration held outside both.
#
#   export CODERFORGE_TOKEN=...
#   terraform init && terraform apply
# -----------------------------------------------------------------------------

terraform {
  required_version = ">= 1.5"
  required_providers {
    coderforge = {
      source  = "coderforge/coderforge"
      version = "~> 2.0"
    }
  }
}

provider "coderforge" {
  endpoint = var.endpoint
}

variable "endpoint" {
  type        = string
  default     = "http://srv12.net.coderforge.org:8080"
  description = "Base URL of the terraform-api service."
}

variable "database_password" {
  type        = string
  sensitive   = true
  description = "Password for the application database account."
}

# --- Configuration, kept out of the compose file ---

resource "coderforge_param" "app_settings" {
  name  = "demo-app-settings"
  value = "LOG_LEVEL=info\nWORKERS=4\n"
}

resource "coderforge_secret" "app_credentials" {
  name = "demo-app-credentials"
  data = {
    DATABASE_USER     = "demo"
    DATABASE_PASSWORD = var.database_password
  }
}

# --- The workload ---

resource "coderforge_container_service" "app" {
  name    = "demo-app"
  nodes   = 2
  compose = file("${path.module}/compose.yaml")

  env_param_rid  = coderforge_param.app_settings.id
  env_secret_rid = coderforge_secret.app_credentials.id

  # A disk per node, created and destroyed with the service.
  storage {
    type    = "new"
    size_gb = 20
    port    = 1
  }

  timeouts {
    create = "90m"
  }
}

# Reachable from outside, on the first node.
resource "coderforge_network_rule" "app_http" {
  src_port  = "8080"
  dest_ip   = coderforge_container_service.app.node_state[0].ip
  dest_port = "80"
  protocol  = "tcp"
}

# --- A standalone machine with its own disk ---

data "coderforge_vm_templates" "available" {}

resource "coderforge_machine" "database" {
  name     = "demo-db01"
  cpu      = 4
  ram      = 8192
  network  = "hostonly,nat"
  template = data.coderforge_vm_templates.available.files[0]

  timeouts {
    create = "45m"
  }
}

resource "coderforge_storage" "database_data" {
  name        = "demo-db-data"
  size_gb     = 100
  attached_to = coderforge_machine.database.name
  port        = 1
}

# --- Outputs ---

output "app_url" {
  value       = "http://${var.endpoint_host}:8080"
  description = "Where the demo application answers."
}

variable "endpoint_host" {
  type        = string
  default     = "srv12.net.coderforge.org"
  description = "Host the port forward is published on."
}

output "app_nodes" {
  value = {
    for node in coderforge_container_service.app.node_state :
    node.vm_name => node.ip
  }
}

output "database_address" {
  value = coderforge_machine.database.ip
}

output "database_password" {
  value     = coderforge_machine.database.password
  sensitive = true
}
