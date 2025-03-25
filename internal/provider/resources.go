package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *Client) GetResource(ctx context.Context, resourceID string) (*ResourceItem, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/1.2/cloud/terraform/resource?resourceId=%s&cloudSpace=%s", c.HostURL, resourceID, c.CloudSpace), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	cloudData := CloudData{}
	err = json.Unmarshal(body, &cloudData)
	if err != nil {
		return nil, err
	}

	resourceItems := &cloudData.ResourceItems

	if len(*resourceItems) > 0 {
		return &(*resourceItems)[0], nil
	}

	return nil, nil
}

func (c *Client) CreateResource(ctx context.Context, resourceItem ResourceItem) (*ResourceItem, error) {
	cloudData := CloudData{
		StackId:    c.StackId,
		CloudSpace: c.CloudSpace,
		Locations:  c.Locations,
		ResourceItems: []ResourceItem{
			resourceItem,
		},
	}

	rb, err := json.Marshal(cloudData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/1.2/cloud/terraform/resource", c.HostURL), io.NopCloser(strings.NewReader(string(rb))))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	cloudDataRes := CloudData{}
	err = json.Unmarshal(body, &cloudDataRes)
	if err != nil {
		return nil, err
	}

	resourceItems := &cloudDataRes.ResourceItems

	if len(*resourceItems) > 0 {
		return &(*resourceItems)[0], nil
	}

	return nil, nil
}

func (c *Client) UpdateResource(ctx context.Context, resourceItem ResourceItem) (*ResourceItem, error) {
	cloudData := CloudData{
		StackId:    c.StackId,
		CloudSpace: c.CloudSpace,
		Locations:  c.Locations,
		ResourceItems: []ResourceItem{
			resourceItem,
		},
	}
	rb, err := json.Marshal(cloudData)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/1.2/cloud/terraform/resource", c.HostURL), io.NopCloser(strings.NewReader(string(rb))))
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	cloudDataRes := CloudData{}
	err = json.Unmarshal(body, &cloudDataRes)
	if err != nil {
		return nil, err
	}
	resourceItems := &cloudDataRes.ResourceItems
	if len(*resourceItems) > 0 {
		return &(*resourceItems)[0], nil
	}
	return nil, nil
}

func (c *Client) DeleteResource(ctx context.Context, resourceID string) error {
	cloudData := CloudData{
		StackId:    c.StackId,
		CloudSpace: c.CloudSpace,
		Locations:  c.Locations,
	}
	rb, err := json.Marshal(cloudData)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/1.2/cloud/terraform/resource?resourceId=%s", c.HostURL, resourceID), io.NopCloser(strings.NewReader(string(rb))))
	if err != nil {
		return err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return err
	}
	cloudDataRes := CloudData{}
	err = json.Unmarshal(body, &cloudDataRes)
	if err != nil {
		return err
	}
	return nil
}
