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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-coderforge/internal/client"
)

var (
	_ resource.Resource                = &networkRuleResource{}
	_ resource.ResourceWithConfigure   = &networkRuleResource{}
	_ resource.ResourceWithImportState = &networkRuleResource{}
)

func NewNetworkRuleResource() resource.Resource {
	return &networkRuleResource{}
}

type networkRuleResource struct {
	client *client.Client
}

type networkRuleResourceModel struct {
	ID        types.String `tfsdk:"id"`
	SrcPort   types.String `tfsdk:"src_port"`
	DestIP    types.String `tfsdk:"dest_ip"`
	DestPort  types.String `tfsdk:"dest_port"`
	Protocol  types.String `tfsdk:"protocol"`
	RID       types.String `tfsdk:"rid"`
	ManagedBy types.String `tfsdk:"managed_by"`
}

func (r *networkRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_rule"
}

func (r *networkRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A port forwarding rule from the host to a machine.\n\n" +
			"The platform identifies a rule by its whole tuple rather than by a handle, so every " +
			"attribute forces replacement: there is no way to modify a rule in place, only to remove " +
			"one and add another. The `id` is derived from the tuple, which is what makes " +
			"`terraform import` work without looking anything up.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Derived identifier, `<protocol>_<src_port>_<dest_ip>_<dest_port>` - " +
					"for example `tcp_8080_192.168.56.10_80`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"src_port": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Host port or range, as a string: `\"8080\"` or `\"50000-50100\"`. " +
					"A range must be the same width as `dest_port`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(portSpecPattern, "must be a port such as \"8080\" or a range such as \"50000-50100\""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dest_ip": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Address traffic is forwarded to, usually a machine's `ip` " +
					"attribute.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(ipv4Pattern, "must be an IPv4 address"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dest_port": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Guest port or range.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(portSpecPattern, "must be a port such as \"80\" or a range such as \"50000-50100\""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"protocol": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("tcp"),
				MarkdownDescription: "`tcp` or `udp`.",
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "udp"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// --- Read-only ---
			"rid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Builder registry id, when the rule carries one.",
			},
			"managed_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Set when another resource owns this rule.",
			},
		},
	}
}

func (r *networkRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *networkRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.CreateNetworkRule(ctx, client.NetworkRuleCreateRequest{
		SrcPort:  plan.SrcPort.ValueString(),
		DestIP:   plan.DestIP.ValueString(),
		DestPort: plan.DestPort.ValueString(),
		Protocol: plan.Protocol.ValueString(),
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "create", "network rule", err)
		return
	}

	applyNetworkRule(&plan, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *networkRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetNetworkRule(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Network rule no longer exists, removing it from state",
				map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "read", "network rule "+state.ID.ValueString(), err)
		return
	}

	applyNetworkRule(&state, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update cannot happen: every attribute is RequiresReplace, so Terraform
// destroys and recreates instead. The method exists to satisfy the interface.
func (r *networkRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Network rules cannot be updated in place",
		"Every attribute of a network rule forces replacement, so this should be unreachable. "+
			"Please report it as a provider bug.",
	)
}

func (r *networkRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteNetworkRule(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "delete", "network rule "+state.ID.ValueString(), err)
	}
}

func (r *networkRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The id encodes the whole rule, so Read fills in every attribute from it.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyNetworkRule(model *networkRuleResourceModel, rule *client.NetworkRule) {
	model.ID = types.StringValue(rule.ID)
	model.SrcPort = types.StringValue(rule.SrcPort)
	model.DestIP = types.StringValue(rule.DestIP)
	model.DestPort = types.StringValue(rule.DestPort)
	model.Protocol = types.StringValue(rule.Protocol)
	model.RID = stringValue(rule.RID)
	model.ManagedBy = stringValue(rule.ManagedBy)
}
