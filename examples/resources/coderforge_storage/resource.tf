# A detached disk, ready to be attached later or handed to a container service.
resource "coderforge_storage" "archive" {
  name    = "archive"
  size_gb = 200
}

# A disk attached to a machine on SATA port 1. Moving it to another machine is
# an in-place detach and reattach; the data survives.
resource "coderforge_storage" "database" {
  name        = "database-data"
  size_gb     = 100
  attached_to = coderforge_machine.db.name
  port        = 1
}

resource "coderforge_machine" "db" {
  name = "db01"
  cpu  = 4
  ram  = 8192
}
