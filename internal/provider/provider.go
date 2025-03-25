package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &coderforgeProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &coderforgeProvider{
			version: version,
		}
	}
}

// coderforgeProviderModel maps provider schema data to a Go type.
type coderforgeProviderModel struct {
	Token      types.String   `tfsdk:"token"`
	CloudSpace types.String   `tfsdk:"cloud_space"`
	Locations  []types.String `tfsdk:"locations"`
	StackId    types.String   `tfsdk:"stack_id"`
}

// coderforgeProvider is the provider implementation.
type coderforgeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *coderforgeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "coderforge"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *coderforgeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"cloud_space": schema.StringAttribute{
				Required: true,
			},
			"locations": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"stack_id": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (p *coderforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring CoderForge.org client")

	// Retrieve provider data from configuration
	var config coderforgeProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var token string

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	} else {
		token = os.Getenv("CODERFORGE_CLOUD_TOKEN")
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing CoderForge.org API API Password",
			"The provider cannot create the CoderForge.org API API client as there is a missing or empty value for the CoderForge.org API token. "+
				"Set the token value in the configuration or use the CODERFORGE_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if config.CloudSpace.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("cloud_space"),
			"Missing CoderForge.org API API cloud_space",
			"The provider cannot create the CoderForge.org API API client as there is a missing or empty value for the CoderForge.org API cloud_space. "+
				"Set the cloud_space inside the provider.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var cloudSpace = config.CloudSpace.ValueString()
	var stackId = config.StackId.ValueString()

	ctx = tflog.SetField(ctx, "coderforge_cloud_token", token)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "coderforge_password")

	tflog.Debug(ctx, "Creating CoderForge.org client")

	var locations []string
	for _, location := range config.Locations {
		locations = append(locations, location.ValueString())
	}

	// Create a new CoderForge.org client using the configuration values
	client, err := NewClient(&token, &cloudSpace, &locations, &stackId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create CoderForge.org API Client",
			"An unexpected error occurred when creating the CoderForge.org API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"CoderForge.org Client Error: "+err.Error(),
		)
		return
	}

	// Make the CoderForge.org client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured CoderForge.org client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *coderforgeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Resources defines the resources implemented in the provider.
func (p *coderforgeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFunctionResource,
	}
}
