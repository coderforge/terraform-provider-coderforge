package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &functionResource{}
	_ resource.ResourceWithConfigure = &functionResource{}
)

func NewFunctionResource() resource.Resource {
	return &functionResource{}
}

type functionResourceModel struct {
	ID           types.String      `tfsdk:"id"`
	FunctionName types.String      `tfsdk:"function_name"`
	Code         functionCodeModel `tfsdk:"code"`
	Timeout      types.Int64       `tfsdk:"timeout"`
	MaxRamSize   types.String      `tfsdk:"max_ram_size"`
	LastUpdated  types.String      `tfsdk:"last_updated"`
}

type functionCodeModel struct {
	PackageType types.String `tfsdk:"package_type"`
	ImageUri    types.String `tfsdk:"image_uri"`
}

type functionResource struct {
	client *Client
}

func (r *functionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (r *functionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"function_name": schema.StringAttribute{
				Computed: false,
				Required: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"code": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"package_type": schema.StringAttribute{
						Computed: false,
						Required: true,
					},
					"image_uri": schema.StringAttribute{
						Computed: false,
						Optional: true,
					},
				},
			},
			"timeout": schema.Int64Attribute{
				Computed: false,
				Optional: true,
			},
			"max_ram_size": schema.StringAttribute{
				Computed: false,
				Optional: true,
			},
		},
	}
}

// Create a new resource.
func (r *functionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan functionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	var resourceItem ResourceItem
	resourceItem.Type = "function"
	resourceItem.FunctionName = plan.FunctionName.ValueString()
	code := Code{
		PackageType: plan.Code.PackageType.ValueString(),
		ImageUri:    plan.Code.ImageUri.ValueString(),
	}
	resourceItem.Code = code
	resourceItem.Timeout = plan.Timeout.ValueInt64()
	resourceItem.MaxRamSize = plan.MaxRamSize.ValueString()
	resourceItemRes, err := r.client.CreateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating resource",
			"Could not create resource, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(resourceItemRes.ID)
	plan.FunctionName = types.StringValue(resourceItemRes.FunctionName)
	if &resourceItemRes.Code != nil {
		plan.Code = functionCodeModel{
			PackageType: types.StringValue(resourceItemRes.Code.PackageType),
			ImageUri:    types.StringValue(resourceItemRes.Code.ImageUri),
		}
	}
	plan.Timeout = types.Int64Value(resourceItemRes.Timeout)
	plan.MaxRamSize = types.StringValue(resourceItemRes.MaxRamSize)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *functionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state functionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed order value from HashiCups
	resourceItemRes, err := r.client.GetResource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Resource",
			"Could not read resource ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	state.ID = types.StringValue(resourceItemRes.ID)
	state.FunctionName = types.StringValue(resourceItemRes.FunctionName)
	state.Code.PackageType = types.StringValue(resourceItemRes.Code.PackageType)
	state.Code.ImageUri = types.StringValue(resourceItemRes.Code.ImageUri)
	state.MaxRamSize = types.StringValue(resourceItemRes.MaxRamSize)
	state.Timeout = types.Int64Value(resourceItemRes.Timeout)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *functionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan functionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	var state functionResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
	var resourceItem ResourceItem
	resourceItem.Type = "function"
	resourceItem.FunctionName = plan.FunctionName.ValueString()
	code := Code{
		PackageType: plan.Code.PackageType.ValueString(),
		ImageUri:    plan.Code.ImageUri.ValueString(),
	}
	resourceItem.Code = code
	resourceItem.Timeout = plan.Timeout.ValueInt64()
	resourceItem.MaxRamSize = plan.MaxRamSize.ValueString()
	resourceItem.ID = state.ID.ValueString()
	resourceItemRes, err := r.client.UpdateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating order",
			"Could not create order, unexpected error: "+err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(resourceItemRes.ID)
	plan.FunctionName = types.StringValue(resourceItemRes.FunctionName)
	if &resourceItemRes.Code != nil {
		plan.Code = functionCodeModel{
			PackageType: types.StringValue(resourceItemRes.Code.PackageType),
			ImageUri:    types.StringValue(resourceItemRes.Code.ImageUri),
		}
	}
	plan.MaxRamSize = types.StringValue(resourceItemRes.MaxRamSize)
	plan.Timeout = types.Int64Value(resourceItemRes.Timeout)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *functionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from plan
	var plan functionResourceModel
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteResource(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting resource",
			"Could not delete resource, unexpected error: "+err.Error(),
		)
	}
	return
}

// Configure adds the provider configured client to the resource.
func (r *functionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *hashicups.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}
