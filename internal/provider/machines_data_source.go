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
	_ datasource.DataSource              = &machinesDataSource{}
	_ datasource.DataSourceWithConfigure = &machinesDataSource{}
)

func NewMachinesDataSource() datasource.DataSource {
	return &machinesDataSource{}
}

type machinesDataSource struct {
	client *client.Client
}

type machinesDataSourceModel struct {
	State        types.String     `tfsdk:"state"`
	IncludeOwned types.Bool       `tfsdk:"include_owned"`
	Machines     []machineSummary `tfsdk:"machines"`
	Names        types.List       `tfsdk:"names"`
}

type machineSummary struct {
	Name      types.String `tfsdk:"name"`
	RID       types.String `tfsdk:"rid"`
	State     types.String `tfsdk:"state"`
	RawStatus types.String `tfsdk:"raw_status"`
	ManagedBy types.String `tfsdk:"managed_by"`
}

func (d *machinesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machines"
}

func (d *machinesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Every machine the token can see.\n\n" +
			"This is the list endpoint, which reports identity and state but not hardware; use the " +
			"`coderforge_machine` data source for the details of one machine.",
		Attributes: map[string]schema.Attribute{
			"state": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Return only machines in this state, such as `running`.",
			},
			"include_owned": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Include machines owned by another resource, such as container " +
					"service nodes. Defaults to `false`, since those must not be managed directly.",
			},
			"names": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Names of the matching machines, convenient for `for_each`.",
			},
			"machines": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The matching machines.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":       schema.StringAttribute{Computed: true},
						"rid":        schema.StringAttribute{Computed: true},
						"state":      schema.StringAttribute{Computed: true},
						"raw_status": schema.StringAttribute{Computed: true},
						"managed_by": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *machinesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *machinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config machinesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	machines, err := d.client.ListMachines(ctx)
	if err != nil {
		addAPIError(&resp.Diagnostics, "list", "machines", err)
		return
	}

	wantState := config.State.ValueString()
	includeOwned := config.IncludeOwned.ValueBool()

	summaries := make([]machineSummary, 0, len(machines))
	names := make([]string, 0, len(machines))

	for _, machine := range machines {
		if wantState != "" && machine.State != wantState {
			continue
		}
		if !includeOwned && machine.ManagedBy != nil && *machine.ManagedBy != "" {
			continue
		}
		summaries = append(summaries, machineSummary{
			Name:      types.StringValue(machine.Name),
			RID:       stringValue(machine.RID),
			State:     types.StringValue(machine.State),
			RawStatus: types.StringValue(machine.RawStatus),
			ManagedBy: stringValue(machine.ManagedBy),
		})
		names = append(names, machine.Name)
	}

	nameList, diags := stringListValue(ctx, names)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.Machines = summaries
	config.Names = nameList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
