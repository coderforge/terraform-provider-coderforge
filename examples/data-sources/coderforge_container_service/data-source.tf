# By name...
data "coderforge_container_service" "edge" {
  name = "edge-proxy"
}

# ...or by the platform's id, which is unambiguous.
data "coderforge_container_service" "api" {
  id = "0002"
}

# Forward a host port to the first node of the service.
resource "coderforge_network_rule" "edge_http" {
  src_port  = "80"
  dest_ip   = data.coderforge_container_service.edge.node_state[0].ip
  dest_port = "80"
}
