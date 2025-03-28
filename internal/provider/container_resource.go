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
	_ resource.Resource              = &containerResource{}
	_ resource.ResourceWithConfigure = &containerResource{}
)

func NewContainerResource() resource.Resource {
	return &containerResource{}
}

type containerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Runtime     types.String `tfsdk:"runtime"`
	ImageUri    types.String `tfsdk:"image_uri"`
	Timeout     types.Int64  `tfsdk:"timeout"`
	MaxRamSize  types.String `tfsdk:"max_ram_size"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type containerResource struct {
	client *Client
}

func (r *containerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *containerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed: false,
				Optional: true,
			},
			"image_uri": schema.StringAttribute{
				Computed: false,
				Optional: true,
			},
			"runtime": schema.StringAttribute{
				Computed: false,
				Required: true,
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
func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan containerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	var resourceItem ResourceItem
	resourceItem.Type = "container"
	resourceItem.Name = plan.Name.ValueString()
	code := Code{
		ImageUri: plan.ImageUri.ValueString(),
		Runtime:  plan.Runtime.ValueString(),
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
	plan.Name = types.StringValue(resourceItemRes.Name)
	plan.ImageUri = types.StringValue(resourceItemRes.Code.ImageUri)
	plan.Runtime = types.StringValue(resourceItemRes.Code.Runtime)
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
func (r *containerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerResourceModel
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
	state.Name = types.StringValue(resourceItemRes.Name)
	state.ImageUri = types.StringValue(resourceItemRes.Code.ImageUri)
	state.Runtime = types.StringValue(resourceItemRes.Code.Runtime)
	state.Timeout = types.Int64Value(resourceItemRes.Timeout)
	state.MaxRamSize = types.StringValue(resourceItemRes.MaxRamSize)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *containerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan containerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	var state containerResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
	var resourceItem ResourceItem
	resourceItem.ID = state.ID.ValueString()
	resourceItem.Type = "container"
	resourceItem.Name = plan.Name.ValueString()
	code := Code{
		ImageUri: plan.ImageUri.ValueString(),
		Runtime:  plan.Runtime.ValueString(),
	}
	resourceItem.Code = code
	resourceItem.Timeout = plan.Timeout.ValueInt64()
	resourceItem.MaxRamSize = plan.MaxRamSize.ValueString()
	resourceItemRes, err := r.client.UpdateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating order",
			"Could not create order, unexpected error: "+err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(resourceItemRes.ID)
	state.Name = types.StringValue(resourceItemRes.Name)
	state.ImageUri = types.StringValue(resourceItemRes.Code.ImageUri)
	state.Runtime = types.StringValue(resourceItemRes.Code.Runtime)
	state.Timeout = types.Int64Value(resourceItemRes.Timeout)
	state.MaxRamSize = types.StringValue(resourceItemRes.MaxRamSize)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *containerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from plan
	var plan containerResourceModel
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
func (r *containerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
