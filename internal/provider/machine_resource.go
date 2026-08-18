// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-coderforge/internal/client"
)

var (
	_ resource.Resource                = &machineResource{}
	_ resource.ResourceWithConfigure   = &machineResource{}
	_ resource.ResourceWithImportState = &machineResource{}
)

func NewMachineResource() resource.Resource {
	return &machineResource{}
}

type machineResource struct {
	client *client.Client
}

type machineResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	RAM               types.Int64    `tfsdk:"ram"`
	CPU               types.Int64    `tfsdk:"cpu"`
	Network           types.String   `tfsdk:"network"`
	Template          types.String   `tfsdk:"template"`
	DesiredState      types.String   `tfsdk:"desired_state"`
	RollbackOnFailure types.Bool     `tfsdk:"rollback_on_failure"`
	RID               types.String   `tfsdk:"rid"`
	State             types.String   `tfsdk:"state"`
	RawStatus         types.String   `tfsdk:"raw_status"`
	IP                types.String   `tfsdk:"ip"`
	Username          types.String   `tfsdk:"username"`
	Password          types.String   `tfsdk:"password"`
	ManagedBy         types.String   `tfsdk:"managed_by"`
	Message           types.String   `tfsdk:"message"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

func (r *machineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (r *machineResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Cloud Builder virtual machine.\n\n" +
			"Creating one provisions a guest from a template and waits for it to finish booting, so " +
			"a completed apply means the machine is usable and its `ip` is known. Changing `ram`, " +
			"`cpu` or `network` restarts the machine: the hypervisor cannot apply those to a running " +
			"guest. The provider restores `desired_state` afterwards.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the machine, which is its name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Machine name, unique across the platform. Letters, digits, " +
					"underscores and dashes only. Renaming replaces the machine.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(
						machineNamePattern,
						"must contain only letters, digits, underscores and dashes",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ram": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(4096),
				MarkdownDescription: "Memory in MiB, between 512 and 32768. Changing this restarts the machine.",
				Validators: []validator.Int64{
					int64validator.Between(512, 32768),
				},
			},
			"cpu": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2),
				MarkdownDescription: "Virtual CPU count, between 1 and 16. Changing this restarts the machine.",
				Validators: []validator.Int64{
					int64validator.Between(1, 16),
				},
			},
			"network": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("hostonly"),
				MarkdownDescription: "Comma-separated network adapters, up to four, from `hostonly`, " +
					"`bridge`, `nat` and `intnet` - for example `\"hostonly,nat\"`. Changing this " +
					"restarts the machine.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						networkPattern,
						"must be a comma-separated list of hostonly, bridge, nat or intnet",
					),
				},
			},
			"template": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Template image to provision from, as reported by the " +
					"`coderforge_vm_templates` data source. Omit for the platform default. " +
					"The template cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"desired_state": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("running"),
				MarkdownDescription: "Power state to hold the machine in: `running` or `stopped`. " +
					"Terraform corrects drift on the next apply, so a machine stopped by hand is " +
					"started again.",
				Validators: []validator.String{
					stringvalidator.OneOf("running", "stopped"),
				},
			},
			// Optional, and deliberately not Computed with a default: this
			// flag only changes what the provider does while creating, and
			// nothing on the platform corresponds to it. A Computed default
			// would be written into state, and an imported machine - which has
			// no value for it - would then show a diff on the next plan.
			"rollback_on_failure": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Delete the half-built machine if provisioning fails. Defaults to " +
					"`true`: Terraform never records a machine whose creation failed, so anything left " +
					"behind is invisible to state. Set to `false` to keep it for inspection and remove " +
					"it by hand.",
			},

			// --- Read-only ---
			"rid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Builder resource id, for example `vm:0007`.",
			},
			"state": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Current state: `running`, `stopped`, `paused`, `provisioning`, " +
					"`pending`, `failed` or `unknown`.",
			},
			"raw_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hypervisor's own status string, for diagnostics.",
			},
			"ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Address assigned to the machine, once it has one.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Default account created by the template.",
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				MarkdownDescription: "Password generated at creation. Cloud Builder returns it exactly " +
					"once, so it is stored in Terraform state and never re-read. It is unavailable on " +
					"a machine brought under management with `terraform import`.",
			},
			"managed_by": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Set when another Cloud Builder resource owns this machine, such " +
					"as a container service node. A machine with an owner must not be managed directly.",
			},
			"message": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last message from the platform about this machine.",
			},
		},
		Blocks: map[string]schema.Block{
			// Block form, not attribute form: `timeouts { create = "40m" }` is
			// what every mainstream provider uses and what practitioners will
			// write. The attribute form would need `timeouts = { ... }`.
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *machineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *machineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan machineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, defaultMachineCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	machine, err := r.client.CreateMachine(ctx, client.MachineCreateRequest{
		Name:         plan.Name.ValueString(),
		RAM:          plan.RAM.ValueInt64(),
		CPU:          plan.CPU.ValueInt64(),
		Network:      plan.Network.ValueString(),
		Template:     plan.Template.ValueString(),
		DesiredState: plan.DesiredState.ValueString(),
		// Null means "not specified", which is rollback enabled. ValueBool()
		// alone would read a null as false and quietly invert the default.
		RollbackOnFailure: plan.RollbackOnFailure.IsNull() || plan.RollbackOnFailure.ValueBool(),
		Wait:              true,
		// The API's wait is bounded by the same number the practitioner wrote
		// in the timeouts block, so one ceiling governs the whole operation.
		WaitTimeoutSeconds: waitSeconds(createTimeout),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "create", "machine "+plan.Name.ValueString(), err)
		return
	}

	r.applyMachine(&plan, machine)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *machineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state machineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	machine, err := r.client.GetMachine(ctx, state.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Machine no longer exists, removing it from state", map[string]any{
				"name": state.Name.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "read", "machine "+state.Name.ValueString(), err)
		return
	}

	r.applyMachine(&state, machine)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *machineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state machineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, defaultMachineUpdateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Only what actually changed is sent. An unconditional PATCH of every
	// field would restart the machine whenever a tag-like attribute moved,
	// because the API restarts on any hardware change it is handed.
	update := client.MachineUpdateRequest{
		Wait:               true,
		WaitTimeoutSeconds: waitSeconds(updateTimeout),
	}
	if !plan.RAM.Equal(state.RAM) {
		update.RAM = int64Pointer(plan.RAM)
	}
	if !plan.CPU.Equal(state.CPU) {
		update.CPU = int64Pointer(plan.CPU)
	}
	if !plan.Network.Equal(state.Network) {
		update.Network = stringPointer(plan.Network)
	}
	if !plan.DesiredState.Equal(state.DesiredState) {
		update.DesiredState = stringPointer(plan.DesiredState)
	}

	if update.RAM == nil && update.CPU == nil && update.Network == nil && update.DesiredState == nil {
		// Only local-only attributes moved, such as rollback_on_failure. There
		// is nothing to ask the platform for.
		plan.copyComputedFrom(state)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return
	}

	machine, err := r.client.UpdateMachine(ctx, state.Name.ValueString(), update)
	if err != nil {
		addAPIError(&resp.Diagnostics, "update", "machine "+state.Name.ValueString(), err)
		return
	}

	// The password is never returned again after creation; carry it forward.
	plan.Password = state.Password
	r.applyMachine(&plan, machine)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *machineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state machineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, defaultMachineDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if err := r.client.DeleteMachine(ctx, state.Name.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return // Already gone; the destroy has nothing left to do.
		}
		addAPIError(&resp.Diagnostics, "delete", "machine "+state.Name.ValueString(), err)
	}
}

func (r *machineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The name is the identifier, so an import is `terraform import
	// coderforge_machine.web01 web01`. Both attributes are set: `id` is what
	// Terraform indexes on, `name` is what Read queries with.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)

	// Cloud Builder returns a generated password once, at creation, and an
	// imported machine has missed it. Saying so here beats a null in state
	// that the practitioner discovers when something else tries to use it.
	resp.Diagnostics.AddWarning(
		"Imported machine has no password in state",
		fmt.Sprintf("Cloud Builder returns the generated password for %q only when the machine is "+
			"created, so it cannot be recovered on import. `password` will be null. Retrieve the "+
			"credential from the Cloud Builder console if you need it.", req.ID),
	)
}

// applyMachine copies an API response into the model, leaving alone the
// attributes the API does not report.
func (r *machineResource) applyMachine(model *machineResourceModel, machine *client.Machine) {
	model.ID = types.StringValue(machine.Name)
	model.Name = types.StringValue(machine.Name)
	model.RID = stringValue(machine.RID)
	model.State = types.StringValue(machine.State)
	model.RawStatus = types.StringValue(machine.RawStatus)
	model.DesiredState = types.StringValue(machine.DesiredState)
	model.IP = stringValue(machine.IP)
	model.Username = stringValue(machine.Username)
	model.ManagedBy = stringValue(machine.ManagedBy)
	model.Message = types.StringValue(machine.Message)

	if machine.RAM != nil {
		model.RAM = types.Int64Value(*machine.RAM)
	}
	if machine.CPU != nil {
		model.CPU = types.Int64Value(*machine.CPU)
	}
	if machine.Network != nil && *machine.Network != "" {
		model.Network = types.StringValue(*machine.Network)
	}
	if machine.Template != nil && *machine.Template != "" {
		model.Template = types.StringValue(*machine.Template)
	}

	// Only the create response carries the password. Everywhere else the
	// value already in state is the only copy there is.
	model.Password = preserveIfEmpty(stringValue(machine.Password), model.Password)
}

// copyComputedFrom fills the computed attributes of a plan from prior state,
// for the case where an update touched nothing the platform knows about.
func (m *machineResourceModel) copyComputedFrom(state machineResourceModel) {
	m.ID = state.ID
	m.RID = state.RID
	m.State = state.State
	m.RawStatus = state.RawStatus
	m.IP = state.IP
	m.Username = state.Username
	m.Password = state.Password
	m.ManagedBy = state.ManagedBy
	m.Message = state.Message
}
