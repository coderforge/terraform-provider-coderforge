package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccServiceResourceConfig("test-service", "256MB"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coderforge_service.test", "name", "test-service"),
					resource.TestCheckResourceAttr("coderforge_service.test", "ram", "256MB"),
					resource.TestCheckResourceAttr("coderforge_service.test", "cpu", "1"),
					resource.TestCheckResourceAttr("coderforge_service.test", "virtual", "true"), // Verifica default
					resource.TestCheckResourceAttrSet("coderforge_service.test", "id"),
				),
			},
			// Step 2: Update (Cambia RAM e CPU)
			{
				Config: testAccServiceResourceConfig("test-service", "512MB"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coderforge_service.test", "ram", "512MB"),
					resource.TestCheckResourceAttr("coderforge_service.test", "virtual", "true"),
				),
			},
			// Step 3: Import
			{
				ResourceName:      "coderforge_service.test",
				ImportState:       true,
				ImportStateVerify: true,
				// FIX: Ignoriamo last_updated perché non viene persistito/restituito dall'API
				ImportStateVerifyIgnore: []string{"code.zip_file", "last_updated"},
			},
		},
	})
}

// Funzione helper per generare l'HCL
func testAccServiceResourceConfig(name, ram string) string {
	return fmt.Sprintf(`
provider "coderforge" {
  host_url    = "http://localhost:8080"
  token       = "mock-token"
  cloud_space = "dev"
}

resource "coderforge_service" "test" {
  name = "%s"
  cpu  = 1
  ram  = "%s"

  code = {
    runtime     = "nodejs"
    execute_cmd = "npm start"
    image_uri   = "node:14-alpine"
  }
}
`, name, ram)
}
