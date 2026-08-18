// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-coderforge/internal/client"
)

// Conversions between the framework's types and plain Go. The framework
// distinguishes null, unknown and set; the API distinguishes absent from
// zero. These helpers are where the two meet, and keeping them in one place
// is what stops each resource from inventing its own convention.

func stringValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func int64Value(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

// stringPointer returns nil for a null or unknown value, so it is omitted from
// the request body rather than sent as an empty string.
func stringPointer(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueString()
	return &value
}

func int64Pointer(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueInt64()
	return &value
}

// preserveIfEmpty keeps a value the API does not return on every read.
//
// A machine password is generated once and returned once; a read afterwards
// carries null. Overwriting state with that null would show the practitioner a
// permanent diff and lose the credential, so the prior value is kept when the
// API has nothing to say.
func preserveIfEmpty(fromAPI types.String, prior types.String) types.String {
	if fromAPI.IsNull() || fromAPI.ValueString() == "" {
		return prior
	}
	return fromAPI
}

func stringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

func stringListValue(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	if values == nil {
		values = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

func stringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

// configureClient is the boilerplate every resource and data source needs to
// pick up the configured client, with the type assertion failure reported as a
// provider bug rather than a practitioner error, which is what it would be.
func configureResourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T. This is a bug in the provider; "+
				"please report it.", req.ProviderData),
		)
		return nil
	}
	return apiClient
}

func configureDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T. This is a bug in the provider; "+
				"please report it.", req.ProviderData),
		)
		return nil
	}
	return apiClient
}

// addAPIError turns an API failure into a diagnostic worth reading. The
// service already writes messages for a human audience, so the job here is to
// name the operation and keep the rest intact.
func addAPIError(diags *diag.Diagnostics, action, resourceName string, err error) {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		detail := apiErr.Error()
		switch apiErr.Code {
		case client.CodeUnauthorized:
			detail += "\n\nThe token is invalid, expired or revoked. Generate a new one and set " +
				envToken + ", then run the command again."
		case client.CodeUpstreamUnavailable:
			detail += "\n\nThis is usually a wrong `endpoint` on the provider block, or the " +
				"terraform-api service being down. It is safe to retry."
		case client.CodeTimeout:
			detail += "\n\nThe operation may still be running. Check the resource in the Cloud " +
				"Builder console before retrying, and consider raising the `timeouts` block on " +
				"this resource."
		}
		diags.AddError(fmt.Sprintf("Unable to %s %s", action, resourceName), detail)
		return
	}
	diags.AddError(fmt.Sprintf("Unable to %s %s", action, resourceName), err.Error())
}

// waitSeconds converts a Terraform timeouts value into the wait ceiling the
// API expects. Zero means "the practitioner did not say", and the service
// applies its own default.
func waitSeconds(d time.Duration) *int64 {
	if d <= 0 {
		return nil
	}
	seconds := int64(d.Seconds())
	return &seconds
}
