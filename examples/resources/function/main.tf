terraform {
  required_providers {
    coderforge = {
      source = "coderforge/coderforge"
      version = "0.1.2"
    }
  }
}

provider "coderforge" {
  stack_id = "stack-helloworld-dev"
  cloud_space = "helloworld.dev.coderforge.org"
  locations = ["gbr-1", "gbr-2", "ita-1"]
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
  value = coderforge_function.helloWorldFunction
}