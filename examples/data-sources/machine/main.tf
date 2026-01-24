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
# We read the existing machine data using its ID.
# This resource is NOT created or managed by Terraform.
data "coderforge_machine" "existing_db" {
  # The ID must match the JSON filename (without .json) on the server
  id = "e7b74bf6-7a5e-444a-8f79-f23153033b82"
}

# --- OUTPUTS ---
# We show the retrieved data to verify everything works
output "machine_details" {
  description = "Legacy machine details retrieved from the server"
  value = {
    host_name       = data.coderforge_machine.existing_db.name
    cpu_cores       = data.coderforge_machine.existing_db.cpu
    total_ram       = data.coderforge_machine.existing_db.ram
    is_virtual      = data.coderforge_machine.existing_db.virtual
    logs_group      = data.coderforge_machine.existing_db.logs_group
    security_groups = data.coderforge_machine.existing_db.security_group_ids

    # Accessing the specific "Owner" tag.
    # If the tag might be missing, consider using: lookup(data.coderforge_machine.existing_db.tags, "Owner", "N/A")
    owner_tag       = data.coderforge_machine.existing_db.tags["Owner"]
  }
}