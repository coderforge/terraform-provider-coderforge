# CoderForge Terraform Provider Examples

This directory contains examples for the CoderForge Terraform Provider. These files are used for official documentation generation and manual testing.

## Resource Overview

### 1. CoderForge Service (`coderforge_service`)
This resource represents an **innovative serverless way to deploy code in a managed environment**. 
Instead of provisioning underlying infrastructure, you focus solely on your code and runtime configuration. The platform manages the execution, scaling, and lifecycle of the application.

### 2. CoderForge Machine (`coderforge_machine`)
This is a **generic object designed to represent either Virtual Machines or Physical (Bare Metal) machines**, controlled by the `virtual` flag.
* **Virtual Machine**: Set `virtual = true`.
* **Physical/Bare Metal**: Set `virtual = false`.

You have full control to define the **machine's characteristics**, such as CPU cores, RAM allocation, and logging groups, making it suitable for traditional infrastructure requirements.

---

## Usage Examples

Below are common configurations for getting started quickly.

### Provider Configuration

```hcl
provider "coderforge" {
  host_url    = "http://localhost:8080"
  token       = "your-auth-token"
  cloud_space = "production"
}
```

### Creating a Service (Managed Serverless)

Define the runtime, command, and artifact source (Image or Zip).

```hcl
resource "coderforge_service" "webapp" {
  name = "frontend-app"
  cpu  = 2
  ram  = "1024MB"
  
  # Default is true, but can be set explicitly
  virtual = true 

  code = {
    runtime     = "nodejs"
    execute_cmd = "npm start"
    # Mutually exclusive: use image_uri OR zip_file
    image_uri   = "node:18-alpine"
  }

  tags = {
    Environment = "Staging"
    Project     = "Alpha"
  }
}
```

### Creating a Machine (Virtual or Physical)

Define the characteristics of your compute instance.

```hcl
resource "coderforge_machine" "database" {
  name       = "postgres-primary"
  
  # Define machine characteristics
  cpu        = 4
  ram        = "8GB"
  
  # Set to 'true' for a Virtual Machine, or 'false' for Physical/Bare Metal
  virtual    = true
  
  logs_group = "db-logs"

  security_group_ids = ["sg-internal", "sg-ssh-access"]
}
```

### Using Data Sources

Read information about legacy resources (created manually or by other tools) using their ID.

```hcl
# Read a machine by its ID
data "coderforge_machine" "legacy_db" {
  id = "existing-vm-id-123"
}

# Use the retrieved data
output "legacy_cpu_count" {
  value = data.coderforge_machine.legacy_db.cpu
}
```

---

## Directory Structure

The examples are organized by category and resource type:

* **provider/**
    * Contains the example configuration for the provider index page (`provider.tf`).

* **data-sources/**
    * **service/**: Examples for the `coderforge_service` data source.
    * **machine/**: Examples for the `coderforge_machine` data source.

* **resources/**
    * **service/**: Examples for the `coderforge_service` resource.
    * **machine/**: Examples for the `coderforge_machine` resource.

## Documentation Generation

The document generation tool (`tfplugindocs`) expects specific filenames to generate the documentation correctly. All other `*.tf` files are ignored by the tool but can be used for local testing.

* **provider/provider.tf**: Example for the provider index page.
* **data-sources/`<name>`/data-source.tf**: Example for the named data source page.
* **resources/`<name>`/resource.tf**: Example for the named resource page.

## How to Run Locally

You can use these examples to test the provider against your local development server.

1.  **Start the Backend**: Ensure your Java API server is running (usually via `run_server.bat` on port 8080).
2.  **Build the Provider**: Ensure the latest version of the provider is installed locally (via `build_provider.bat`).
3.  **Run the Example**:
    * Navigate to the specific example directory (e.g., `examples/data-sources/service`).
    * Run the provided helper script: `test_local.bat`.
    * Alternatively, run standard Terraform commands:
        ```bash
        terraform init
        terraform apply
        ```
