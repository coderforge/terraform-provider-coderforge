# Terraform Provider for CoderForge.org

This provider is for the CoderForge.org Cloud service

## Resources

The provider supports the following resources:

- `coderforge_function` - Deploy serverless functions
- `coderforge_container` - Deploy containerized applications
- `coderforge_container_registry` - Manage container registries
- `coderforge_ks` - Deploy Kubernetes clusters (KS)
- `coderforge_cs` - Deploy container services (CS)

## KS Resource (Enhanced)

The `coderforge_ks` resource provides comprehensive Kubernetes cluster management with support for:

- **Cluster Management**: Configurable Kubernetes versions, cluster status monitoring
- **Node Group Configuration**: Instance types, scaling (min/max/desired), AMI types, disk sizes
- **Networking**: VPC and subnet configuration, security groups
- **Access Control**: Endpoint private/public access, public access CIDRs
- **Logging**: Configurable logging with multiple log types
- **Security**: Encryption configuration, node group taints and labels
- **Addons**: Kubernetes addon management (VPC CNI, CoreDNS, etc.)
- **IAM Integration**: Service roles, cluster roles, node roles
- **Monitoring**: Cluster and node group status, cluster endpoint, CA certificate
- **Tagging**: Comprehensive resource tagging

## CS Resource (Enhanced)

The `coderforge_cs` resource provides comprehensive container service management with support for:

- **Service Management**: Service and cluster configuration, status monitoring
- **Task Definition**: Family and revision management, launch types (Fargate/EC2)
- **Container Configuration**: Image, memory, CPU, port configuration
- **Networking**: VPC, subnets, security groups, load balancer integration
- **Environment**: Environment variables and secrets management
- **Health & Deployment**: Health check grace period, deployment configuration
- **IAM Integration**: Service roles, task roles, execution roles
- **Monitoring**: Service status, task status, running/pending counts
- **Compatibility**: Network mode and compatibility requirements
- **Tagging**: Comprehensive resource tagging