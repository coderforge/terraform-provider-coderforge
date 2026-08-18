# The templates this platform offers. Reading the catalogue beats hard-coding a
# filename that may be retired.
data "coderforge_vm_templates" "available" {}

output "template_files" {
  value = data.coderforge_vm_templates.available.files
}

resource "coderforge_machine" "web" {
  name     = "web01"
  cpu      = 2
  ram      = 4096
  template = data.coderforge_vm_templates.available.files[0]
}
