package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "coderforge" {
  host_url    = "http://localhost:8080"
  token       = "mock-token"
  cloud_space = "dev"
}

resource "coderforge_service" "test" {
  name = "ds-test-service"
  cpu  = 1
  ram  = "1GB"
  code = {
      runtime     = "go"
      execute_cmd = "go run ."
      image_uri   = "golang:alpine" # CAMPO AGGIUNTO PER VALIDAZIONE
  }
}

data "coderforge_service" "ds" {
  id = coderforge_service.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.coderforge_service.ds", "name",
						"coderforge_service.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.coderforge_service.ds", "ram",
						"coderforge_service.test", "ram",
					),
				),
			},
		},
	})
}
