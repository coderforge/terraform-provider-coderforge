terraform {
  required_providers {
    coderforge = {
      source  = "registry.terraform.io/coderforge/coderforge"
      version = "1.1.0"
    }
  }
}

provider "coderforge" {
  host_url    = "http://localhost:8080"
  token       = "mock-token"
  cloud_space = "dev"
}

resource "coderforge_machine" "my_test_machine" {
  name = "local-test-machine"
  cpu          = 2
  ram          = "512MB"
  timeout      = 300
  logs_group   = "test-logs"
  virtual      = true
  tags = {
    Environment = "LocalTest"
    Owner       = "Developer"
  }
  security_group_ids = ["sg-12345", "sg-67890"]
}

output "machine_id" {
  value = coderforge_machine.my_test_machine.id
}