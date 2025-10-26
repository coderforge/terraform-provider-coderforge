terraform {
  required_providers {
    coderforge = {
      source = "registry.terraform.io/coderforge/coderforge"
    }
  }
}

provider "coderforge" {
  stack_id    = "stack-cs-dev"
  cloud_space = "cs.dev.coderforge.org"
  locations   = ["us-east-1", "us-west-2"]
}

resource "coderforge_cs" "example" {
  service_name      = "my-cs-service"
  desired_count     = 2
  platform_version  = "LATEST"
  container_port    = 80
  container_name    = "my-container"
  container_image   = "nginx:latest"
  container_memory  = 512
  container_cpu     = 256
  
  environment_variables = {
    NODE_ENV = "production"
    PORT     = "80"
    LOG_LEVEL = "info"
  }
  
  # Inherited fields from BaseResourceModel
  security_group_ids = ["sg-12345678"]
  logging_enabled   = true
  log_types        = ["application", "platform"]
  
  tags = {
    Environment = "development"
    Project     = "my-project"
    Owner       = "devops-team"
  }
}

output "cs_service" {
  value = coderforge_cs.example
}