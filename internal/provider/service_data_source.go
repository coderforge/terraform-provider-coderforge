package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &serviceDataSource{}
	_ datasource.DataSourceWithConfigure = &serviceDataSource{}
)

func NewServiceDataSource() datasource.DataSource {
	return &serviceDataSource{}
}

type serviceDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Virtual types.Bool   `tfsdk:"virtual"`
	Cpu     types.Int64  `tfsdk:"cpu"`
	Ram     types.String `tfsdk:"ram"`
}

type serviceDataSource struct {
	client *Client
}

func (d *serviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *serviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"virtual": schema.BoolAttribute{
				Computed: true,
			},
			"cpu": schema.Int64Attribute{
				Computed: true,
			},
			"ram": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *serviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state serviceDataSourceModel
	req.Config.Get(ctx, &state)

	res, err := d.client.GetResource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service data source", err.Error())
		return
	}

	state.Name = types.StringValue(res.Name)
	state.Virtual = types.BoolValue(res.Virtual)
	state.Cpu = types.Int64Value(res.Cpu)
	state.Ram = types.StringValue(res.Ram)

	resp.State.Set(ctx, &state)
}

func (d *serviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *provider.Client, got: %T", req.ProviderData))
		return
	}
	d.client = client
}
