package provider

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func configureWith(t *testing.T, raw tftypes.Value, schema providerschema.Schema) provider.ConfigureResponse {
	t.Helper()
	var resp provider.ConfigureResponse
	(&Provider{version: "test"}).Configure(context.Background(), provider.ConfigureRequest{
		Config: tfsdk.Config{Schema: schema, Raw: raw},
	}, &resp)
	return resp
}

func TestNewReturnsConfiguredProvider(t *testing.T) {
	p := New("1.2.3")()
	var metadata provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &metadata)
	if metadata.Version != "1.2.3" {
		t.Fatalf("version = %q", metadata.Version)
	}
}

func TestProviderResourcesAndDataSources(t *testing.T) {
	p := &Provider{}
	resources := p.Resources(context.Background())
	if len(resources) != 1 {
		t.Fatalf("resources = %d", len(resources))
	}
	if _, ok := resources[0]().(*projectResource); !ok {
		t.Fatal("kestrix_project resource is not registered")
	}
	if got := p.DataSources(context.Background()); got != nil {
		t.Fatalf("data sources = %v", got)
	}
}

func TestProviderConfigureUsesExplicitConfiguration(t *testing.T) {
	// Explicit configuration must win over the environment.
	t.Setenv("KESTRIX_PROVIDER_ENDPOINT", "http://from-env.test")
	t.Setenv("KESTRIX_PROVIDER_API_TOKEN", "env-token")

	schema := testProviderSchema(t)
	resp := configureWith(t, providerRaw(t, schema, map[string]tftypes.Value{
		"endpoint":        tftypes.NewValue(tftypes.String, "https://kestrix.test"),
		"api_token":       tftypes.NewValue(tftypes.String, "config-token"),
		"insecure":        tftypes.NewValue(tftypes.Bool, true),
		"timeout_seconds": tftypes.NewValue(tftypes.Number, big.NewFloat(5)),
	}), schema)
	requireNoError(t, resp.Diagnostics)

	data, ok := resp.ResourceData.(*providerData)
	if !ok {
		t.Fatalf("resource data = %T", resp.ResourceData)
	}
	if data.client.baseURL.String() != "https://kestrix.test" || data.client.token != "config-token" {
		t.Fatalf("client = %+v", data.client)
	}
	if data.client.httpClient.Timeout.Seconds() != 5 {
		t.Fatalf("timeout = %s", data.client.httpClient.Timeout)
	}
	if resp.DataSourceData != resp.ResourceData {
		t.Fatal("data source and resource data must share one client")
	}
}

func TestProviderConfigureReadsEnvironment(t *testing.T) {
	t.Setenv("KESTRIX_PROVIDER_ENDPOINT", "https://env.kestrix.test")
	t.Setenv("KESTRIX_PROVIDER_API_TOKEN", "env-token")
	t.Setenv("KESTRIX_PROVIDER_INSECURE", "true")
	t.Setenv("KESTRIX_PROVIDER_TIMEOUT_SECONDS", "45")

	schema := testProviderSchema(t)
	resp := configureWith(t, providerRaw(t, schema, nil), schema)
	requireNoError(t, resp.Diagnostics)

	data := resp.ResourceData.(*providerData)
	if data.client.baseURL.String() != "https://env.kestrix.test" || data.client.token != "env-token" {
		t.Fatalf("client = %+v", data.client)
	}
	if data.client.httpClient.Timeout.Seconds() != 45 {
		t.Fatalf("timeout = %s", data.client.httpClient.Timeout)
	}
}

func TestProviderConfigureFallsBackToLocalDefaults(t *testing.T) {
	t.Setenv("KESTRIX_PROVIDER_ENDPOINT", "")
	t.Setenv("KESTRIX_PROVIDER_API_TOKEN", "")
	t.Setenv("KESTRIX_PROVIDER_CA_FILE", "")
	t.Setenv("KESTRIX_PROVIDER_INSECURE", "")
	t.Setenv("KESTRIX_PROVIDER_TIMEOUT_SECONDS", "")

	schema := testProviderSchema(t)
	resp := configureWith(t, providerRaw(t, schema, nil), schema)
	requireNoError(t, resp.Diagnostics)

	data := resp.ResourceData.(*providerData)
	if data.client.baseURL.String() != "http://localhost:8080" || data.client.token != "" {
		t.Fatalf("client = %+v", data.client)
	}
	if data.client.httpClient.Timeout.Seconds() != 30 {
		t.Fatalf("timeout = %s", data.client.httpClient.Timeout)
	}
}

func TestProviderConfigureRejectsInvalidInput(t *testing.T) {
	schema := testProviderSchema(t)

	badSchema := providerschema.Schema{Attributes: map[string]providerschema.Attribute{
		"endpoint": providerschema.StringAttribute{Optional: true},
	}}
	badRaw := tftypes.NewValue(badSchema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"endpoint": tftypes.NewValue(tftypes.String, "http://localhost:8080"),
	})
	requireError(t, configureWith(t, badRaw, badSchema).Diagnostics, "undecodable configuration")

	t.Setenv("KESTRIX_PROVIDER_TIMEOUT_SECONDS", "not-a-number")
	requireError(t, configureWith(t, providerRaw(t, schema, nil), schema).Diagnostics, "non-numeric timeout")

	t.Setenv("KESTRIX_PROVIDER_TIMEOUT_SECONDS", "")
	invalidEndpoint := providerRaw(t, schema, map[string]tftypes.Value{
		"endpoint": tftypes.NewValue(tftypes.String, "ftp://kestrix.test"),
	})
	requireError(t, configureWith(t, invalidEndpoint, schema).Diagnostics, "non-HTTP endpoint")
}

func TestValueOrEnvPrecedence(t *testing.T) {
	t.Setenv("KESTRIX_PROVIDER_TEST_VALUE", "from-env")
	if got := valueOrEnv(types.StringValue("explicit"), "KESTRIX_PROVIDER_TEST_VALUE", "fallback"); got != "explicit" {
		t.Fatalf("explicit value = %q", got)
	}
	if got := valueOrEnv(types.StringNull(), "KESTRIX_PROVIDER_TEST_VALUE", "fallback"); got != "from-env" {
		t.Fatalf("env value = %q", got)
	}
	if got := valueOrEnv(types.StringUnknown(), "KESTRIX_PROVIDER_UNSET_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("fallback value = %q", got)
	}
}
