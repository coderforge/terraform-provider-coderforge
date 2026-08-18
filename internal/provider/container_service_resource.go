// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-coderforge/internal/client"
)

var (
	_ resource.Resource                     = &containerServiceResource{}
	_ resource.ResourceWithConfigure        = &containerServiceResource{}
	_ resource.ResourceWithImportState      = &containerServiceResource{}
	_ resource.ResourceWithConfigValidators = &containerServiceResource{}
)

func NewContainerServiceResource() resource.Resource {
	return &containerServiceResource{}
}

type containerServiceResource struct {
	client *client.Client
}

type containerServiceModel struct {
	ID              types.String   `tfsdk:"id"`
	Name            types.String   `tfsdk:"name"`
	Nodes           types.Int64    `tfsdk:"nodes"`
	Compose         types.String   `tfsdk:"compose"`
	ComposeFilename types.String   `tfsdk:"compose_filename"`
	ArchiveBase64   types.String   `tfsdk:"archive_base64"`
	ArchiveFilename types.String   `tfsdk:"archive_filename"`
	Provisioner     types.String   `tfsdk:"provisioner_name"`
	Storage         []storageBlock `tfsdk:"storage"`
	EnvParamRID     types.String   `tfsdk:"env_param_rid"`
	EnvSecretRID    types.String   `tfsdk:"env_secret_rid"`
	NetworkRID      types.String   `tfsdk:"network_rid"`
	DesiredState    types.String   `tfsdk:"desired_state"`

	RID         types.String   `tfsdk:"rid"`
	State       types.String   `tfsdk:"state"`
	RawStatus   types.String   `tfsdk:"raw_status"`
	Version     types.Int64    `tfsdk:"version"`
	SourceType  types.String   `tfsdk:"source_type"`
	SourceName  types.String   `tfsdk:"source_name"`
	ComposeFile types.String   `tfsdk:"compose_file"`
	Files       types.List     `tfsdk:"files"`
	Containers  types.List     `tfsdk:"containers"`
	NodeState   types.List     `tfsdk:"node_state"`
	ManagedBy   types.String   `tfsdk:"managed_by"`
	Message     types.String   `tfsdk:"message"`
	CreatedAt   types.String   `tfsdk:"created_at"`
	UpdatedAt   types.String   `tfsdk:"updated_at"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

type storageBlock struct {
	Type      types.String `tfsdk:"type"`
	SizeGB    types.Int64  `tfsdk:"size_gb"`
	RID       types.String `tfsdk:"rid"`
	Port      types.Int64  `tfsdk:"port"`
	NodeIndex types.Int64  `tfsdk:"node_index"`
}

// nodeModel is the element of the computed node_state list. It is an
// attribute rather than a block: Terraform plans blocks from configuration, so
// a read-only block plans as empty and then disagrees with whatever the apply
// returns ("block count changed from 0 to 1").
type nodeModel struct {
	Index   types.Int64  `tfsdk:"index"`
	VMName  types.String `tfsdk:"vm_name"`
	IP      types.String `tfsdk:"ip"`
	Status  types.String `tfsdk:"status"`
	Message types.String `tfsdk:"message"`
}

func nodeObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"index":   types.Int64Type,
			"vm_name": types.StringType,
			"ip":      types.StringType,
			"status":  types.StringType,
			"message": types.StringType,
		},
	}
}

func (r *containerServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_service"
}

func (r *containerServiceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A containerised workload running on a Docker Swarm cluster that Cloud " +
			"Builder provisions and manages.\n\n" +
			"Each service owns `nodes` machines, forms a Swarm across them and deploys a compose " +
			"stack. Those machines appear in Cloud Builder as managed resources and must not be " +
			"declared as `coderforge_machine`; this resource owns their whole lifecycle.\n\n" +
			"Creating one waits for the cluster to form and the stack to converge, so a completed " +
			"apply means the containers are up.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Four-digit service id allocated by the platform.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the service.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						serviceNamePattern,
						"must start with a letter or digit and may contain letters, digits, spaces, "+
							"underscores and dashes (63 characters maximum)",
					),
				},
			},
			"nodes": schema.Int64Attribute{
				Required: true,
				MarkdownDescription: "Number of Swarm nodes, between 1 and 32. Raising this provisions " +
					"more machines and joins them to the cluster.",
				Validators: []validator.Int64{
					int64validator.Between(1, 32),
				},
			},
			"compose": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Compose file contents, usually from `file(\"${path.module}/compose.yaml\")`. " +
					"Mutually exclusive with `archive_base64`. Changing it redeploys the stack.",
			},
			"compose_filename": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("compose.yaml"),
				MarkdownDescription: "Filename to present `compose` under when uploading it.",
			},
			"archive_base64": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Base64 of a `.zip` or `.tar.gz` project bundle, for a stack that " +
					"needs more than a compose file - build contexts, Dockerfiles, configuration. " +
					"Usually `filebase64(\"${path.module}/project.zip\")`. Mutually exclusive with `compose`.",
			},
			"archive_filename": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Filename for the archive. The extension selects the reader, so " +
					"it must end in `.zip`, `.tar.gz` or `.tgz`. Required with `archive_base64`.",
			},
			// Named `provisioner_name` rather than `provisioner`: the latter is
			// a reserved root name in Terraform, since `provisioner` blocks are
			// core configuration syntax.
			"provisioner_name": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Provisioner script to run on each node before deploying, as " +
					"listed by the `coderforge_cs_provisioners` catalogue.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						provisionerNamePattern,
						"must contain only letters, digits, dots, dashes and underscores",
					),
				},
			},
			"env_param_rid": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Resource id of a `coderforge_param` whose contents are projected " +
					"into the stack's environment, for example `param:0004`. The parameter's value has " +
					"to be a JSON object - its keys become the environment variables. A parameter that " +
					"is anything else is refused here rather than silently contributing nothing.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(paramRIDPattern, "must look like param:0001"),
				},
			},
			"env_secret_rid": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Resource id of a `coderforge_secret` to project into the stack's " +
					"environment, for example `secret:0002`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(secretRIDPattern, "must look like secret:0001"),
				},
			},
			"network_rid": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Resource id of an existing Swarm overlay network, e.g. `cs_net:0001`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(networkRIDPattern, "must look like cs_net:0001"),
				},
			},
			"desired_state": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("running"),
				MarkdownDescription: "Whether the stack should be `running` or `stopped`.",
				Validators: []validator.String{
					stringvalidator.OneOf("running", "stopped"),
				},
			},

			// --- Read-only ---
			"rid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Builder resource id, for example `cs:0001`.",
			},
			"state": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Current state: `running`, `degraded`, `stopped`, `provisioning`, " +
					"`pending` or `failed`. `degraded` means the stack is up with something unhealthy in it.",
			},
			"raw_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The platform's own status string, for diagnostics.",
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Bundle version, incremented on each redeploy.",
			},
			"source_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "How the bundle was supplied: `compose` or an archive format.",
			},
			"source_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Filename the bundle was uploaded under.",
			},
			"compose_file": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Compose file the platform found inside the bundle.",
			},
			"files": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Files extracted from the bundle.",
			},
			"containers": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Service names declared by the compose file.",
			},
			"node_state": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The machines backing this service, with their addresses.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"index":   schema.Int64Attribute{Computed: true, MarkdownDescription: "Position in the cluster, from zero."},
						"vm_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the machine the platform provisioned."},
						"ip":      schema.StringAttribute{Computed: true, MarkdownDescription: "Address of the node, once it has one."},
						"status":  schema.StringAttribute{Computed: true, MarkdownDescription: "The node's own status."},
						"message": schema.StringAttribute{Computed: true, MarkdownDescription: "Last message about this node."},
					},
				},
			},
			"managed_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Set when another resource owns this service.",
			},
			"message": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last message from the platform about this service.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the service was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the service was last changed.",
			},
		},
		Blocks: map[string]schema.Block{
			// Block form: `timeouts { create = "90m" }`, as in every
			// mainstream provider.
			"timeouts": timeouts.BlockAll(ctx),
			"storage": schema.ListNestedBlock{
				MarkdownDescription: "Disks to attach to the service's nodes. A `new` disk is created " +
					"and destroyed with the service; an `existing` one is attached and left behind.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("new"),
							MarkdownDescription: "`new` to have the platform create the disk, `existing` to attach one.",
							Validators: []validator.String{
								stringvalidator.OneOf("new", "existing"),
							},
						},
						"size_gb": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Size in GiB. Required when `type` is `new`.",
							Validators: []validator.Int64{
								int64validator.Between(1, 500),
							},
						},
						"rid": schema.StringAttribute{
							Optional: true,
							MarkdownDescription: "Resource id of the disk to attach. Required when `type` " +
								"is `existing`; usually `coderforge_storage.example.id`.",
							Validators: []validator.String{
								stringvalidator.RegexMatches(storeRIDPattern, "must look like store:0001"),
							},
						},
						"port": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "SATA port to attach on.",
							Validators: []validator.Int64{
								int64validator.Between(0, 30),
							},
						},
						"node_index": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Attach to this node only. Omit to attach to every node.",
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
						},
					},
				},
			},
		},
	}
}

func (r *containerServiceResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		// The platform needs exactly one bundle source. Catching it here makes
		// it a plan-time error against the right attributes, rather than a 400
		// halfway through an apply.
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("compose"),
			path.MatchRoot("archive_base64"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("archive_base64"),
			path.MatchRoot("archive_filename"),
		),
	}
}

func (r *containerServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *containerServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerServiceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, defaultServiceCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	service, err := r.client.CreateContainerService(ctx, client.ContainerServiceCreateRequest{
		Name:               plan.Name.ValueString(),
		Nodes:              plan.Nodes.ValueInt64(),
		Source:             plan.bundleSource(),
		Provisioner:        stringPointer(plan.Provisioner),
		Storages:           plan.storages(),
		EnvParamRID:        stringPointer(plan.EnvParamRID),
		EnvSecretRID:       stringPointer(plan.EnvSecretRID),
		NetworkRID:         stringPointer(plan.NetworkRID),
		DesiredState:       plan.DesiredState.ValueString(),
		Wait:               true,
		WaitTimeoutSeconds: waitSeconds(createTimeout),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "create", "container service "+plan.Name.ValueString(), err)
		return
	}

	resp.Diagnostics.Append(applyContainerService(ctx, &plan, service)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *containerServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerServiceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.client.GetContainerService(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Container service no longer exists, removing it from state",
				map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "read", "container service "+state.ID.ValueString(), err)
		return
	}

	resp.Diagnostics.Append(applyContainerService(ctx, &state, service)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *containerServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state containerServiceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, defaultServiceUpdateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	update := client.ContainerServiceUpdateRequest{
		Wait:               true,
		WaitTimeoutSeconds: waitSeconds(updateTimeout),
	}
	if !plan.Name.Equal(state.Name) {
		update.Name = stringPointer(plan.Name)
	}
	if !plan.Nodes.Equal(state.Nodes) {
		update.Nodes = int64Pointer(plan.Nodes)
	}
	if !plan.Provisioner.Equal(state.Provisioner) {
		update.Provisioner = stringPointer(plan.Provisioner)
	}
	if !plan.EnvParamRID.Equal(state.EnvParamRID) {
		update.EnvParamRID = stringOrEmpty(plan.EnvParamRID)
	}
	if !plan.EnvSecretRID.Equal(state.EnvSecretRID) {
		update.EnvSecretRID = stringOrEmpty(plan.EnvSecretRID)
	}
	if !plan.NetworkRID.Equal(state.NetworkRID) {
		update.NetworkRID = stringOrEmpty(plan.NetworkRID)
	}
	if !plan.DesiredState.Equal(state.DesiredState) {
		update.DesiredState = stringPointer(plan.DesiredState)
	}
	if !storagesEqual(plan.Storage, state.Storage) {
		update.Storages = plan.storages()
	}

	// The bundle is re-uploaded only when its contents changed. Sending it on
	// every apply would redeploy the stack, and restart every container, for
	// something as small as a node-count change.
	if !plan.Compose.Equal(state.Compose) ||
		!plan.ArchiveBase64.Equal(state.ArchiveBase64) ||
		!plan.ComposeFilename.Equal(state.ComposeFilename) ||
		!plan.ArchiveFilename.Equal(state.ArchiveFilename) {
		source := plan.bundleSource()
		update.Source = &source
	}

	service, err := r.client.UpdateContainerService(ctx, state.ID.ValueString(), update)
	if err != nil {
		addAPIError(&resp.Diagnostics, "update", "container service "+state.ID.ValueString(), err)
		return
	}

	resp.Diagnostics.Append(applyContainerService(ctx, &plan, service)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *containerServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerServiceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, defaultServiceDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if err := r.client.DeleteContainerService(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "delete", "container service "+state.ID.ValueString(), err)
	}
}

func (r *containerServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// The bundle is not readable back in a form that would round-trip, so the
	// first plan after an import will want to redeploy it. Better to say so
	// than to let it look like unexplained drift.
	resp.Diagnostics.AddWarning(
		"Imported service will show a bundle change on the first plan",
		"Cloud Builder stores the deployed bundle, not the `compose` or `archive_base64` value that "+
			"produced it, so Terraform cannot match them up. The first plan after an import will "+
			"propose redeploying the bundle. Review it before applying.",
	)
}

// --- Model helpers ---

func (m *containerServiceModel) bundleSource() client.BundleSource {
	source := client.BundleSource{}
	if !m.Compose.IsNull() && m.Compose.ValueString() != "" {
		source.Compose = stringPointer(m.Compose)
		source.ComposeFilename = m.ComposeFilename.ValueString()
		if source.ComposeFilename == "" {
			source.ComposeFilename = "compose.yaml"
		}
		return source
	}
	source.ArchiveBase64 = stringPointer(m.ArchiveBase64)
	source.ArchiveFilename = stringPointer(m.ArchiveFilename)
	return source
}

func (m *containerServiceModel) storages() []client.ServiceStorage {
	storages := make([]client.ServiceStorage, 0, len(m.Storage))
	for _, block := range m.Storage {
		storages = append(storages, client.ServiceStorage{
			Type:      block.Type.ValueString(),
			SizeGB:    int64Pointer(block.SizeGB),
			RID:       stringPointer(block.RID),
			Port:      block.Port.ValueInt64(),
			NodeIndex: int64Pointer(block.NodeIndex),
		})
	}
	return storages
}

func storagesEqual(a, b []storageBlock) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Type.Equal(b[i].Type) || !a[i].SizeGB.Equal(b[i].SizeGB) ||
			!a[i].RID.Equal(b[i].RID) || !a[i].Port.Equal(b[i].Port) ||
			!a[i].NodeIndex.Equal(b[i].NodeIndex) {
			return false
		}
	}
	return true
}

// stringOrEmpty sends an empty string rather than omitting the field, which is
// how the API is told to clear an association. Omitting it means "leave alone".
func stringOrEmpty(v types.String) *string {
	value := ""
	if !v.IsNull() && !v.IsUnknown() {
		value = v.ValueString()
	}
	return &value
}

func applyContainerService(ctx context.Context, model *containerServiceModel, service *client.ContainerService) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(service.ID)
	model.Name = types.StringValue(service.Name)
	model.Nodes = types.Int64Value(service.Nodes)
	model.RID = stringValue(service.RID)
	model.State = types.StringValue(service.State)
	model.RawStatus = types.StringValue(service.RawStatus)
	model.DesiredState = types.StringValue(service.DesiredState)
	model.Version = int64Value(service.Version)
	model.SourceType = types.StringValue(service.SourceType)
	model.SourceName = types.StringValue(service.SourceName)
	model.ComposeFile = stringValue(service.ComposeFile)
	model.ManagedBy = stringValue(service.ManagedBy)
	model.Message = types.StringValue(service.Message)
	model.CreatedAt = stringValue(service.CreatedAt)
	model.UpdatedAt = stringValue(service.UpdatedAt)

	// The platform reports an unset provisioner as the sentinel "none", which
	// is not what the practitioner wrote. Writing it back would produce a
	// permanent diff against a configuration that simply omits the attribute.
	if service.Provisioner != nil && *service.Provisioner != "" && *service.Provisioner != "none" {
		model.Provisioner = types.StringValue(*service.Provisioner)
	}

	model.EnvParamRID = stringValue(service.EnvParamRID)
	model.EnvSecretRID = stringValue(service.EnvSecretRID)
	model.NetworkRID = stringValue(service.NetworkRID)

	files, d := stringListValue(ctx, service.Files)
	diags = append(diags, d...)
	model.Files = files

	containers, d := stringListValue(ctx, service.Containers)
	diags = append(diags, d...)
	model.Containers = containers

	nodes := make([]nodeModel, 0, len(service.NodeState))
	for _, node := range service.NodeState {
		nodes = append(nodes, nodeModel{
			Index:   types.Int64Value(node.Index),
			VMName:  types.StringValue(node.VMName),
			IP:      stringValue(node.IP),
			Status:  types.StringValue(node.Status),
			Message: types.StringValue(node.Message),
		})
	}

	nodeList, d := types.ListValueFrom(ctx, nodeObjectType(), nodes)
	diags = append(diags, d...)
	model.NodeState = nodeList

	return diags
}
