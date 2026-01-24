package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path" // IMPORT NECESSARIO
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithConfigure   = &serviceResource{}
	_ resource.ResourceWithImportState = &serviceResource{} // INTERFACCIA IMPORT AGGIUNTA
)

func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

type serviceResourceModel struct {
	ID               types.String      `tfsdk:"id"`
	Name             types.String      `tfsdk:"name"`
	Code             *serviceCodeModel `tfsdk:"code"`
	Timeout          types.Int64       `tfsdk:"timeout"`
	Cpu              types.Int64       `tfsdk:"cpu"`
	Ram              types.String      `tfsdk:"ram"`
	Virtual          types.Bool        `tfsdk:"virtual"`
	SecurityGroupIds types.List        `tfsdk:"security_group_ids"`
	Tags             types.Map         `tfsdk:"tags"`
	LogsGroup        types.String      `tfsdk:"logs_group"`
	LastUpdated      types.String      `tfsdk:"last_updated"`
}

type serviceCodeModel struct {
	Runtime    types.String `tfsdk:"runtime"`
	ExecuteCmd types.String `tfsdk:"execute_cmd"`
	ZipFile    types.String `tfsdk:"zip_file"`
	ImageUri   types.String `tfsdk:"image_uri"`
}

type serviceResource struct {
	client *Client
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"virtual": schema.BoolAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Default is true if not specified.",
			},
			"code": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"runtime": schema.StringAttribute{
						Required: true,
					},
					"execute_cmd": schema.StringAttribute{
						Required: true,
					},
					"zip_file": schema.StringAttribute{
						Optional: true,
					},
					"image_uri": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"timeout": schema.Int64Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"cpu": schema.Int64Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"ram": schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"security_group_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"logs_group": schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasImage := !plan.Code.ImageUri.IsNull() && !plan.Code.ImageUri.IsUnknown()
	hasZip := !plan.Code.ZipFile.IsNull() && !plan.Code.ZipFile.IsUnknown()

	if hasImage == hasZip {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Devi specificare esattamente uno tra 'image_uri' o 'zip_file' nel blocco 'code'.",
		)
		return
	}

	var resourceItem ResourceItem
	resourceItem.Type = "service"
	resourceItem.Name = plan.Name.ValueString()

	code := Code{
		Runtime:    plan.Code.Runtime.ValueString(),
		ExecuteCmd: plan.Code.ExecuteCmd.ValueString(),
		ImageUri:   plan.Code.ImageUri.ValueString(),
	}

	if hasZip {
		zipContent, err := os.ReadFile(plan.Code.ZipFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading zip file", err.Error())
			return
		}
		code.ZipFile = base64.StdEncoding.EncodeToString(zipContent)
	}
	resourceItem.Code = &code

	if !plan.Timeout.IsNull() {
		resourceItem.Timeout = plan.Timeout.ValueInt64()
	} else {
		resourceItem.Timeout = 300
	}

	if plan.Virtual.IsNull() || plan.Virtual.IsUnknown() {
		resourceItem.Virtual = true
	} else {
		resourceItem.Virtual = plan.Virtual.ValueBool()
	}

	if !plan.Cpu.IsNull() {
		resourceItem.Cpu = plan.Cpu.ValueInt64()
	}
	if !plan.Ram.IsNull() {
		resourceItem.Ram = plan.Ram.ValueString()
	}

	resourceItem.LogsGroup = plan.LogsGroup.ValueString()

	if !plan.SecurityGroupIds.IsNull() {
		var securityGroupIds []string
		diags = plan.SecurityGroupIds.ElementsAs(ctx, &securityGroupIds, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		resourceItem.SecurityGroupIds = securityGroupIds
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

	resourceItemRes, err := r.client.CreateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}

	plan.ID = types.StringValue(resourceItemRes.ID)
	plan.Name = types.StringValue(resourceItemRes.Name)
	plan.Virtual = types.BoolValue(resourceItemRes.Virtual)

	if resourceItemRes.Code != nil {
		if plan.Code == nil {
			plan.Code = &serviceCodeModel{}
		}
		plan.Code.Runtime = types.StringValue(resourceItemRes.Code.Runtime)
		plan.Code.ExecuteCmd = types.StringValue(resourceItemRes.Code.ExecuteCmd)
		plan.Code.ImageUri = types.StringValue(resourceItemRes.Code.ImageUri)
	}

	plan.Timeout = types.Int64Value(resourceItemRes.Timeout)
	plan.Cpu = types.Int64Value(resourceItemRes.Cpu)
	plan.Ram = types.StringValue(resourceItemRes.Ram)
	plan.LogsGroup = types.StringValue(resourceItemRes.LogsGroup)

	securityGroupIdsList, _ := types.ListValueFrom(ctx, types.StringType, resourceItemRes.SecurityGroupIds)
	plan.SecurityGroupIds = securityGroupIdsList
	tagsMap, _ := types.MapValueFrom(ctx, types.StringType, resourceItemRes.Tags)
	plan.Tags = tagsMap

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
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
		resp.Diagnostics.AddError("Error Reading Resource", err.Error())
		return
	}
	if resourceItemRes == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(resourceItemRes.ID)
	state.Name = types.StringValue(resourceItemRes.Name)
	state.Virtual = types.BoolValue(resourceItemRes.Virtual)
	state.Timeout = types.Int64Value(resourceItemRes.Timeout)
	state.Cpu = types.Int64Value(resourceItemRes.Cpu)
	state.Ram = types.StringValue(resourceItemRes.Ram)
	state.LogsGroup = types.StringValue(resourceItemRes.LogsGroup)

	if resourceItemRes.Code != nil {
		if state.Code == nil {
			state.Code = &serviceCodeModel{}
		}
		state.Code.Runtime = types.StringValue(resourceItemRes.Code.Runtime)
		state.Code.ExecuteCmd = types.StringValue(resourceItemRes.Code.ExecuteCmd)
		state.Code.ImageUri = types.StringValue(resourceItemRes.Code.ImageUri)
	}

	securityGroupIdsList, _ := types.ListValueFrom(ctx, types.StringType, resourceItemRes.SecurityGroupIds)
	state.SecurityGroupIds = securityGroupIdsList
	tagsMap, _ := types.MapValueFrom(ctx, types.StringType, resourceItemRes.Tags)
	state.Tags = tagsMap

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	req.Plan.Get(ctx, &plan)

	var state serviceResourceModel
	req.State.Get(ctx, &state)

	hasImage := !plan.Code.ImageUri.IsNull() && !plan.Code.ImageUri.IsUnknown()
	hasZip := !plan.Code.ZipFile.IsNull() && !plan.Code.ZipFile.IsUnknown()

	if hasImage == hasZip {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Devi specificare esattamente uno tra 'image_uri' o 'zip_file' nel blocco 'code'.",
		)
		return
	}

	var resourceItem ResourceItem
	resourceItem.ID = state.ID.ValueString()
	resourceItem.Type = "service"
	resourceItem.Name = plan.Name.ValueString()

	code := Code{
		Runtime:    plan.Code.Runtime.ValueString(),
		ExecuteCmd: plan.Code.ExecuteCmd.ValueString(),
		ImageUri:   plan.Code.ImageUri.ValueString(),
	}

	if hasZip {
		zipContent, err := os.ReadFile(plan.Code.ZipFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading zip file", err.Error())
			return
		}
		code.ZipFile = base64.StdEncoding.EncodeToString(zipContent)
	}
	resourceItem.Code = &code

	if !plan.Timeout.IsNull() {
		resourceItem.Timeout = plan.Timeout.ValueInt64()
	} else {
		resourceItem.Timeout = 300
	}

	if plan.Virtual.IsNull() || plan.Virtual.IsUnknown() {
		resourceItem.Virtual = true
	} else {
		resourceItem.Virtual = plan.Virtual.ValueBool()
	}

	if !plan.Cpu.IsNull() {
		resourceItem.Cpu = plan.Cpu.ValueInt64()
	}
	if !plan.Ram.IsNull() {
		resourceItem.Ram = plan.Ram.ValueString()
	}
	resourceItem.LogsGroup = plan.LogsGroup.ValueString()

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

	resourceItemRes, err := r.client.UpdateResource(ctx, resourceItem)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}

	plan.ID = types.StringValue(resourceItemRes.ID) // ID from response (or state)
	plan.Name = types.StringValue(resourceItemRes.Name)
	plan.Virtual = types.BoolValue(resourceItemRes.Virtual)

	if resourceItemRes.Code != nil && plan.Code != nil {
		plan.Code.Runtime = types.StringValue(resourceItemRes.Code.Runtime)
		plan.Code.ExecuteCmd = types.StringValue(resourceItemRes.Code.ExecuteCmd)
		plan.Code.ImageUri = types.StringValue(resourceItemRes.Code.ImageUri)
	}

	plan.Timeout = types.Int64Value(resourceItemRes.Timeout)
	plan.Cpu = types.Int64Value(resourceItemRes.Cpu)
	plan.Ram = types.StringValue(resourceItemRes.Ram)
	plan.LogsGroup = types.StringValue(resourceItemRes.LogsGroup)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.State.Set(ctx, plan)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	req.State.Get(ctx, &state)
	err := r.client.DeleteResource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
	}
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
