// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-coderforge/internal/client"
)

var (
	_ resource.Resource                = &paramResource{}
	_ resource.ResourceWithConfigure   = &paramResource{}
	_ resource.ResourceWithImportState = &paramResource{}
)

func NewParamResource() resource.Resource {
	return &paramResource{}
}

type paramResource struct {
	client *client.Client
}

type paramResourceModel struct {
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

func (r *paramResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_param"
}

func (r *paramResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A value in the Cloud Builder parameter store.\n\n" +
			"Parameters hold configuration, not credentials: the value is stored and read back in " +
			"clear text, and lands in Terraform state as written. Use `coderforge_secret` for " +
			"anything that must not.\n\n" +
			"A parameter whose value is a **JSON object** can be projected into a container " +
			"service's environment through `env_param_rid`: the platform turns the object's keys " +
			"into environment variables. Any other value - a JSON array, or plain `KEY=value` " +
			"lines - is stored perfectly well but is refused as an environment source. " +
			"`is_env_capable` reports which one you have.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Builder resource id, for example `param:0012`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Parameter name, unique across the platform. Renaming replaces " +
					"the parameter, which changes its resource id.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						paramNamePattern,
						"must start with a letter or digit and may contain letters, digits, dots, "+
							"dashes and underscores (64 characters maximum)",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The stored value, up to 256 KiB.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(262144),
				},
			},

			// --- Read-only ---
			"is_json": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the platform could parse the value as JSON.",
			},
			"is_env_capable": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether the value can be projected into a container service's " +
					"environment through `env_param_rid`. True only for a JSON object.",
			},
			"size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Size of the stored value in bytes.",
			},
			"owner_rid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Set when another resource owns this parameter.",
			},
			"created_at": schema.StringAttribute{Computed: true, MarkdownDescription: "When the parameter was created."},
			"updated_at": schema.StringAttribute{Computed: true, MarkdownDescription: "When the value was last changed."},
		},
	}
}

func (r *paramResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *paramResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan paramResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	param, err := r.client.CreateParam(ctx, client.ParamCreateRequest{
		Name:  plan.Name.ValueString(),
		Value: plan.Value.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "create", "parameter "+plan.Name.ValueString(), err)
		return
	}

	applyParam(&plan, param)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *paramResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state paramResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	param, err := r.client.GetParam(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Parameter no longer exists, removing it from state",
				map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "read", "parameter "+state.ID.ValueString(), err)
		return
	}

	applyParam(&state, param)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *paramResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state paramResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	param, err := r.client.UpdateParam(ctx, state.ID.ValueString(), client.ParamUpdateRequest{
		Value: plan.Value.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "update", "parameter "+state.ID.ValueString(), err)
		return
	}

	applyParam(&plan, param)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *paramResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state paramResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteParam(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "delete", "parameter "+state.ID.ValueString(), err)
	}
}

func (r *paramResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyParam(model *paramResourceModel, param *client.Param) {
	model.ID = types.StringValue(param.ID)
	model.Name = types.StringValue(param.Name)
	model.IsJSON = types.BoolValue(param.IsJSON)
	model.IsEnvCapable = types.BoolValue(param.IsEnvCapable)
	model.Size = types.Int64Value(param.Size)
	model.OwnerRID = stringValue(param.OwnerRID)
	model.CreatedAt = stringValue(param.CreatedAt)
	model.UpdatedAt = stringValue(param.UpdatedAt)

	// The create response omits the value it was just handed; only a read
	// carries it. Keeping the configured value in that case is correct and
	// avoids a spurious diff on the next plan.
	if param.Value != nil {
		model.Value = types.StringValue(*param.Value)
	}
}
