terraform {
  required_providers {
    coderforge = {
      source = "registry.terraform.io/coderforge/coderforge"
    }
  }
}

provider "coderforge" {
  stack_id    = "stack-eks-dev"
  cloud_space = "eks.dev.coderforge.org"
  locations   = ["us-east-1", "us-west-2"]
}

resource "coderforge_eks" "example" {
  cluster_name           = "my-eks-cluster"
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
  encryption_config    = ["secrets"]
  addons              = ["vpc-cni", "coredns", "kube-proxy", "aws-ebs-csi-driver"]
  
  tags = {
    Environment = "development"
    Project     = "my-project"
    Owner       = "devops-team"
  }
}

output "eks_cluster" {
  value = coderforge_eks.example
}