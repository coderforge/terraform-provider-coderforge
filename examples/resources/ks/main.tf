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
  cluster_name           = "my-ks-cluster"
  version               = "1.28"
  region                = "us-east-1"
  node_group_name       = "my-node-group"
  node_instance_type    = "t3.medium"
  node_min_size         = 1
  node_max_size         = 3
  node_desired_size     = 2
  vpc_id                = "vpc-12345678"
  subnet_ids            = ["subnet-12345678", "subnet-87654321"]
  security_group_ids    = ["sg-12345678"]
  endpoint_private_access = true
  endpoint_public_access  = true
  public_access_cidrs   = ["0.0.0.0/0"]
  logging_enabled       = true
  log_types            = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
  
  tags = {
    Environment = "development"
    Project     = "my-project"
    Owner       = "devops-team"
  }
}

output "ks_cluster" {
  value = coderforge_ks.example
}