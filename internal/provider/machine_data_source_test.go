package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMachineDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "coderforge" {
  host_url    = "http://127.0.0.1:8080"
  token       = "mock-token"
  cloud_space = "dev"
}

resource "coderforge_machine" "test_vm" {
  name         = "ds-test-machine"
  cpu          = 4
  ram          = "8GB"
  logs_group   = "machine-logs"
  virtual      = true
  
  tags = {
    env = "datasource-test"
  }
}

data "coderforge_machine" "ds_vm" {
  id = coderforge_machine.test_vm.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verifica che il data source legga gli stessi valori della risorsa
					resource.TestCheckResourceAttrPair(
						"data.coderforge_machine.ds_vm", "name",
						"coderforge_machine.test_vm", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.coderforge_machine.ds_vm", "cpu",
						"coderforge_machine.test_vm", "cpu",
					),
					resource.TestCheckResourceAttrPair(
						"data.coderforge_machine.ds_vm", "virtual",
						"coderforge_machine.test_vm", "virtual",
					),
					resource.TestCheckResourceAttr("data.coderforge_machine.ds_vm", "tags.env", "datasource-test"),
				),
			},
		},
	})
}
