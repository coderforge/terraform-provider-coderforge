terraform {
  required_providers {
    coderforge = {
      source = "registry.terraform.io/coderforge/coderforge"
    }
  }
}

provider "coderforge" {
  stack_id    = "stack-ecs-dev"
  cloud_space = "ecs.dev.coderforge.org"
  locations   = ["us-east-1", "us-west-2"]
}

resource "coderforge_ecs" "example" {
  cluster_name            = "my-ecs-cluster"
  service_name            = "my-ecs-service"
  task_definition_family  = "my-task-family"
  task_definition_revision = "1"
  desired_count           = 2
  launch_type             = "FARGATE"
  platform_version        = "LATEST"
  region                  = "us-east-1"
  vpc_id                  = "vpc-12345678"
  subnet_ids              = ["subnet-12345678", "subnet-87654321"]
  security_group_ids      = ["sg-12345678"]
  load_balancer_arn       = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188"
  target_group_arn        = "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067"
  container_port           = 80
  container_name           = "my-container"
  container_image          = "nginx:latest"
  container_memory         = 512
  container_cpu            = 256
  health_check_grace_period = 300
  
  environment_variables = {
    NODE_ENV = "production"
    PORT     = "80"
    LOG_LEVEL = "info"
  }
  
  secrets = {
    DATABASE_URL = "arn:aws:secretsmanager:us-east-1:123456789012:secret:myapp/database-url"
    API_KEY      = "arn:aws:secretsmanager:us-east-1:123456789012:secret:myapp/api-key"
  }
  
  deployment_configuration = [
    "maximum_percent=200",
    "minimum_healthy_percent=100"
  ]
  
  tags = {
    Environment = "development"
    Project     = "my-project"
    Owner       = "devops-team"
  }
}

output "ecs_service" {
  value = coderforge_ecs.example
}