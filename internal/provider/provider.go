package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ provider.Provider = &coderforgeProvider{}
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &coderforgeProvider{
			version: version,
		}
	}
}

type coderforgeProviderModel struct {
	Token      types.String   `tfsdk:"token"`
	CloudSpace types.String   `tfsdk:"cloud_space"`
	Locations  []types.String `tfsdk:"locations"`
	StackId    types.String   `tfsdk:"stack_id"`
	HostURL    types.String   `tfsdk:"host_url"`
}

type coderforgeProvider struct {
	version string
}

func (p *coderforgeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "coderforge"
	resp.Version = p.version
}

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
			"host_url": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (p *coderforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring CoderForge.org client")

	var config coderforgeProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var token string
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	} else {
		token = os.Getenv("CODERFORGE_CLOUD_TOKEN")
		if token == "" {
			token = os.Getenv("CODERFORGE_TOKEN")
		}
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing CoderForge.org API token",
			"The provider cannot create the CoderForge.org API client because the API token is missing.",
		)
	}

	if config.CloudSpace.IsNull() || config.CloudSpace.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("cloud_space"),
			"Missing CoderForge.org cloud_space",
			"The provider cannot create the CoderForge.org API client because cloud_space is missing.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var cloudSpace = config.CloudSpace.ValueString()
	var stackId = config.StackId.ValueString()
	var hostURLStr string
	if !config.HostURL.IsNull() && !config.HostURL.IsUnknown() {
		hostURLStr = config.HostURL.ValueString()
	}
	if hostURLStr == "" {
		if v := os.Getenv("CODERFORGE_API_URL"); v != "" {
			hostURLStr = v
		} else if v := os.Getenv("CODERFORGE_HOST"); v != "" {
			hostURLStr = v
		}
	}

	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "token", "coderforge_token", "coderforge_cloud_token")

	var locations []string
	for _, location := range config.Locations {
		locations = append(locations, location.ValueString())
	}

	var hostOverride *string
	if hostURLStr != "" {
		hostOverride = &hostURLStr
	}
	client, err := NewClient(&token, &cloudSpace, &locations, &stackId, hostOverride)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create CoderForge.org API Client",
			"An unexpected error occurred when creating the CoderForge.org API client: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *coderforgeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	// PRIMA ERA: return []func() datasource.DataSource{}
	// DEVE ESSERE:
	return []func() datasource.DataSource{
		NewServiceDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *coderforgeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServiceResource,
		NewMachineResource,
	}
}
