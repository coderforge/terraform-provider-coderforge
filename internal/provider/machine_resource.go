package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &machineResource{}
	_ resource.ResourceWithConfigure = &machineResource{}
)

func NewMachineResource() resource.Resource {
	return &machineResource{}
}

type machineResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Cpu              types.Int64  `tfsdk:"cpu"`
	Ram              types.String `tfsdk:"ram"`
	Timeout          types.Int64  `tfsdk:"timeout"`
	LogsGroup        types.String `tfsdk:"logs_group"`
	Virtual          types.Bool   `tfsdk:"virtual"`
	SecurityGroupIds types.List   `tfsdk:"security_group_ids"`
	Tags             types.Map    `tfsdk:"tags"`
	LastUpdated      types.String `tfsdk:"last_updated"`
}

type machineResource struct {
	client *Client
}

func (r *machineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (r *machineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"cpu": schema.Int64Attribute{
				Required: true,
			},
			"ram": schema.StringAttribute{
				Required: true,
			},
			"timeout": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"logs_group": schema.StringAttribute{
				Optional: true,
			},
			"virtual": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Default is true if not specified.",
			},
			"security_group_ids": schema.ListAttribute{
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

func (r *machineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan machineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var resourceItem ResourceItem
	resourceItem.Type = "machine"
	resourceItem.Name = plan.Name.ValueString()
	resourceItem.Cpu = plan.Cpu.ValueInt64()
	resourceItem.Ram = plan.Ram.ValueString()
	resourceItem.LogsGroup = plan.LogsGroup.ValueString()

	// Handle Timeout (Optional)
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		resourceItem.Timeout = plan.Timeout.ValueInt64()
	} else {
		resourceItem.Timeout = 300 // Default API timeout if needed
	}

	// Handle Virtual (Default True)
	if plan.Virtual.IsNull() || plan.Virtual.IsUnknown() {
		resourceItem.Virtual = true
	} else {
		resourceItem.Virtual = plan.Virtual.ValueBool()
	}

	if !plan.SecurityGroupIds.IsNull() {
		var sg []string
		diags = plan.SecurityGroupIds.ElementsAs(ctx, &sg, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		resourceItem.SecurityGroupIds = sg
	}

	if !plan.Tags.IsNull() {
		var tags map[string]string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		resourceItem.Tags = tags
	}

	res, err := r.client.CreateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError("Error creating machine", err.Error())
		return
	}

	plan.ID = types.StringValue(res.ID)
	plan.Virtual = types.BoolValue(res.Virtual)
	plan.Timeout = types.Int64Value(res.Timeout)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *machineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state machineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetResource(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading machine", err.Error())
		return
	}

	state.Name = types.StringValue(res.Name)
	state.Cpu = types.Int64Value(res.Cpu)
	state.Ram = types.StringValue(res.Ram)
	state.Virtual = types.BoolValue(res.Virtual)
	state.Timeout = types.Int64Value(res.Timeout)
	state.LogsGroup = types.StringValue(res.LogsGroup)

	sgList, _ := types.ListValueFrom(ctx, types.StringType, res.SecurityGroupIds)
	state.SecurityGroupIds = sgList
	tagsMap, _ := types.MapValueFrom(ctx, types.StringType, res.Tags)
	state.Tags = tagsMap

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *machineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan machineResourceModel
	req.Plan.Get(ctx, &plan)
	var state machineResourceModel
	req.State.Get(ctx, &state)

	var resourceItem ResourceItem
	resourceItem.ID = state.ID.ValueString()
	resourceItem.Type = "machine"
	resourceItem.Name = plan.Name.ValueString()
	resourceItem.Cpu = plan.Cpu.ValueInt64()
	resourceItem.Ram = plan.Ram.ValueString()
	resourceItem.LogsGroup = plan.LogsGroup.ValueString()

	if plan.Virtual.IsNull() || plan.Virtual.IsUnknown() {
		resourceItem.Virtual = true
	} else {
		resourceItem.Virtual = plan.Virtual.ValueBool()
	}

	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		resourceItem.Timeout = plan.Timeout.ValueInt64()
	}

	// Handle SGs and Tags (simplified for brevity, identical to Create)
	if !plan.SecurityGroupIds.IsNull() {
		var sg []string
		plan.SecurityGroupIds.ElementsAs(ctx, &sg, false)
		resourceItem.SecurityGroupIds = sg
	}
	if !plan.Tags.IsNull() {
		var t map[string]string
		plan.Tags.ElementsAs(ctx, &t, false)
		resourceItem.Tags = t
	}

	res, err := r.client.UpdateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError("Error updating machine", err.Error())
		return
	}

	plan.ID = types.StringValue(res.ID)
	plan.Virtual = types.BoolValue(res.Virtual)
	plan.Timeout = types.Int64Value(res.Timeout)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.State.Set(ctx, plan)
}

func (r *machineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state machineResourceModel
	req.State.Get(ctx, &state)
	err := r.client.DeleteResource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting machine", err.Error())
	}
}

func (r *machineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *provider.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}
