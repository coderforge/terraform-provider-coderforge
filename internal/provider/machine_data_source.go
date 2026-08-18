// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-coderforge/internal/client"
)

var (
	_ datasource.DataSource              = &machineDataSource{}
	_ datasource.DataSourceWithConfigure = &machineDataSource{}
)

func NewMachineDataSource() datasource.DataSource {
	return &machineDataSource{}
}

type machineDataSource struct {
	client *client.Client
}

type machineDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	ID           types.String `tfsdk:"id"`
	RID          types.String `tfsdk:"rid"`
	State        types.String `tfsdk:"state"`
	RawStatus    types.String `tfsdk:"raw_status"`
	DesiredState types.String `tfsdk:"desired_state"`
	RAM          types.Int64  `tfsdk:"ram"`
	CPU          types.Int64  `tfsdk:"cpu"`
	Network      types.String `tfsdk:"network"`
	IP           types.String `tfsdk:"ip"`
	Username     types.String `tfsdk:"username"`
	ManagedBy    types.String `tfsdk:"managed_by"`
	Message      types.String `tfsdk:"message"`
}

func (d *machineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (d *machineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a machine that exists in Cloud Builder, whether or not Terraform " +
			"manages it. Useful for pointing a network rule at a machine someone else created.\n\n" +
			"The generated password is not available here: Cloud Builder returns it only at creation.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the machine to look up.",
			},
			"id":            schema.StringAttribute{Computed: true, MarkdownDescription: "Same as `name`."},
			"rid":           schema.StringAttribute{Computed: true, MarkdownDescription: "Cloud Builder resource id."},
			"state":         schema.StringAttribute{Computed: true, MarkdownDescription: "Normalised state."},
			"raw_status":    schema.StringAttribute{Computed: true, MarkdownDescription: "Hypervisor status verbatim."},
			"desired_state": schema.StringAttribute{Computed: true, MarkdownDescription: "Power state the machine currently satisfies."},
			"ram":           schema.Int64Attribute{Computed: true, MarkdownDescription: "Memory in MiB."},
			"cpu":           schema.Int64Attribute{Computed: true, MarkdownDescription: "Virtual CPU count."},
			"network":       schema.StringAttribute{Computed: true, MarkdownDescription: "Configured network adapters."},
			"ip":            schema.StringAttribute{Computed: true, MarkdownDescription: "Assigned address, if any."},
			"username":      schema.StringAttribute{Computed: true, MarkdownDescription: "Default account."},
			"managed_by":    schema.StringAttribute{Computed: true, MarkdownDescription: "Owning resource, if any."},
			"message":       schema.StringAttribute{Computed: true},
		},
	}
}

func (d *machineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *machineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config machineDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	machine, err := d.client.GetMachine(ctx, config.Name.ValueString())
	if err != nil {
		// A data source pointing at nothing is an error, not silent drift: the
		// configuration asked for something that has to exist.
		addAPIError(&resp.Diagnostics, "read", "machine "+config.Name.ValueString(), err)
		return
	}

	config.ID = types.StringValue(machine.Name)
	config.RID = stringValue(machine.RID)
	config.State = types.StringValue(machine.State)
	config.RawStatus = types.StringValue(machine.RawStatus)
	config.DesiredState = types.StringValue(machine.DesiredState)
	config.RAM = int64Value(machine.RAM)
	config.CPU = int64Value(machine.CPU)
	config.Network = stringValue(machine.Network)
	config.IP = stringValue(machine.IP)
	config.Username = stringValue(machine.Username)
	config.ManagedBy = stringValue(machine.ManagedBy)
	config.Message = types.StringValue(machine.Message)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
