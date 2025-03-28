package provider

type CloudData struct {
	StackId       string         `json:"stackId"`
	CloudSpace    string         `json:"cloudSpace"`
	Locations     []string       `json:"locations"`
	ResourceItems []ResourceItem `json:"resourceItems"`
	DataItems     []DataItem     `json:"dataItems"`
}

type ResourceItem struct {
	ID                    string `json:"id,omitempty"`
	Type                  string `json:"type"`
	Name                  string `json:"name,omitempty"`
	Code                  Code   `json:"code,omitempty"`
	Timeout               int64  `json:"timeout,omitempty"`
	Protocol              string `json:"protocol,omitempty"`
	Domain                string `json:"domain,omitempty"`
	Port                  int64  `json:"port,omitempty"`
	Active                bool   `json:"active,omitempty"`
	LoadBalancePercentage int64  `json:"loadBalancePercentage,omitempty"`
	MaxRamSize            string `json:"maxRamSize"`
}

type Code struct {
	PackageType string `json:"packageType"`
	ImageUri    string `json:"imageUri,omitempty"`
	Runtime     string `json:"runtime,omitempty"`
}

type DataItem struct {
	ID    string `json:"id,omitempty"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type LogoutStruct struct {
	IdTokenHint string `json:"id_token_hint"`
}
