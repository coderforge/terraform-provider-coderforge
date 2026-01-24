package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMachineResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Creazione
			{
				Config: testAccMachineConfig("vm-test", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coderforge_machine.vm", "name", "vm-test"),
					resource.TestCheckResourceAttr("coderforge_machine.vm", "virtual", "true"),
					resource.TestCheckResourceAttr("coderforge_machine.vm", "logs_group", "var-logs"),
				),
			},
			// Update: Cambia virtual a false
			{
				Config: testAccMachineConfig("vm-test", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coderforge_machine.vm", "virtual", "false"),
				),
			},
		},
	})
}

func testAccMachineConfig(name string, virtual bool) string {
	return fmt.Sprintf(`
provider "coderforge" {
  host_url    = "http://localhost:8080"
  token       = "mock-token"
  cloud_space = "dev"
}

resource "coderforge_machine" "vm" {
  name         = "%s"
  cpu          = 2
  ram          = "4GB"
  logs_group   = "var-logs"
  virtual      = %t
  
  tags = {
    env = "test"
  }
}
`, name, virtual)
}
