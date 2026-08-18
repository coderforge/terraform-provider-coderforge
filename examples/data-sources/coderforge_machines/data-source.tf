# Every machine Terraform is allowed to see. Machines owned by a container
# service are excluded unless include_owned is set, since those must not be
# managed directly.
data "coderforge_machines" "all" {}

# Only the ones that are up.
data "coderforge_machines" "running" {
  state = "running"
}

output "machine_names" {
  value = data.coderforge_machines.all.names
}

output "running_count" {
  value = length(data.coderforge_machines.running.machines)
}
