// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the provider into the acceptance test
// framework. Acceptance tests talk to a real terraform-api and are gated on
// TF_ACC, so `go test ./...` stays a unit-test run.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"coderforge": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	// Acceptance tests create real machines, which cost real minutes. Failing
	// early with a clear message beats a run that gets halfway and times out.
	for _, key := range []string{envEndpoint, envToken} {
		if os.Getenv(key) == "" {
			t.Fatalf("%s must be set for acceptance tests", key)
		}
	}
}

func TestProviderSchemaIsValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &provider.SchemaResponse{}
	New("test")().Schema(ctx, provider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() reported: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("the provider schema is invalid: %v", diags)
	}
}

// TestResourceSchemasAreValid runs the framework's own implementation checks
// over every resource. They catch the schema mistakes that would otherwise
// only appear when a practitioner writes the configuration that trips them -
// a Computed attribute with a Default, a nested block with no attributes.
func TestResourceSchemasAreValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for name, factory := range map[string]func() resource.Resource{
		"coderforge_machine":           NewMachineResource,
		"coderforge_container_service": NewContainerServiceResource,
		"coderforge_storage":           NewStorageResource,
		"coderforge_param":             NewParamResource,
		"coderforge_secret":            NewSecretResource,
		"coderforge_network_rule":      NewNetworkRuleResource,
	} {
		t.Run(name, func(t *testing.T) {
			resp := &resource.SchemaResponse{}
			factory().Schema(ctx, resource.SchemaRequest{}, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("Schema() reported: %v", resp.Diagnostics)
			}
			if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
				t.Fatalf("the schema is invalid: %v", diags)
			}
		})
	}
}

func TestDataSourceSchemasAreValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for name, factory := range map[string]func() datasource.DataSource{
		"coderforge_machine":            NewMachineDataSource,
		"coderforge_machines":           NewMachinesDataSource,
		"coderforge_container_service":  NewContainerServiceDataSource,
		"coderforge_storage":            NewStorageDataSource,
		"coderforge_param":              NewParamDataSource,
		"coderforge_secret":             NewSecretDataSource,
		"coderforge_vm_templates":       NewVMTemplatesDataSource,
		"coderforge_container_networks": NewContainerNetworksDataSource,
	} {
		t.Run(name, func(t *testing.T) {
			resp := &datasource.SchemaResponse{}
			factory().Schema(ctx, datasource.SchemaRequest{}, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("Schema() reported: %v", resp.Diagnostics)
			}
			if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
				t.Fatalf("the schema is invalid: %v", diags)
			}
		})
	}
}

// TestEveryResourceDocumentsItself guards the registry documentation quality
// gate: an attribute with no description renders as a blank cell on the
// provider's registry page.
func TestEveryResourceDocumentsItself(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for name, factory := range map[string]func() resource.Resource{
		"coderforge_machine":           NewMachineResource,
		"coderforge_container_service": NewContainerServiceResource,
		"coderforge_storage":           NewStorageResource,
		"coderforge_param":             NewParamResource,
		"coderforge_secret":            NewSecretResource,
		"coderforge_network_rule":      NewNetworkRuleResource,
	} {
		t.Run(name, func(t *testing.T) {
			resp := &resource.SchemaResponse{}
			factory().Schema(ctx, resource.SchemaRequest{}, resp)

			if resp.Schema.MarkdownDescription == "" {
				t.Error("the resource itself has no description")
			}
			for attrName, attr := range resp.Schema.Attributes {
				// `timeouts` is documented by the framework that supplies it.
				if attrName == "timeouts" {
					continue
				}
				if attr.GetMarkdownDescription() == "" && attr.GetDescription() == "" {
					t.Errorf("attribute %q has no description", attrName)
				}
			}
		})
	}
}
