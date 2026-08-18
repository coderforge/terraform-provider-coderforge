# The id is derived from the rule itself:
#   <protocol>_<src_port>_<dest_ip>_<dest_port>
terraform import coderforge_network_rule.http tcp_8080_192.168.56.10_80
