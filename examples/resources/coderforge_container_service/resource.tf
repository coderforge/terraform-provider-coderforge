# A single-file stack, the common case.
resource "coderforge_container_service" "edge" {
  name    = "edge-proxy"
  nodes   = 2
  compose = file("${path.module}/compose.yaml")
}

# A project that needs more than a compose file: build contexts, Dockerfiles,
# configuration. Pack the directory and hand over the archive.
resource "coderforge_container_service" "api" {
  name             = "api-stack"
  nodes            = 3
  archive_base64   = filebase64("${path.module}/api-project.zip")
  archive_filename = "api-project.zip"
  provisioner_name = "docker"

  # Environment comes from the parameter store and the vault, so no secret is
  # written into the compose file.
  env_param_rid  = coderforge_param.api_settings.id
  env_secret_rid = coderforge_secret.api_credentials.id

  # A disk on every node, created with the service and destroyed with it.
  storage {
    type    = "new"
    size_gb = 50
    port    = 1
  }

  # An existing disk, attached to the first node only. It outlives the service.
  storage {
    type       = "existing"
    rid        = coderforge_storage.archive.id
    port       = 2
    node_index = 0
  }

  timeouts {
    create = "90m"
  }
}

# A JSON object, because that is the only shape the platform accepts as an
# environment source: its keys become the environment variables.
resource "coderforge_param" "api_settings" {
  name = "api-settings"
  value = jsonencode({
    LOG_LEVEL = "info"
    WORKERS   = "4"
  })
}

resource "coderforge_secret" "api_credentials" {
  name = "api-credentials"
  data = {
    DATABASE_PASSWORD = var.database_password
  }
}

resource "coderforge_storage" "archive" {
  name    = "api-archive"
  size_gb = 200
}

variable "database_password" {
  type      = string
  sensitive = true
}

# The addresses of the machines the platform provisioned for this service.
output "api_node_addresses" {
  value = [for node in coderforge_container_service.api.node_state : node.ip]
}
