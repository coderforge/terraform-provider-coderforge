terraform {
  required_providers {
    coderforge = {
      source = "terraform.coderforge.org/coderforge/coderforge"
    }
  }
  required_version = ">= 1.1.0"
}

provider "coderforge" {
  cloud_space = "ledger.dev.coderforge.org"
  locations = ["gbr-1", "gbr-2"]
}

resource "coderforge_function" "helloWorldFunction" {
  function_name = "helloWorld"
  code = {
    package_type = "container_image"
    image_uri = "docker.coderforge.org/function-hello-world:latest"
  }
  timeout = 180
  max_ram_size = "512MB"
}

output "func1_function" {
  value = coderforge_function.func1
}