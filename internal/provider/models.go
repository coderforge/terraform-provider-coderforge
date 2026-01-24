package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type CloudData struct {
	StackId       string         `json:"stackId"`
	CloudSpace    string         `json:"cloudSpace"`
	Locations     []string       `json:"locations"`
	ResourceItems []ResourceItem `json:"resourceItems"`
	DataItems     []DataItem     `json:"dataItems"`
}

type ResourceItem struct {
	ID                    string            `json:"id,omitempty"`
	Type                  string            `json:"type"`
	Name                  string            `json:"name,omitempty"`
	Code                  *Code             `json:"code,omitempty"` // Pointer
	Timeout               int64             `json:"timeout,omitempty"`
	Protocol              string            `json:"protocol,omitempty"`
	Domain                string            `json:"domain,omitempty"`
	Port                  int64             `json:"port,omitempty"`
	Active                bool              `json:"active,omitempty"`
	LoadBalancePercentage int64             `json:"loadBalancePercentage,omitempty"`
	Cpu                   int64             `json:"cpu,omitempty"`
	Ram                   string            `json:"ram,omitempty"`
	SecurityGroupIds      []string          `json:"securityGroupIds,omitempty"`
	Tags                  map[string]string `json:"tags,omitempty"`
	LogsGroup             string            `json:"logsGroup,omitempty"`
	Virtual               bool              `json:"virtual"` // Added Virtual
}

type Code struct {
	PackageType string `json:"packageType,omitempty"`
	ImageUri    string `json:"imageUri,omitempty"`
	Runtime     string `json:"runtime,omitempty"`
	ExecuteCmd  string `json:"executeCmd,omitempty"`
	ZipFile     string `json:"zipFile,omitempty"`
}

type DataItem struct {
	ID    string `json:"id,omitempty"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// BaseResourceModel contains common fields that all resources inherit
type BaseResourceModel struct {
	SecurityGroupIds types.List   `tfsdk:"security_group_ids"`
	LoggingEnabled   types.Bool   `tfsdk:"logging_enabled"`
	LogTypes         types.List   `tfsdk:"log_types"`
	Tags             types.Map    `tfsdk:"tags"`
	LastUpdated      types.String `tfsdk:"last_updated"`
}
