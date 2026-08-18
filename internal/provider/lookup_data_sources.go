// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-coderforge/internal/client"
)

// The store, parameter and secret data sources share one shape: look something
// up by id or by name and report its attributes. They live together because
// splitting three near-identical files apart would hide how alike they are.

// --- Storage ---

var (
	_ datasource.DataSource                     = &storageDataSource{}
	_ datasource.DataSourceWithConfigure        = &storageDataSource{}
	_ datasource.DataSourceWithConfigValidators = &storageDataSource{}
)

func NewStorageDataSource() datasource.DataSource {
	return &storageDataSource{}
}

type storageDataSource struct {
	client *client.Client
}

type storageDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	SizeGB     types.Int64  `tfsdk:"size_gb"`
	Path       types.String `tfsdk:"path"`
	AttachedTo types.String `tfsdk:"attached_to"`
	Port       types.Int64  `tfsdk:"port"`
	ManagedBy  types.String `tfsdk:"managed_by"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

func (d *storageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage"
}

func (d *storageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a disk by id or by name, to attach an existing one to a " +
			"container service or to read where it is attached.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Resource id, for example `store:0003`. Specify this or `name`.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Disk name. Specify this or `id`.",
			},
			"size_gb":     schema.Int64Attribute{Computed: true, MarkdownDescription: "Size in GiB."},
			"path":        schema.StringAttribute{Computed: true, MarkdownDescription: "Image path on the hypervisor host."},
			"attached_to": schema.StringAttribute{Computed: true, MarkdownDescription: "Machine the disk is attached to, if any."},
			"port":        schema.Int64Attribute{Computed: true, MarkdownDescription: "SATA port, when attached."},
			"managed_by":  schema.StringAttribute{Computed: true, MarkdownDescription: "Owning resource, if any."},
			"created_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *storageDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *storageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *storageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config storageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		storage *client.Storage
		err     error
	)
	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		storage, err = d.client.GetStorage(ctx, config.ID.ValueString())
	} else {
		storage, err = d.client.GetStorageByName(ctx, config.Name.ValueString())
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "storage", err)
		return
	}

	config.ID = types.StringValue(storage.ID)
	config.Name = types.StringValue(storage.Name)
	config.SizeGB = int64Value(storage.SizeGB)
	config.Path = stringValue(storage.Path)
	config.AttachedTo = stringValue(storage.AttachedTo)
	config.Port = int64Value(storage.Port)
	config.ManagedBy = stringValue(storage.ManagedBy)
	config.CreatedAt = stringValue(storage.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// --- Parameter ---

var (
	_ datasource.DataSource                     = &paramDataSource{}
	_ datasource.DataSourceWithConfigure        = &paramDataSource{}
	_ datasource.DataSourceWithConfigValidators = &paramDataSource{}
)

func NewParamDataSource() datasource.DataSource {
	return &paramDataSource{}
}

type paramDataSource struct {
	client *client.Client
}

type paramDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Value        types.String `tfsdk:"value"`
	IsJSON       types.Bool   `tfsdk:"is_json"`
	IsEnvCapable types.Bool   `tfsdk:"is_env_capable"`
	Size         types.Int64  `tfsdk:"size"`
	OwnerRID     types.String `tfsdk:"owner_rid"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (d *paramDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_param"
}

func (d *paramDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a value from the parameter store by id or by name.\n\n" +
			"~> The value is written into Terraform state as it is read. Parameters are not the " +
			"place for credentials; use `coderforge_secret` for those.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Resource id, for example `param:0012`. Specify this or `name`.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Parameter name. Specify this or `id`.",
			},
			"value":          schema.StringAttribute{Computed: true, MarkdownDescription: "The stored value."},
			"is_json":        schema.BoolAttribute{Computed: true},
			"is_env_capable": schema.BoolAttribute{Computed: true},
			"size":           schema.Int64Attribute{Computed: true},
			"owner_rid":      schema.StringAttribute{Computed: true},
			"created_at":     schema.StringAttribute{Computed: true},
			"updated_at":     schema.StringAttribute{Computed: true},
		},
	}
}

func (d *paramDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *paramDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *paramDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config paramDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		param *client.Param
		err   error
	)
	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		param, err = d.client.GetParam(ctx, config.ID.ValueString())
	} else {
		param, err = d.client.GetParamByName(ctx, config.Name.ValueString())
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "parameter", err)
		return
	}

	config.ID = types.StringValue(param.ID)
	config.Name = types.StringValue(param.Name)
	config.Value = stringValue(param.Value)
	config.IsJSON = types.BoolValue(param.IsJSON)
	config.IsEnvCapable = types.BoolValue(param.IsEnvCapable)
	config.Size = types.Int64Value(param.Size)
	config.OwnerRID = stringValue(param.OwnerRID)
	config.CreatedAt = stringValue(param.CreatedAt)
	config.UpdatedAt = stringValue(param.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// --- Secret ---

var (
	_ datasource.DataSource                     = &secretDataSource{}
	_ datasource.DataSourceWithConfigure        = &secretDataSource{}
	_ datasource.DataSourceWithConfigValidators = &secretDataSource{}
)

func NewSecretDataSource() datasource.DataSource {
	return &secretDataSource{}
}

type secretDataSource struct {
	client *client.Client
}

type secretDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Keys      types.List   `tfsdk:"keys"`
	OwnerRID  types.String `tfsdk:"owner_rid"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *secretDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (d *secretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a secret's metadata by id or by name, usually to wire its id " +
			"into a container service's `env_secret_rid`.\n\n" +
			"The material is deliberately not returned: a data source that read it would copy every " +
			"secret value into Terraform state on every plan. Only the key names are reported.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Resource id, for example `secret:0004`. Specify this or `name`.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Secret name. Specify this or `id`.",
			},
			"type": schema.StringAttribute{Computed: true, MarkdownDescription: "`KV` or `PKI`."},
			"keys": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Key names stored in the vault. The values are not returned.",
			},
			"owner_rid":  schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *secretDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *secretDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *secretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config secretDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		secret *client.Secret
		err    error
	)
	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		secret, err = d.client.GetSecret(ctx, config.ID.ValueString())
	} else {
		secret, err = d.client.GetSecretByName(ctx, config.Name.ValueString())
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "secret", err)
		return
	}

	keys, diags := stringListValue(ctx, secret.Keys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.ID = types.StringValue(secret.ID)
	config.Name = types.StringValue(secret.Name)
	config.Type = types.StringValue(secret.Type)
	config.Keys = keys
	config.OwnerRID = stringValue(secret.OwnerRID)
	config.CreatedAt = stringValue(secret.CreatedAt)
	config.UpdatedAt = stringValue(secret.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
