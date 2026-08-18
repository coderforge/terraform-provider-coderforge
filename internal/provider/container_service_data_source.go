// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-coderforge/internal/client"
)

var (
	_ datasource.DataSource                     = &containerServiceDataSource{}
	_ datasource.DataSourceWithConfigure        = &containerServiceDataSource{}
	_ datasource.DataSourceWithConfigValidators = &containerServiceDataSource{}
)

func NewContainerServiceDataSource() datasource.DataSource {
	return &containerServiceDataSource{}
}

type containerServiceDataSource struct {
	client *client.Client
}

type containerServiceDataSourceModel struct {
	ID           types.String  `tfsdk:"id"`
	Name         types.String  `tfsdk:"name"`
	RID          types.String  `tfsdk:"rid"`
	State        types.String  `tfsdk:"state"`
	RawStatus    types.String  `tfsdk:"raw_status"`
	DesiredState types.String  `tfsdk:"desired_state"`
	Nodes        types.Int64   `tfsdk:"nodes"`
	Version      types.Int64   `tfsdk:"version"`
	Provisioner  types.String  `tfsdk:"provisioner_name"`
	EnvParamRID  types.String  `tfsdk:"env_param_rid"`
	EnvSecretRID types.String  `tfsdk:"env_secret_rid"`
	NetworkRID   types.String  `tfsdk:"network_rid"`
	Containers   types.List    `tfsdk:"containers"`
	NodeState    []nodeSummary `tfsdk:"node_state"`
	ManagedBy    types.String  `tfsdk:"managed_by"`
	Message      types.String  `tfsdk:"message"`
	CreatedAt    types.String  `tfsdk:"created_at"`
	UpdatedAt    types.String  `tfsdk:"updated_at"`
}

type nodeSummary struct {
	Index   types.Int64  `tfsdk:"index"`
	VMName  types.String `tfsdk:"vm_name"`
	IP      types.String `tfsdk:"ip"`
	Status  types.String `tfsdk:"status"`
	Message types.String `tfsdk:"message"`
}

func (d *containerServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_service"
}

func (d *containerServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a container service by id or by name, whether or not Terraform " +
			"manages it. Its `node_state` gives the addresses of the machines behind it, which is " +
			"what a port forwarding rule needs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Four-digit service id. Specify this or `name`.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Service name. Specify this or `id`.",
			},
			"rid":           schema.StringAttribute{Computed: true},
			"state":         schema.StringAttribute{Computed: true, MarkdownDescription: "Normalised state."},
			"raw_status":    schema.StringAttribute{Computed: true},
			"desired_state": schema.StringAttribute{Computed: true},
			"nodes":         schema.Int64Attribute{Computed: true},
			"version":       schema.Int64Attribute{Computed: true},
			// `provisioner` is reserved by Terraform; see the resource schema.
			"provisioner_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Provisioner script run on each node."},
			"env_param_rid":    schema.StringAttribute{Computed: true},
			"env_secret_rid":   schema.StringAttribute{Computed: true},
			"network_rid":      schema.StringAttribute{Computed: true},
			"managed_by":       schema.StringAttribute{Computed: true},
			"message":          schema.StringAttribute{Computed: true},
			"created_at":       schema.StringAttribute{Computed: true},
			"updated_at":       schema.StringAttribute{Computed: true},
			"containers": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Service names declared by the compose file.",
			},
			"node_state": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The machines backing this service.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"index":   schema.Int64Attribute{Computed: true},
						"vm_name": schema.StringAttribute{Computed: true},
						"ip":      schema.StringAttribute{Computed: true},
						"status":  schema.StringAttribute{Computed: true},
						"message": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *containerServiceDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

func (d *containerServiceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *containerServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config containerServiceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var service *client.ContainerService

	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		found, err := d.client.GetContainerService(ctx, config.ID.ValueString())
		if err != nil {
			addAPIError(&resp.Diagnostics, "read", "container service "+config.ID.ValueString(), err)
			return
		}
		service = found
	} else {
		name := config.Name.ValueString()
		services, err := d.client.ListContainerServices(ctx, name)
		if err != nil {
			addAPIError(&resp.Diagnostics, "list", "container services", err)
			return
		}
		switch len(services) {
		case 0:
			resp.Diagnostics.AddError(
				"No container service found",
				fmt.Sprintf("No container service is named %q.", name),
			)
			return
		case 1:
			service = &services[0]
		default:
			// Names are not unique upstream, so an ambiguous lookup has to
			// fail rather than pick one and produce a config that works today
			// and silently points elsewhere tomorrow.
			resp.Diagnostics.AddError(
				"Multiple container services found",
				fmt.Sprintf("%d services are named %q. Use `id` to identify the one you mean.",
					len(services), name),
			)
			return
		}
	}

	config.ID = types.StringValue(service.ID)
	config.Name = types.StringValue(service.Name)
	config.RID = stringValue(service.RID)
	config.State = types.StringValue(service.State)
	config.RawStatus = types.StringValue(service.RawStatus)
	config.DesiredState = types.StringValue(service.DesiredState)
	config.Nodes = types.Int64Value(service.Nodes)
	config.Version = int64Value(service.Version)
	config.Provisioner = stringValue(service.Provisioner)
	config.EnvParamRID = stringValue(service.EnvParamRID)
	config.EnvSecretRID = stringValue(service.EnvSecretRID)
	config.NetworkRID = stringValue(service.NetworkRID)
	config.ManagedBy = stringValue(service.ManagedBy)
	config.Message = types.StringValue(service.Message)
	config.CreatedAt = stringValue(service.CreatedAt)
	config.UpdatedAt = stringValue(service.UpdatedAt)

	containers, diags := stringListValue(ctx, service.Containers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Containers = containers

	nodes := make([]nodeSummary, 0, len(service.NodeState))
	for _, node := range service.NodeState {
		nodes = append(nodes, nodeSummary{
			Index:   types.Int64Value(node.Index),
			VMName:  types.StringValue(node.VMName),
			IP:      stringValue(node.IP),
			Status:  types.StringValue(node.Status),
			Message: types.StringValue(node.Message),
		})
	}
	config.NodeState = nodes

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
