resource "coderforge_machine" "web" {
  name = "web01"
  cpu  = 2
  ram  = 4096
}

# Forward one host port to the machine.
resource "coderforge_network_rule" "http" {
  src_port  = "8080"
  dest_ip   = coderforge_machine.web.ip
  dest_port = "80"
  protocol  = "tcp"
}

# A range. Both sides must be the same width.
resource "coderforge_network_rule" "game_servers" {
  src_port  = "50000-50100"
  dest_ip   = coderforge_machine.web.ip
  dest_port = "50000-50100"
  protocol  = "udp"
}
