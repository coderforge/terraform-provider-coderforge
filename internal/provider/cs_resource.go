package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &csResource{}
	_ resource.ResourceWithConfigure = &csResource{}
)

func NewCsResource() resource.Resource {
	return &csResource{}
}

type csResourceModel struct {
	BaseResourceModel
	ID                   types.String `tfsdk:"id"`
	ServiceName          types.String `tfsdk:"service_name"`
	DesiredCount         types.Int64  `tfsdk:"desired_count"`
	PlatformVersion      types.String `tfsdk:"platform_version"`
	ContainerPort        types.Int64  `tfsdk:"container_port"`
	ContainerName        types.String `tfsdk:"container_name"`
	ContainerImage       types.String `tfsdk:"container_image"`
	ContainerMemory      types.Int64  `tfsdk:"container_memory"`
	ContainerCpu         types.Int64  `tfsdk:"container_cpu"`
	EnvironmentVariables types.Map    `tfsdk:"environment_variables"`
}

type csResource struct {
	client *Client
}

func (r *csResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cs"
}

func (r *csResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"service_name": schema.StringAttribute{
				Required: true,
			},
			"desired_count": schema.Int64Attribute{
				Optional: true,
			},
			"platform_version": schema.StringAttribute{
				Optional: true,
			},
			"container_port": schema.Int64Attribute{
				Optional: true,
			},
			"container_name": schema.StringAttribute{
				Optional: true,
			},
			"container_image": schema.StringAttribute{
				Optional: true,
			},
			"container_memory": schema.Int64Attribute{
				Optional: true,
			},
			"container_cpu": schema.Int64Attribute{
				Optional: true,
			},
			"environment_variables": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			// Inherited fields from BaseResourceModel
			"security_group_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"logging_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"log_types": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create a new resource.
func (r *csResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan csResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	var resourceItem ResourceItem
	resourceItem.Type = "cs"
	resourceItem.Name = plan.ServiceName.ValueString()
	
	// For CS, we'll store configuration in a simplified way
	code := Code{
		PackageType: "container",
		ImageUri:    plan.ContainerImage.ValueString(),
		Runtime:     "container",
	}
	resourceItem.Code = code
	resourceItem.MaxRamSize = fmt.Sprintf("%d", plan.ContainerMemory.ValueInt64())

	resourceItemRes, err := r.client.CreateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CS service",
			"Could not create CS service, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(resourceItemRes.ID)
	plan.ServiceName = types.StringValue(resourceItemRes.Name)
	plan.ContainerImage = types.StringValue(resourceItemRes.Code.ImageUri)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *csResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state csResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceItemRes, err := r.client.GetResource(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading CS Service",
			"Could not read CS service ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	if resourceItemRes == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	
	state.ID = types.StringValue(resourceItemRes.ID)
	state.ServiceName = types.StringValue(resourceItemRes.Name)
	state.ContainerImage = types.StringValue(resourceItemRes.Code.ImageUri)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *csResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan csResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	var state csResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
	
	var resourceItem ResourceItem
	resourceItem.ID = state.ID.ValueString()
	resourceItem.Type = "cs"
	resourceItem.Name = plan.ServiceName.ValueString()
	code := Code{
		PackageType: "container",
		ImageUri:    plan.ContainerImage.ValueString(),
		Runtime:     "container",
	}
	resourceItem.Code = code
	resourceItem.MaxRamSize = fmt.Sprintf("%d", plan.ContainerMemory.ValueInt64())

	resourceItemRes, err := r.client.UpdateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating CS service",
			"Could not update CS service, unexpected error: "+err.Error(),
		)
		return
	}
	
	plan.ID = types.StringValue(resourceItemRes.ID)
	plan.ServiceName = types.StringValue(resourceItemRes.Name)
	plan.ContainerImage = types.StringValue(resourceItemRes.Code.ImageUri)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *csResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from plan
	var plan csResourceModel
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteResource(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting CS service",
			"Could not delete CS service, unexpected error: "+err.Error(),
		)
	}
	return
}

// Configure adds the provider configured client to the resource.
func (r *csResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *provider.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}