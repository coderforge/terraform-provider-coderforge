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
	ID      types.String      `tfsdk:"id"`
	Name    types.String      `tfsdk:"name"`
	Code    *serviceCodeModel `tfsdk:"code"`
	Timeout types.Int64       `tfsdk:"timeout"`
	Cpu     types.Int64       `tfsdk:"cpu"`
	Ram     types.String      `tfsdk:"ram"`
	Virtual types.Bool        `tfsdk:"virtual"`
	// ServiceType rimosso
	LogsGroup        types.String `tfsdk:"logs_group"`
	SecurityGroupIds types.List   `tfsdk:"security_group_ids"`
	Tags             types.Map    `tfsdk:"tags"`
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
			"code": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"runtime": schema.StringAttribute{
						Computed: true,
					},
					"execute_cmd": schema.StringAttribute{
						Computed: true,
					},
					"zip_file": schema.StringAttribute{
						Computed: true,
					},
					"image_uri": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"timeout": schema.Int64Attribute{
				Computed: true,
			},
			"cpu": schema.Int64Attribute{
				Computed: true,
			},
			"ram": schema.StringAttribute{
				Computed: true,
			},
			"virtual": schema.BoolAttribute{
				Computed: true,
			},
			// service_type rimosso dallo schema
			"logs_group": schema.StringAttribute{
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

func (d *serviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state serviceDataSourceModel
	req.Config.Get(ctx, &state)

	res, err := d.client.GetResource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service data source", err.Error())
		return
	}

	state.Name = types.StringValue(res.Name)
	state.Timeout = types.Int64Value(res.Timeout)
	state.Cpu = types.Int64Value(res.Cpu)
	state.Ram = types.StringValue(res.Ram)
	state.Virtual = types.BoolValue(res.Virtual)
	// state.ServiceType rimosso
	state.LogsGroup = types.StringValue(res.LogsGroup)

	// Mappiamo la struttura Code se presente
	if res.Code != nil {
		state.Code = &serviceCodeModel{
			Runtime:    types.StringValue(res.Code.Runtime),
			ExecuteCmd: types.StringValue(res.Code.ExecuteCmd),
			ImageUri:   types.StringValue(res.Code.ImageUri),
			ZipFile:    types.StringValue(res.Code.ZipFile),
		}
	}

	sgList, diags := types.ListValueFrom(ctx, types.StringType, res.SecurityGroupIds)
	resp.Diagnostics.Append(diags...)
	state.SecurityGroupIds = sgList

	tagsMap, diags := types.MapValueFrom(ctx, types.StringType, res.Tags)
	resp.Diagnostics.Append(diags...)
	state.Tags = tagsMap

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
