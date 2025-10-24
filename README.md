# Terraform Provider for CoderForge.org

This provider is for the CoderForge.org Cloud service

## Resources

The provider supports the following resources:

- `coderforge_function` - Deploy serverless functions
- `coderforge_container` - Deploy containerized applications
- `coderforge_container_registry` - Manage container registries
- `coderforge_ks` - Deploy Kubernetes clusters (KS)
- `coderforge_cs` - Deploy container services (CS)
- `coderforge_eks` - Deploy Amazon EKS-like clusters
- `coderforge_ecs` - Deploy Amazon ECS-like services

## EKS Resource

The `coderforge_eks` resource provides Amazon EKS-like functionality with support for:

- Cluster management with configurable Kubernetes versions
- Node group configuration with instance types and scaling
- VPC and networking configuration
- Security group and subnet management
- Endpoint access control (private/public)
- Logging configuration
- Encryption settings
- Addon management
- Tagging support

## ECS Resource

The `coderforge_ecs` resource provides Amazon ECS-like functionality with support for:

- Service and cluster management
- Task definition configuration
- Fargate and EC2 launch types
- Load balancer integration
- Container configuration (image, memory, CPU)
- Environment variables and secrets
- Health check configuration
- Deployment configuration
- VPC and networking setup
- Tagging support