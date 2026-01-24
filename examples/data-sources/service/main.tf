terraform {
  required_providers {
    coderforge = {
      source  = "registry.terraform.io/coderforge/coderforge"
      version = "1.1.0"
    }
  }
}

provider "coderforge" {
  host_url    = "http://localhost:8080"
  token       = "read-only-token"
  cloud_space = "production"
}

# --- DATA SOURCE DEFINITION ---
# This data source reads a service that already exists on the server.
# It is NOT managed by this Terraform stack (no resource block).
data "coderforge_service" "legacy_app" {
  # The ID must match an existing resource ID on the backend
  id = "40c8055b-b92d-481b-8138-cb6d0d20d354"
}

# --- OUTPUTS ---
# Displaying the values fetched from the remote API
output "service_summary" {
  description = "Summary of the legacy service configuration"
  value = {
    service_name = data.coderforge_service.legacy_app.name
    cpu_cores    = data.coderforge_service.legacy_app.cpu
    is_virtual   = data.coderforge_service.legacy_app.virtual
    runtime      = data.coderforge_service.legacy_app.code.runtime
    command      = data.coderforge_service.legacy_app.code.execute_cmd
  }
}