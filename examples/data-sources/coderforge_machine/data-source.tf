# Look up a machine someone else created, to point a rule at it.
data "coderforge_machine" "legacy_db" {
  name = "db-legacy"
}

resource "coderforge_network_rule" "postgres" {
  src_port  = "15432"
  dest_ip   = data.coderforge_machine.legacy_db.ip
  dest_port = "5432"
}

output "legacy_db_state" {
  value = data.coderforge_machine.legacy_db.state
}
