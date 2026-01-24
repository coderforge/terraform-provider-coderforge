package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Mappa dei provider per il framework di test
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"coderforge": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Qui potresti verificare se il server Java è attivo,
	// o se le variabili d'ambiente (CODERFORGE_TOKEN) sono settate.
}
