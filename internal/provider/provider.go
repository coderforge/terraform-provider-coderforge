// Copyright (c) 2026 CoderForge.org Ltd.
// Licensed under the MIT License.

// Package provider implements the CoderForge Terraform provider. It talks to
// terraform-api, which fronts Cloud Builder; see the README for the shape of
// that split and why it exists.
package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-coderforge/internal/client"
)

// Environment variables, in the order they are consulted. Configuration in
// HCL always wins; the environment is the fallback, which is what lets a
// pipeline supply a token without it appearing in a committed file.
const (
	envEndpoint   = "CODERFORGE_ENDPOINT"
	envToken      = "CODERFORGE_TOKEN"
	envInsecure   = "CODERFORGE_INSECURE"
	envMaxRetries = "CODERFORGE_MAX_RETRIES"
)

var (
	_ provider.Provider = &coderforgeProvider{}
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &coderforgeProvider{version: version}
	}
}

type coderforgeProvider struct {
	version string
}

type providerModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	Token          types.String `tfsdk:"token"`
	Insecure       types.Bool   `tfsdk:"insecure"`
	MaxRetries     types.Int64  `tfsdk:"max_retries"`
	RequestTimeout types.String `tfsdk:"request_timeout"`
}

func (p *coderforgeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "coderforge"
	resp.Version = p.version
}

func (p *coderforgeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage CoderForge Cloud Builder infrastructure: virtual machines, " +
			"container services, disks, parameters, secrets and port forwarding rules.\n\n" +
			"The provider talks to the `terraform-api` service, which fronts Cloud Builder. " +
			"Authentication uses the same OAuth2 bearer token as the Cloud Builder console.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Base URL of the terraform-api service, for example " +
					"`http://srv12.net.coderforge.org:8080`. May also be set with the " +
					"`CODERFORGE_ENDPOINT` environment variable. Defaults to `" + client.DefaultEndpoint + "`.",
			},
			"token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				MarkdownDescription: "OAuth2 bearer token. May also be set with the `CODERFORGE_TOKEN` " +
					"environment variable, which is the recommended way to supply it: a token written " +
					"into a configuration file ends up in version control.\n\n" +
					"Obtain one with:\n\n" +
					"```shell\n" +
					"curl -s -X POST https://auth.coderforge.org/api/1.3/auth/oauth2/token \\\n" +
					"  -H 'Content-Type: application/x-www-form-urlencoded' \\\n" +
					"  -d 'grant_type=password&username=USER&password=PASSWORD'\n" +
					"```",
			},
			"insecure": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Skip TLS certificate verification. Only for an endpoint using a " +
					"certificate from an internal CA that is not in the trust store. May also be set " +
					"with `CODERFORGE_INSECURE`.",
			},
			"max_retries": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "How many times to retry a request that failed because the API was " +
					"unreachable. Only read-only requests are retried, so a retry can never create a " +
					"second resource. Defaults to `3`; may also be set with `CODERFORGE_MAX_RETRIES`.",
			},
			"request_timeout": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Ceiling on a single API call, as a Go duration such as `\"90m\"`. " +
					"Creating a machine blocks until it has booted, so this has to be generous; the " +
					"per-resource `timeouts` block is the right place to bound an individual operation. " +
					"Defaults to `\"60m\"`.",
			},
		},
	}
}

func (p *coderforgeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An unknown value at this point means it comes from another resource that
	// has not been created yet. Terraform cannot build a client from it, and
	// the error it would otherwise produce names the wrong thing.
	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Provider endpoint is not known at configuration time",
			"The endpoint depends on a value that Terraform will only know after apply. "+
				"Set it to a literal, or supply it through the "+envEndpoint+" environment variable.",
		)
	}
	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Provider token is not known at configuration time",
			"The token depends on a value that Terraform will only know after apply. "+
				"Set it to a literal, or supply it through the "+envToken+" environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := stringOrEnv(config.Endpoint, envEndpoint, client.DefaultEndpoint)
	token := stringOrEnv(config.Token, envToken, "")

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing CoderForge API token",
			"The provider needs an OAuth2 bearer token. Set the `token` argument on the provider "+
				"block, or the "+envToken+" environment variable.\n\n"+
				"Generate one with:\n"+
				"  curl -s -X POST https://auth.coderforge.org/api/1.3/auth/oauth2/token \\\n"+
				"    -H 'Content-Type: application/x-www-form-urlencoded' \\\n"+
				"    -d 'grant_type=password&username=USER&password=PASSWORD'",
		)
		return
	}

	insecure := config.Insecure.ValueBool()
	if config.Insecure.IsNull() {
		insecure = os.Getenv(envInsecure) == "true" || os.Getenv(envInsecure) == "1"
	}

	maxRetries := int(client.DefaultMaxRetries)
	if !config.MaxRetries.IsNull() {
		maxRetries = int(config.MaxRetries.ValueInt64())
	} else if raw := os.Getenv(envMaxRetries); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("max_retries"),
				"Invalid "+envMaxRetries,
				fmt.Sprintf("%q is not a number: %s", raw, err),
			)
			return
		}
		maxRetries = parsed
	}

	requestTimeout := client.DefaultRequestTimeout
	if !config.RequestTimeout.IsNull() && config.RequestTimeout.ValueString() != "" {
		parsed, err := time.ParseDuration(config.RequestTimeout.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("request_timeout"),
				"Invalid request_timeout",
				fmt.Sprintf("%q is not a Go duration such as \"90m\" or \"3600s\": %s",
					config.RequestTimeout.ValueString(), err),
			)
			return
		}
		requestTimeout = parsed
	}

	// Masked before anything is logged, so an enabled TF_LOG never prints it.
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "token", "password", "data")
	tflog.Info(ctx, "Configuring the CoderForge client", map[string]any{
		"endpoint":    endpoint,
		"max_retries": maxRetries,
	})

	apiClient, err := client.New(client.Config{
		Endpoint:       endpoint,
		Token:          token,
		Insecure:       insecure,
		MaxRetries:     maxRetries,
		RequestTimeout: requestTimeout,
		UserAgent:      fmt.Sprintf("terraform-provider-coderforge/%s", p.version),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create the CoderForge API client",
			"The provider could not be configured: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func (p *coderforgeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMachineResource,
		NewContainerServiceResource,
		NewStorageResource,
		NewParamResource,
		NewSecretResource,
		NewNetworkRuleResource,
	}
}

func (p *coderforgeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewMachineDataSource,
		NewMachinesDataSource,
		NewContainerServiceDataSource,
		NewStorageDataSource,
		NewParamDataSource,
		NewSecretDataSource,
		NewVMTemplatesDataSource,
		NewContainerNetworksDataSource,
	}
}

// stringOrEnv resolves a provider argument: HCL first, then the environment,
// then the built-in default.
func stringOrEnv(value types.String, envKey, fallback string) string {
	if !value.IsNull() && value.ValueString() != "" {
		return value.ValueString()
	}
	if fromEnv := os.Getenv(envKey); fromEnv != "" {
		return fromEnv
	}
	return fallback
}
