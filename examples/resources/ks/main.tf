terraform {
  required_providers {
    coderforge = {
      source = "registry.terraform.io/coderforge/coderforge"
    }
  }
}

provider "coderforge" {
  stack_id    = "stack-ks-dev"
  cloud_space = "ks.dev.coderforge.org"
  locations   = ["us-east-1", "us-west-2"]
}

resource "coderforge_ks" "example" {
  cluster_name     = "my-ks-cluster"
  version         = "1.28"
  node_group_name = "my-node-group"
  node_instance_type = "t3.medium"
  node_min_size   = 1
  node_max_size   = 3
  node_desired_size = 2
  
  # Inherited fields from BaseResourceModel
  security_group_ids = ["sg-12345678"]
  logging_enabled   = true
  log_types        = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
  
  tags = {
    Environment = "development"
    Project     = "my-project"
    Owner       = "devops-team"
  }
}

output "ks_cluster" {
  value = coderforge_ks.example
}