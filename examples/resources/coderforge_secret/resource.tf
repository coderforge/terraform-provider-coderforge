# Secret material is write-only: the provider never reads it back, so a value
# changed in the vault by hand is not reported as drift. What is tracked is the
# set of key names.
#
# The values are still written into Terraform state. Protect the state file.
resource "coderforge_secret" "database" {
  name = "database-credentials"
  data = {
    DATABASE_USER     = "app"
    DATABASE_PASSWORD = var.database_password
  }
}

variable "database_password" {
  type      = string
  sensitive = true
}

output "database_secret_keys" {
  value = coderforge_secret.database.keys
}
