terraform {
  required_providers {
    coderforge = {
      source = "terraform.coderforge.org/coderforge/coderforge"
    }
  }
}

provider "coderforge" {
  stack_id    = "stack-helloworld-dev"
  cloud_space = "helloworld.dev.coderforge.org"
  locations   = ["gbr-1", "gbr-2", "ita-1"]
}

resource "coderforge_container_registry" "example" {
  name       = "demo-container"
  runtime    = "docker"
  image_uri  = "docker.coderforge.org/demo:latest"
  timeout    = 120
  max_ram_size = "512MB"
}

output "container_registry" {
  value = coderforge_container_registry.example
}

