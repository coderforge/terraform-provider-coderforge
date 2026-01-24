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
  token       = "mock-token-123"
  cloud_space = "dev-space"
  stack_id    = "stack-1"
  locations   = ["us-east-1"]
}

resource "coderforge_service" "my_test_service" {
  name = "local-test-service"
  cpu  = 1
  ram  = "256MB"
  code = {
    runtime     = "nodejs"
    execute_cmd = "npm start"
    image_uri   = "node:14-alpine"
  }
}

data "coderforge_service" "existing" {
  id = coderforge_service.my_test_service.id
}

output "data_source_name" {
  value = data.coderforge_service.existing.name
}