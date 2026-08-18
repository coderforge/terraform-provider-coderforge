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

// Catalogue data sources: what the platform offers, rather than what exists on
// it. Both answer the question "what may I put in this attribute", which is
// otherwise a trip to the console and a copied string.

// --- VM templates ---

var (
	_ datasource.DataSource              = &vmTemplatesDataSource{}
	_ datasource.DataSourceWithConfigure = &vmTemplatesDataSource{}
)

func NewVMTemplatesDataSource() datasource.DataSource {
	return &vmTemplatesDataSource{}
}

type vmTemplatesDataSource struct {
	client *client.Client
}

type vmTemplatesDataSourceModel struct {
	Templates []templateSummary `tfsdk:"templates"`
	Files     types.List        `tfsdk:"files"`
}

type templateSummary struct {
	File        types.String `tfsdk:"file"`
	Description types.String `tfsdk:"description"`
	Logo        types.String `tfsdk:"logo"`
}

func (d *vmTemplatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_templates"
}

func (d *vmTemplatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Templates available for the `template` attribute of " +
			"`coderforge_machine`. The platform rejects a template it does not know, so reading the " +
			"catalogue beats hard-coding a filename that may be retired.",
		Attributes: map[string]schema.Attribute{
			"files": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Template filenames, which is what `template` expects.",
			},
			"templates": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"file":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"logo":        schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *vmTemplatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *vmTemplatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config vmTemplatesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templates, err := d.client.ListTemplates(ctx)
	if err != nil {
		addAPIError(&resp.Diagnostics, "list", "VM templates", err)
		return
	}

	summaries := make([]templateSummary, 0, len(templates))
	files := make([]string, 0, len(templates))
	for _, template := range templates {
		summaries = append(summaries, templateSummary{
			File:        types.StringValue(template.File),
			Description: types.StringValue(template.Description),
			Logo:        types.StringValue(template.Logo),
		})
		files = append(files, template.File)
	}

	fileList, diags := stringListValue(ctx, files)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.Templates = summaries
	config.Files = fileList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// --- Container networks ---

var (
	_ datasource.DataSource              = &containerNetworksDataSource{}
	_ datasource.DataSourceWithConfigure = &containerNetworksDataSource{}
)

func NewContainerNetworksDataSource() datasource.DataSource {
	return &containerNetworksDataSource{}
}

type containerNetworksDataSource struct {
	client *client.Client
}

type containerNetworksDataSourceModel struct {
	Name     types.String     `tfsdk:"name"`
	Networks []networkSummary `tfsdk:"networks"`
}

type networkSummary struct {
	ID        types.String `tfsdk:"id"`
	RID       types.String `tfsdk:"rid"`
	Name      types.String `tfsdk:"name"`
	Driver    types.String `tfsdk:"driver"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *containerNetworksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_networks"
}

func (d *containerNetworksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Swarm overlay networks the platform tracks. Their `rid` is what " +
			"`coderforge_container_service.network_rid` expects, which is how two services are put " +
			"on the same network.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Return only the network with this exact name.",
			},
			"networks": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true},
						"rid":        schema.StringAttribute{Computed: true, MarkdownDescription: "Resource id, e.g. `cs_net:0001`."},
						"name":       schema.StringAttribute{Computed: true},
						"driver":     schema.StringAttribute{Computed: true},
						"created_at": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *containerNetworksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *containerNetworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config containerNetworksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networks, err := d.client.ListContainerNetworks(ctx, config.Name.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, "list", "container networks", err)
		return
	}

	summaries := make([]networkSummary, 0, len(networks))
	for _, network := range networks {
		summaries = append(summaries, networkSummary{
			ID:        types.StringValue(network.ID),
			RID:       types.StringValue(network.RID),
			Name:      types.StringValue(network.Name),
			Driver:    types.StringValue(network.Driver),
			CreatedAt: stringValue(network.CreatedAt),
		})
	}
	config.Networks = summaries

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
