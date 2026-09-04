package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Address is the provider source address used by Terraform and OpenTofu when
// resolving the plugin from the registry or a dev override.
const Address = "registry.terraform.io/kestrix/kestrix"

type Provider struct {
	version string
}

type providerConfig struct {
	Endpoint    types.String `tfsdk:"endpoint"`
	APIToken    types.String `tfsdk:"api_token"`
	CAFile      types.String `tfsdk:"ca_file"`
	Insecure    types.Bool   `tfsdk:"insecure"`
	TimeoutSecs types.Int64  `tfsdk:"timeout_seconds"`
}

type providerData struct {
	client *apiClient
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &Provider{version: version} }
}

func (p *Provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kestrix"
	resp.Version = p.version
}

func (p *Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{
		Attributes: map[string]providerschema.Attribute{
			"endpoint":        providerschema.StringAttribute{Optional: true, Description: "Kestrix API endpoint (KESTRIX_PROVIDER_ENDPOINT)."},
			"api_token":       providerschema.StringAttribute{Optional: true, Sensitive: true, Description: "Kestrix API token (KESTRIX_PROVIDER_API_TOKEN)."},
			"ca_file":         providerschema.StringAttribute{Optional: true, Description: "PEM CA bundle path (KESTRIX_PROVIDER_CA_FILE)."},
			"insecure":        providerschema.BoolAttribute{Optional: true, Description: "Disable TLS verification; not recommended."},
			"timeout_seconds": providerschema.Int64Attribute{Optional: true, Description: "HTTP timeout in seconds (KESTRIX_PROVIDER_TIMEOUT_SECONDS)."},
		},
	}
}

func (p *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerConfig
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint := valueOrEnv(config.Endpoint, "KESTRIX_PROVIDER_ENDPOINT", "http://localhost:8080")
	token := valueOrEnv(config.APIToken, "KESTRIX_PROVIDER_API_TOKEN", "")
	caFile := valueOrEnv(config.CAFile, "KESTRIX_PROVIDER_CA_FILE", "")
	insecure := config.Insecure.ValueBool()
	if config.Insecure.IsNull() || config.Insecure.IsUnknown() {
		insecure = os.Getenv("KESTRIX_PROVIDER_INSECURE") == "true"
	}
	timeout := config.TimeoutSecs.ValueInt64()
	if config.TimeoutSecs.IsNull() || config.TimeoutSecs.IsUnknown() {
		timeout = 30
		if raw := os.Getenv("KESTRIX_PROVIDER_TIMEOUT_SECONDS"); raw != "" {
			var parsed int64
			if _, err := fmt.Sscan(raw, &parsed); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("timeout_seconds"), "Invalid timeout", "KESTRIX_PROVIDER_TIMEOUT_SECONDS must be an integer.")
				return
			}
			timeout = parsed
		}
	}
	client, err := newAPIClient(endpoint, token, caFile, insecure, time.Duration(timeout)*time.Second)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Kestrix provider configuration", err.Error())
		return
	}
	data := &providerData{client: client}
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{newProjectResource}
}

func (p *Provider) DataSources(_ context.Context) []func() datasource.DataSource { return nil }

func valueOrEnv(value types.String, name, fallback string) string {
	if !value.IsNull() && !value.IsUnknown() && value.ValueString() != "" {
		return value.ValueString()
	}
	if env := os.Getenv(name); env != "" {
		return env
	}
	return fallback
}

var _ provider.Provider = (*Provider)(nil)
