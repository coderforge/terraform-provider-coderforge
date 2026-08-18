// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package client

import (
	"context"
	"fmt"
	"net/url"
)

// One method per API operation. They are thin on purpose: path construction
// and nothing else, so that a change to the contract is a one-line change here
// rather than a hunt through the resource implementations.

// --- Machines ---

func (c *Client) GetMachine(ctx context.Context, name string) (*Machine, error) {
	var machine Machine
	if err := c.get(ctx, "/machines/"+url.PathEscape(name), &machine); err != nil {
		return nil, err
	}
	return &machine, nil
}

func (c *Client) ListMachines(ctx context.Context) ([]Machine, error) {
	var list MachineList
	if err := c.get(ctx, "/machines", &list); err != nil {
		return nil, err
	}
	return list.Machines, nil
}

func (c *Client) CreateMachine(ctx context.Context, req MachineCreateRequest) (*Machine, error) {
	var machine Machine
	if err := c.post(ctx, "/machines", req, &machine); err != nil {
		return nil, err
	}
	return &machine, nil
}

func (c *Client) UpdateMachine(ctx context.Context, name string, req MachineUpdateRequest) (*Machine, error) {
	var machine Machine
	if err := c.patch(ctx, "/machines/"+url.PathEscape(name), req, &machine); err != nil {
		return nil, err
	}
	return &machine, nil
}

func (c *Client) DeleteMachine(ctx context.Context, name string) error {
	return c.delete(ctx, "/machines/"+url.PathEscape(name), nil)
}

func (c *Client) MachineProvisioningLog(ctx context.Context, name string, tail int64) (*ProvisioningLog, error) {
	var log ProvisioningLog
	path := fmt.Sprintf("/machines/%s/provisioning-log?tail=%d", url.PathEscape(name), tail)
	if err := c.get(ctx, path, &log); err != nil {
		return nil, err
	}
	return &log, nil
}

func (c *Client) ListTemplates(ctx context.Context) ([]Template, error) {
	var list TemplateList
	if err := c.get(ctx, "/vm-templates", &list); err != nil {
		return nil, err
	}
	return list.Templates, nil
}

// --- Container services ---

func (c *Client) GetContainerService(ctx context.Context, id string) (*ContainerService, error) {
	var service ContainerService
	if err := c.get(ctx, "/container-services/"+url.PathEscape(id), &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) ListContainerServices(ctx context.Context, name string) ([]ContainerService, error) {
	path := "/container-services"
	if name != "" {
		path += "?name=" + url.QueryEscape(name)
	}
	var list ContainerServiceList
	if err := c.get(ctx, path, &list); err != nil {
		return nil, err
	}
	return list.Services, nil
}

func (c *Client) CreateContainerService(ctx context.Context, req ContainerServiceCreateRequest) (*ContainerService, error) {
	var service ContainerService
	if err := c.post(ctx, "/container-services", req, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) UpdateContainerService(ctx context.Context, id string, req ContainerServiceUpdateRequest) (*ContainerService, error) {
	var service ContainerService
	if err := c.patch(ctx, "/container-services/"+url.PathEscape(id), req, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) DeleteContainerService(ctx context.Context, id string) error {
	return c.delete(ctx, "/container-services/"+url.PathEscape(id), nil)
}

func (c *Client) ListProvisioners(ctx context.Context) ([]Provisioner, error) {
	var list ProvisionerList
	if err := c.get(ctx, "/cs-provisioners", &list); err != nil {
		return nil, err
	}
	return list.Provisioners, nil
}

func (c *Client) ListContainerNetworks(ctx context.Context, name string) ([]ContainerNetwork, error) {
	path := "/cs-networks"
	if name != "" {
		path += "?name=" + url.QueryEscape(name)
	}
	var list ContainerNetworkList
	if err := c.get(ctx, path, &list); err != nil {
		return nil, err
	}
	return list.Networks, nil
}

// --- Storage ---

func (c *Client) GetStorage(ctx context.Context, rid string) (*Storage, error) {
	var storage Storage
	if err := c.get(ctx, "/storages/"+url.PathEscape(rid), &storage); err != nil {
		return nil, err
	}
	return &storage, nil
}

func (c *Client) GetStorageByName(ctx context.Context, name string) (*Storage, error) {
	var storage Storage
	if err := c.get(ctx, "/storages/by-name/"+url.PathEscape(name), &storage); err != nil {
		return nil, err
	}
	return &storage, nil
}

func (c *Client) CreateStorage(ctx context.Context, req StorageCreateRequest) (*Storage, error) {
	var storage Storage
	if err := c.post(ctx, "/storages", req, &storage); err != nil {
		return nil, err
	}
	return &storage, nil
}

func (c *Client) UpdateStorage(ctx context.Context, rid string, req StorageUpdateRequest) (*Storage, error) {
	var storage Storage
	if err := c.patch(ctx, "/storages/"+url.PathEscape(rid), req, &storage); err != nil {
		return nil, err
	}
	return &storage, nil
}

func (c *Client) DeleteStorage(ctx context.Context, rid string) error {
	return c.delete(ctx, "/storages/"+url.PathEscape(rid), nil)
}

// --- Parameters ---

func (c *Client) GetParam(ctx context.Context, rid string) (*Param, error) {
	var param Param
	if err := c.get(ctx, "/params/"+url.PathEscape(rid), &param); err != nil {
		return nil, err
	}
	return &param, nil
}

func (c *Client) GetParamByName(ctx context.Context, name string) (*Param, error) {
	var param Param
	if err := c.get(ctx, "/params/by-name/"+url.PathEscape(name), &param); err != nil {
		return nil, err
	}
	return &param, nil
}

func (c *Client) CreateParam(ctx context.Context, req ParamCreateRequest) (*Param, error) {
	var param Param
	if err := c.post(ctx, "/params", req, &param); err != nil {
		return nil, err
	}
	return &param, nil
}

func (c *Client) UpdateParam(ctx context.Context, rid string, req ParamUpdateRequest) (*Param, error) {
	var param Param
	if err := c.patch(ctx, "/params/"+url.PathEscape(rid), req, &param); err != nil {
		return nil, err
	}
	return &param, nil
}

func (c *Client) DeleteParam(ctx context.Context, rid string) error {
	return c.delete(ctx, "/params/"+url.PathEscape(rid), nil)
}

// --- Secrets ---

func (c *Client) GetSecret(ctx context.Context, rid string) (*Secret, error) {
	var secret Secret
	if err := c.get(ctx, "/secrets/"+url.PathEscape(rid), &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

func (c *Client) GetSecretByName(ctx context.Context, name string) (*Secret, error) {
	var secret Secret
	if err := c.get(ctx, "/secrets/by-name/"+url.PathEscape(name), &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

func (c *Client) CreateSecret(ctx context.Context, req SecretCreateRequest) (*Secret, error) {
	var secret Secret
	if err := c.post(ctx, "/secrets", req, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

func (c *Client) UpdateSecret(ctx context.Context, rid string, req SecretUpdateRequest) (*Secret, error) {
	var secret Secret
	if err := c.patch(ctx, "/secrets/"+url.PathEscape(rid), req, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

func (c *Client) DeleteSecret(ctx context.Context, rid string) error {
	return c.delete(ctx, "/secrets/"+url.PathEscape(rid), nil)
}

// --- Network rules ---

func (c *Client) GetNetworkRule(ctx context.Context, id string) (*NetworkRule, error) {
	var rule NetworkRule
	if err := c.get(ctx, "/network-rules/"+url.PathEscape(id), &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *Client) ListNetworkRules(ctx context.Context, destIP string) ([]NetworkRule, error) {
	path := "/network-rules"
	if destIP != "" {
		path += "?dest_ip=" + url.QueryEscape(destIP)
	}
	var list NetworkRuleList
	if err := c.get(ctx, path, &list); err != nil {
		return nil, err
	}
	return list.Rules, nil
}

func (c *Client) CreateNetworkRule(ctx context.Context, req NetworkRuleCreateRequest) (*NetworkRule, error) {
	var rule NetworkRule
	if err := c.post(ctx, "/network-rules", req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *Client) DeleteNetworkRule(ctx context.Context, id string) error {
	return c.delete(ctx, "/network-rules/"+url.PathEscape(id), nil)
}
