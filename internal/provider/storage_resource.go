// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-coderforge/internal/client"
)

var (
	_ resource.Resource                     = &storageResource{}
	_ resource.ResourceWithConfigure        = &storageResource{}
	_ resource.ResourceWithImportState      = &storageResource{}
	_ resource.ResourceWithConfigValidators = &storageResource{}
)

func NewStorageResource() resource.Resource {
	return &storageResource{}
}

type storageResource struct {
	client *client.Client
}

type storageResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Name       types.String   `tfsdk:"name"`
	SizeGB     types.Int64    `tfsdk:"size_gb"`
	AttachedTo types.String   `tfsdk:"attached_to"`
	Port       types.Int64    `tfsdk:"port"`
	Path       types.String   `tfsdk:"path"`
	ManagedBy  types.String   `tfsdk:"managed_by"`
	CreatedAt  types.String   `tfsdk:"created_at"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func (r *storageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage"
}

func (r *storageResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A virtual disk, optionally attached to a machine.\n\n" +
			"Attachment is an attribute of the disk rather than a resource of its own, so one apply " +
			"creates a disk and gives it to a machine. Moving it to another machine detaches and " +
			"reattaches in place, without destroying the data.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Builder resource id, for example `store:0003`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Disk name, unique across the platform. Renaming replaces the disk.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"size_gb": schema.Int64Attribute{
				Required: true,
				MarkdownDescription: "Size in GiB, between 1 and 500. The platform cannot resize a disk, " +
					"so changing this destroys and recreates it, losing the contents.",
				Validators: []validator.Int64{
					int64validator.Between(1, 500),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"attached_to": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Name of the machine to attach the disk to. Omit to leave it " +
					"detached, or to hand the disk to a `coderforge_container_service`, which " +
					"attaches it to its own nodes and owns that attachment. Requires `port`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						machineNamePattern,
						"must contain only letters, digits, underscores and dashes",
					),
				},
			},
			"port": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "SATA port to attach on, between 0 and 30. Required with " +
					"`attached_to`, and unique per machine.",
				Validators: []validator.Int64{
					int64validator.Between(0, 30),
				},
			},

			// --- Read-only ---
			"path": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Path of the disk image on the hypervisor host.",
			},
			"managed_by": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Set when a container service owns this disk. A disk with an " +
					"owner must not be managed directly.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the disk was created.",
			},
		},
		Blocks: map[string]schema.Block{
			// Block form: `timeouts { create = "10m" }`.
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *storageResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		// A port without a machine has nothing to attach to, and a machine
		// without a port has nowhere to attach. Neither reaches the API.
		resourcevalidator.RequiredTogether(
			path.MatchRoot("attached_to"),
			path.MatchRoot("port"),
		),
	}
}

func (r *storageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *storageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, defaultStorageTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	storage, err := r.client.CreateStorage(ctx, client.StorageCreateRequest{
		Name:       plan.Name.ValueString(),
		SizeGB:     plan.SizeGB.ValueInt64(),
		AttachedTo: stringPointer(plan.AttachedTo),
		Port:       int64Pointer(plan.Port),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "create", "storage "+plan.Name.ValueString(), err)
		return
	}

	applyStorage(&plan, storage)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *storageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	storage, err := r.client.GetStorage(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Storage no longer exists, removing it from state",
				map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "read", "storage "+state.ID.ValueString(), err)
		return
	}

	// Whether this resource is the one managing the attachment, decided from
	// prior state before the refresh overwrites it. A disk given to a
	// container service through its `storage` block is attached by the
	// service, to a node the service owns - so the attachment is real, but it
	// is not this resource's to report. Refreshing it anyway would put a value
	// in state that the configuration does not declare, and every subsequent
	// plan would offer to detach a disk nobody asked to detach.
	//
	// Drift detection is unaffected for a disk this resource does attach:
	// state holds a value there, so the refresh happens and a detach performed
	// behind Terraform's back still shows up.
	managesAttachment := !state.AttachedTo.IsNull()
	priorAttachedTo, priorPort := state.AttachedTo, state.Port

	applyStorage(&state, storage)

	if !managesAttachment {
		state.AttachedTo = priorAttachedTo
		state.Port = priorPort
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *storageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state storageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, defaultStorageTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Only the attachment can change; name and size are RequiresReplace.
	storage, err := r.client.UpdateStorage(ctx, state.ID.ValueString(), client.StorageUpdateRequest{
		AttachedTo: stringPointer(plan.AttachedTo),
		Port:       int64Pointer(plan.Port),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "update", "storage "+state.ID.ValueString(), err)
		return
	}

	applyStorage(&plan, storage)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *storageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, defaultStorageTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	// The API detaches before deleting: the platform refuses to delete an
	// attached disk, and a destroy has no separate detach step.
	if err := r.client.DeleteStorage(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "delete", "storage "+state.ID.ValueString(), err)
	}
}

func (r *storageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyStorage(model *storageResourceModel, storage *client.Storage) {
	model.ID = types.StringValue(storage.ID)
	model.Name = types.StringValue(storage.Name)
	model.Path = stringValue(storage.Path)
	model.ManagedBy = stringValue(storage.ManagedBy)
	model.CreatedAt = stringValue(storage.CreatedAt)
	model.AttachedTo = stringValue(storage.AttachedTo)
	model.Port = int64Value(storage.Port)

	if storage.SizeGB != nil {
		model.SizeGB = types.Int64Value(*storage.SizeGB)
	}
}
