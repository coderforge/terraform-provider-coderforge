# Terraform Provider for CoderForge.org

This provider is for the CoderForge.org Cloud service

## Resources

The provider supports the following resources:

- `coderforge_function` - Deploy serverless functions
- `coderforge_container` - Deploy containerized applications
- `coderforge_container_registry` - Manage container registries
- `coderforge_ks` - Deploy Kubernetes clusters (KS)
- `coderforge_cs` - Deploy container services (CS)

## KS Resource (Simplified)

The `coderforge_ks` resource provides focused Kubernetes cluster management with core fields:

- **Core Fields**:
  - `cluster_name` - Name of the Kubernetes cluster
  - `version` - Kubernetes version
  - `node_group_name` - Name of the node group
  - `node_instance_type` - Instance type for nodes
  - `node_min_size` - Minimum number of nodes
  - `node_max_size` - Maximum number of nodes
  - `node_desired_size` - Desired number of nodes

- **Inherited Fields** (from BaseResourceModel):
  - `security_group_ids` - Security group IDs
  - `logging_enabled` - Enable logging
  - `log_types` - Types of logs to collect
  - `tags` - Resource tags
  - `last_updated` - Last update timestamp (computed)

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