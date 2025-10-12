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
	_ resource.Resource              = &eksResource{}
	_ resource.ResourceWithConfigure = &eksResource{}
)

func NewEksResource() resource.Resource {
	return &eksResource{}
}

type eksResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ClusterName           types.String `tfsdk:"cluster_name"`
	Version               types.String `tfsdk:"version"`
	Region                types.String `tfsdk:"region"`
	NodeGroupName         types.String `tfsdk:"node_group_name"`
	NodeInstanceType      types.String `tfsdk:"node_instance_type"`
	NodeMinSize           types.Int64  `tfsdk:"node_min_size"`
	NodeMaxSize           types.Int64  `tfsdk:"node_max_size"`
	NodeDesiredSize       types.Int64  `tfsdk:"node_desired_size"`
	VpcId                 types.String `tfsdk:"vpc_id"`
	SubnetIds             types.List   `tfsdk:"subnet_ids"`
	SecurityGroupIds      types.List   `tfsdk:"security_group_ids"`
	EndpointPrivateAccess types.Bool   `tfsdk:"endpoint_private_access"`
	EndpointPublicAccess  types.Bool   `tfsdk:"endpoint_public_access"`
	PublicAccessCidrs     types.List   `tfsdk:"public_access_cidrs"`
	LoggingEnabled        types.Bool   `tfsdk:"logging_enabled"`
	LogTypes              types.List   `tfsdk:"log_types"`
	Tags                  types.Map    `tfsdk:"tags"`
	LastUpdated           types.String `tfsdk:"last_updated"`
}

type eksResource struct {
	client *Client
}

func (r *eksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_eks"
}

func (r *eksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"cluster_name": schema.StringAttribute{
				Required: true,
			},
			"version": schema.StringAttribute{
				Optional: true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"node_group_name": schema.StringAttribute{
				Optional: true,
			},
			"node_instance_type": schema.StringAttribute{
				Optional: true,
			},
			"node_min_size": schema.Int64Attribute{
				Optional: true,
			},
			"node_max_size": schema.Int64Attribute{
				Optional: true,
			},
			"node_desired_size": schema.Int64Attribute{
				Optional: true,
			},
			"vpc_id": schema.StringAttribute{
				Optional: true,
			},
			"subnet_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"security_group_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"endpoint_private_access": schema.BoolAttribute{
				Optional: true,
			},
			"endpoint_public_access": schema.BoolAttribute{
				Optional: true,
			},
			"public_access_cidrs": schema.ListAttribute{
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
func (r *eksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan eksResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	var resourceItem ResourceItem
	resourceItem.Type = "eks"
	resourceItem.Name = plan.ClusterName.ValueString()
	
	// For EKS, we'll store configuration in a simplified way
	// In a real implementation, you'd want to extend the ResourceItem model
	// to support more complex configurations
	code := Code{
		PackageType: "kubernetes",
		Runtime:     plan.Version.ValueString(),
	}
	resourceItem.Code = code
	resourceItem.MaxRamSize = "0" // EKS doesn't use RAM size in the same way

	resourceItemRes, err := r.client.CreateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating EKS cluster",
			"Could not create EKS cluster, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(resourceItemRes.ID)
	plan.ClusterName = types.StringValue(resourceItemRes.Name)
	plan.Version = types.StringValue(resourceItemRes.Code.Runtime)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *eksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state eksResourceModel
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
			"Error Reading EKS Cluster",
			"Could not read EKS cluster ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	if resourceItemRes == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	
	state.ID = types.StringValue(resourceItemRes.ID)
	state.ClusterName = types.StringValue(resourceItemRes.Name)
	state.Version = types.StringValue(resourceItemRes.Code.Runtime)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan eksResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	var state eksResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
	
	var resourceItem ResourceItem
	resourceItem.ID = state.ID.ValueString()
	resourceItem.Type = "eks"
	resourceItem.Name = plan.ClusterName.ValueString()
	code := Code{
		PackageType: "kubernetes",
		Runtime:     plan.Version.ValueString(),
	}
	resourceItem.Code = code
	resourceItem.MaxRamSize = "0"

	resourceItemRes, err := r.client.UpdateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating EKS cluster",
			"Could not update EKS cluster, unexpected error: "+err.Error(),
		)
		return
	}
	
	plan.ID = types.StringValue(resourceItemRes.ID)
	plan.ClusterName = types.StringValue(resourceItemRes.Name)
	plan.Version = types.StringValue(resourceItemRes.Code.Runtime)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from plan
	var plan eksResourceModel
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteResource(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting EKS cluster",
			"Could not delete EKS cluster, unexpected error: "+err.Error(),
		)
	}
	return
}

// Configure adds the provider configured client to the resource.
func (r *eksResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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