package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &machineDataSource{}
	_ datasource.DataSourceWithConfigure = &machineDataSource{}
)

func NewMachineDataSource() datasource.DataSource {
	return &machineDataSource{}
}

type machineDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Cpu              types.Int64  `tfsdk:"cpu"`
	Ram              types.String `tfsdk:"ram"`
	Timeout          types.Int64  `tfsdk:"timeout"`
	LogsGroup        types.String `tfsdk:"logs_group"`
	Virtual          types.Bool   `tfsdk:"virtual"`
	SecurityGroupIds types.List   `tfsdk:"security_group_ids"`
	Tags             types.Map    `tfsdk:"tags"`
}

type machineDataSource struct {
	client *Client
}

func (d *machineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (d *machineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"cpu": schema.Int64Attribute{
				Computed: true,
			},
			"ram": schema.StringAttribute{
				Computed: true,
			},
			"timeout": schema.Int64Attribute{
				Computed: true,
			},
			"logs_group": schema.StringAttribute{
				Computed: true,
			},
			"virtual": schema.BoolAttribute{
				Computed: true,
			},
			"security_group_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *machineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state machineDataSourceModel
	req.Config.Get(ctx, &state)

	res, err := d.client.GetResource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading machine data source", err.Error())
		return
	}

	state.Name = types.StringValue(res.Name)
	state.Cpu = types.Int64Value(res.Cpu)
	state.Ram = types.StringValue(res.Ram)
	state.Timeout = types.Int64Value(res.Timeout)
	state.LogsGroup = types.StringValue(res.LogsGroup)
	state.Virtual = types.BoolValue(res.Virtual)

	sgList, diags := types.ListValueFrom(ctx, types.StringType, res.SecurityGroupIds)
	resp.Diagnostics.Append(diags...)
	state.SecurityGroupIds = sgList

	tagsMap, diags := types.MapValueFrom(ctx, types.StringType, res.Tags)
	resp.Diagnostics.Append(diags...)
	state.Tags = tagsMap

	resp.State.Set(ctx, &state)
}

func (d *machineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
