// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &secretResource{}
	_ resource.ResourceWithConfigure   = &secretResource{}
	_ resource.ResourceWithImportState = &secretResource{}
)

func NewSecretResource() resource.Resource {
	return &secretResource{}
}

type secretResource struct {
	client *client.Client
}

type secretResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Data      types.Map    `tfsdk:"data"`
	Keys      types.List   `tfsdk:"keys"`
	OwnerRID  types.String `tfsdk:"owner_rid"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *secretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *secretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A secret in the Cloud Builder vault.\n\n" +
			"`data` is write-only: the provider never reads the material back, so a value changed " +
			"in the vault by hand is not detected as drift. What is tracked is the set of keys, " +
			"which does detect a secret that has been restructured.\n\n" +
			"~> **The value is still in Terraform state.** Terraform stores every attribute it is " +
			"given, including this one. Protect the state file accordingly.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Builder resource id, for example `secret:0004`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Secret name, unique across the platform. Renaming replaces it.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("KV"),
				MarkdownDescription: "`KV` for key/value material, `PKI` for certificate material.",
				Validators: []validator.String{
					stringvalidator.OneOf("KV", "PKI"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data": schema.MapAttribute{
				Required:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
				MarkdownDescription: "The secret material. Replaced wholesale on every change.",
			},

			// --- Read-only ---
			"keys": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Key names stored in the vault, sorted. The values are not returned.",
			},
			"owner_rid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Set when another resource owns this secret, such as machine credentials.",
			},
			"created_at": schema.StringAttribute{Computed: true, MarkdownDescription: "When the secret was created."},
			"updated_at": schema.StringAttribute{Computed: true, MarkdownDescription: "When the material was last replaced."},
		},
	}
}

func (r *secretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, diags := stringMap(ctx, plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.CreateSecret(ctx, client.SecretCreateRequest{
		Name:       plan.Name.ValueString(),
		SecretType: plan.Type.ValueString(),
		Data:       data,
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "create", "secret "+plan.Name.ValueString(), err)
		return
	}

	resp.Diagnostics.Append(applySecret(ctx, &plan, secret)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.GetSecret(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Info(ctx, "Secret no longer exists, removing it from state",
				map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		addAPIError(&resp.Diagnostics, "read", "secret "+state.ID.ValueString(), err)
		return
	}

	// `data` is deliberately not refreshed: the API returns metadata only, and
	// the configured value is the only copy of the material the provider has.
	resp.Diagnostics.Append(applySecret(ctx, &state, secret)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state secretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, diags := stringMap(ctx, plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.UpdateSecret(ctx, state.ID.ValueString(), client.SecretUpdateRequest{
		Data: data,
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, "update", "secret "+state.ID.ValueString(), err)
		return
	}

	resp.Diagnostics.Append(applySecret(ctx, &plan, secret)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSecret(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		addAPIError(&resp.Diagnostics, "delete", "secret "+state.ID.ValueString(), err)
	}
}

func (r *secretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	resp.Diagnostics.AddWarning(
		"Imported secret has no material in state",
		"The vault does not return secret material to this provider, so `data` cannot be recovered "+
			"on import. Declare the same values in your configuration; the first apply will write "+
			"them to the vault.",
	)
}

func applySecret(ctx context.Context, model *secretResourceModel, secret *client.Secret) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(secret.ID)
	model.Name = types.StringValue(secret.Name)
	model.Type = types.StringValue(secret.Type)
	model.OwnerRID = stringValue(secret.OwnerRID)
	model.CreatedAt = stringValue(secret.CreatedAt)
	model.UpdatedAt = stringValue(secret.UpdatedAt)

	keys, d := stringListValue(ctx, secret.Keys)
	diags = append(diags, d...)
	model.Keys = keys

	return diags
}
