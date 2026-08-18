// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package client

// These types mirror the terraform-api schemas one for one. Pointer fields
// mark "not sent" as distinct from "sent as zero": on a PATCH the difference
// decides whether an attribute is left alone or set to zero, which for a
// machine's CPU count is not a subtle distinction.

// --- Machines ---

type Machine struct {
	Name         string  `json:"name"`
	RID          *string `json:"rid"`
	State        string  `json:"state"`
	RawStatus    string  `json:"raw_status"`
	DesiredState string  `json:"desired_state"`
	RAM          *int64  `json:"ram"`
	CPU          *int64  `json:"cpu"`
	Network      *string `json:"network"`
	Template     *string `json:"template"`
	IP           *string `json:"ip"`
	Username     *string `json:"username"`
	Password     *string `json:"password"`
	ManagedBy    *string `json:"managed_by"`
	Message      string  `json:"message"`
}

type MachineCreateRequest struct {
	Name               string `json:"name"`
	RAM                int64  `json:"ram"`
	CPU                int64  `json:"cpu"`
	Network            string `json:"network"`
	Template           string `json:"template"`
	DesiredState       string `json:"desired_state"`
	RollbackOnFailure  bool   `json:"rollback_on_failure"`
	Wait               bool   `json:"wait"`
	WaitTimeoutSeconds *int64 `json:"wait_timeout_seconds,omitempty"`
}

type MachineUpdateRequest struct {
	RAM                *int64  `json:"ram,omitempty"`
	CPU                *int64  `json:"cpu,omitempty"`
	Network            *string `json:"network,omitempty"`
	DesiredState       *string `json:"desired_state,omitempty"`
	Wait               bool    `json:"wait"`
	WaitTimeoutSeconds *int64  `json:"wait_timeout_seconds,omitempty"`
}

type MachineList struct {
	Machines []Machine `json:"machines"`
	Meta     ListMeta  `json:"meta"`
}

type ProvisioningLog struct {
	Name  string   `json:"name"`
	Lines []string `json:"lines"`
}

// --- Container services ---

type ContainerService struct {
	ID           string           `json:"id"`
	RID          *string          `json:"rid"`
	Name         string           `json:"name"`
	State        string           `json:"state"`
	RawStatus    string           `json:"raw_status"`
	DesiredState string           `json:"desired_state"`
	Nodes        int64            `json:"nodes"`
	Version      *int64           `json:"version"`
	SourceType   string           `json:"source_type"`
	SourceName   string           `json:"source_name"`
	ComposeFile  *string          `json:"compose_file"`
	Files        []string         `json:"files"`
	Provisioner  *string          `json:"provisioner"`
	EnvParamRID  *string          `json:"env_param_rid"`
	EnvSecretRID *string          `json:"env_secret_rid"`
	NetworkRID   *string          `json:"network_rid"`
	Storages     []ServiceStorage `json:"storages"`
	Containers   []string         `json:"containers"`
	NodeState    []ServiceNode    `json:"node_state"`
	ManagedBy    *string          `json:"managed_by"`
	Message      string           `json:"message"`
	CreatedAt    *string          `json:"created_at"`
	UpdatedAt    *string          `json:"updated_at"`
}

type ServiceNode struct {
	Index   int64   `json:"index"`
	VMName  string  `json:"vm_name"`
	IP      *string `json:"ip"`
	Status  string  `json:"status"`
	Message string  `json:"message"`
}

type ServiceStorage struct {
	Type      string  `json:"type"`
	SizeGB    *int64  `json:"size_gb,omitempty"`
	RID       *string `json:"rid,omitempty"`
	Port      int64   `json:"port"`
	NodeIndex *int64  `json:"node_index,omitempty"`
}

// BundleSource carries the deployable. Exactly one of Compose and
// ArchiveBase64 is populated; the service rejects both or neither.
type BundleSource struct {
	Compose         *string `json:"compose,omitempty"`
	ComposeFilename string  `json:"compose_filename,omitempty"`
	ArchiveBase64   *string `json:"archive_base64,omitempty"`
	ArchiveFilename *string `json:"archive_filename,omitempty"`
}

type ContainerServiceCreateRequest struct {
	Name               string           `json:"name"`
	Nodes              int64            `json:"nodes"`
	Source             BundleSource     `json:"source"`
	Provisioner        *string          `json:"provisioner,omitempty"`
	Storages           []ServiceStorage `json:"storages"`
	EnvParamRID        *string          `json:"env_param_rid,omitempty"`
	EnvSecretRID       *string          `json:"env_secret_rid,omitempty"`
	NetworkRID         *string          `json:"network_rid,omitempty"`
	DesiredState       string           `json:"desired_state"`
	Wait               bool             `json:"wait"`
	WaitTimeoutSeconds *int64           `json:"wait_timeout_seconds,omitempty"`
}

type ContainerServiceUpdateRequest struct {
	Name               *string          `json:"name,omitempty"`
	Nodes              *int64           `json:"nodes,omitempty"`
	Source             *BundleSource    `json:"source,omitempty"`
	Provisioner        *string          `json:"provisioner,omitempty"`
	Storages           []ServiceStorage `json:"storages,omitempty"`
	EnvParamRID        *string          `json:"env_param_rid,omitempty"`
	EnvSecretRID       *string          `json:"env_secret_rid,omitempty"`
	NetworkRID         *string          `json:"network_rid,omitempty"`
	DesiredState       *string          `json:"desired_state,omitempty"`
	Wait               bool             `json:"wait"`
	WaitTimeoutSeconds *int64           `json:"wait_timeout_seconds,omitempty"`
}

type ContainerServiceList struct {
	Services []ContainerService `json:"services"`
	Meta     ListMeta           `json:"meta"`
}

// --- Storage ---

type Storage struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	SizeGB     *int64  `json:"size_gb"`
	Path       *string `json:"path"`
	AttachedTo *string `json:"attached_to"`
	Port       *int64  `json:"port"`
	ManagedBy  *string `json:"managed_by"`
	CreatedAt  *string `json:"created_at"`
}

type StorageCreateRequest struct {
	Name       string  `json:"name"`
	SizeGB     int64   `json:"size_gb"`
	AttachedTo *string `json:"attached_to,omitempty"`
	Port       *int64  `json:"port,omitempty"`
}

type StorageUpdateRequest struct {
	AttachedTo *string `json:"attached_to"`
	Port       *int64  `json:"port"`
}

type StorageList struct {
	Storages []Storage `json:"storages"`
	Meta     ListMeta  `json:"meta"`
}

// --- Parameters ---

type Param struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Value        *string `json:"value"`
	IsJSON       bool    `json:"is_json"`
	IsEnvCapable bool    `json:"is_env_capable"`
	Size         int64   `json:"size"`
	OwnerRID     *string `json:"owner_rid"`
	CreatedAt    *string `json:"created_at"`
	UpdatedAt    *string `json:"updated_at"`
}

type ParamCreateRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ParamUpdateRequest struct {
	Value string `json:"value"`
}

type ParamList struct {
	Params []Param  `json:"params"`
	Meta   ListMeta `json:"meta"`
}

// --- Secrets ---

type Secret struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Keys      []string `json:"keys"`
	OwnerRID  *string  `json:"owner_rid"`
	CreatedAt *string  `json:"created_at"`
	UpdatedAt *string  `json:"updated_at"`
}

type SecretCreateRequest struct {
	Name       string            `json:"name"`
	SecretType string            `json:"secret_type"`
	Data       map[string]string `json:"data"`
}

type SecretUpdateRequest struct {
	Data map[string]string `json:"data"`
}

type SecretList struct {
	Secrets []Secret `json:"secrets"`
	Meta    ListMeta `json:"meta"`
}

// --- Network rules ---

type NetworkRule struct {
	ID        string  `json:"id"`
	RID       *string `json:"rid"`
	SrcPort   string  `json:"src_port"`
	DestIP    string  `json:"dest_ip"`
	DestPort  string  `json:"dest_port"`
	Protocol  string  `json:"protocol"`
	ManagedBy *string `json:"managed_by"`
}

type NetworkRuleCreateRequest struct {
	SrcPort  string `json:"src_port"`
	DestIP   string `json:"dest_ip"`
	DestPort string `json:"dest_port"`
	Protocol string `json:"protocol"`
}

type NetworkRuleList struct {
	Rules []NetworkRule `json:"rules"`
	Meta  ListMeta      `json:"meta"`
}

// --- Catalogue ---

type Template struct {
	File        string `json:"file"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
}

type TemplateList struct {
	Templates []Template `json:"templates"`
	Meta      ListMeta   `json:"meta"`
}

type Provisioner struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ProvisionerList struct {
	Provisioners []Provisioner `json:"provisioners"`
	Meta         ListMeta      `json:"meta"`
}

type ContainerNetwork struct {
	ID        string  `json:"id"`
	RID       string  `json:"rid"`
	Name      string  `json:"name"`
	Driver    string  `json:"driver"`
	CreatedAt *string `json:"created_at"`
}

type ContainerNetworkList struct {
	Networks []ContainerNetwork `json:"networks"`
	Meta     ListMeta           `json:"meta"`
}

// --- Shared ---

type ListMeta struct {
	Count int64 `json:"count"`
}

type DeletionResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
	Message string `json:"message"`
}
