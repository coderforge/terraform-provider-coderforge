# A machine on the host-only network, kept running.
resource "coderforge_machine" "web" {
  name     = "web01"
  cpu      = 2
  ram      = 4096
  network  = "hostonly"
  template = "ubuntu-24.04-server.vdi"
}

# Two adapters: host-only for management, NAT for outbound access.
resource "coderforge_machine" "build" {
  name    = "build01"
  cpu     = 8
  ram     = 16384
  network = "hostonly,nat"

  # Creating a machine boots a full guest image. Raise the ceiling for a
  # template that takes longer than usual to come up.
  timeouts {
    create = "45m"
  }
}

# A machine that exists but is normally powered off. Terraform corrects drift,
# so one started by hand is stopped again on the next apply.
resource "coderforge_machine" "standby" {
  name          = "standby01"
  cpu           = 1
  ram           = 2048
  desired_state = "stopped"
}

output "web_address" {
  value = coderforge_machine.web.ip
}

output "web_password" {
  # Generated at creation and returned only once, so it lives in state.
  value     = coderforge_machine.web.password
  sensitive = true
}
