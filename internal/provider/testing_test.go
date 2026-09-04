package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Test helpers that build the raw Terraform values the plugin framework hands
// to the provider at runtime, so resource and provider entry points can be
// driven without a Terraform CLI.

func testResourceSchema(t *testing.T) resourceschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	newProjectResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}

func testProviderSchema(t *testing.T) providerschema.Schema {
	t.Helper()
	var resp provider.SchemaResponse
	(&Provider{}).Schema(context.Background(), provider.SchemaRequest{}, &resp)
	return resp.Schema
}

// projectRaw builds a resource object value with every attribute null unless
// overridden, matching how Terraform sends partially configured objects.
func projectRaw(t *testing.T, schema resourceschema.Schema, overrides map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	values := map[string]tftypes.Value{
		"name":                tftypes.NewValue(tftypes.String, nil),
		"repo_url":            tftypes.NewValue(tftypes.String, nil),
		"repo_ref":            tftypes.NewValue(tftypes.String, nil),
		"repo_credential_ref": tftypes.NewValue(tftypes.String, nil),
		"pipeline_dir":        tftypes.NewValue(tftypes.String, nil),
		"pipeline_file":       tftypes.NewValue(tftypes.String, nil),
		"gitlab_project_id":   tftypes.NewValue(tftypes.Number, nil),
		"default_branch":      tftypes.NewValue(tftypes.String, nil),
		"webhook_secret":      tftypes.NewValue(tftypes.String, nil),
	}
	for name, value := range overrides {
		values[name] = value
	}
	return tftypes.NewValue(schema.Type().TerraformType(context.Background()), values)
}

func providerRaw(t *testing.T, schema providerschema.Schema, overrides map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	values := map[string]tftypes.Value{
		"endpoint":        tftypes.NewValue(tftypes.String, nil),
		"api_token":       tftypes.NewValue(tftypes.String, nil),
		"ca_file":         tftypes.NewValue(tftypes.String, nil),
		"insecure":        tftypes.NewValue(tftypes.Bool, nil),
		"timeout_seconds": tftypes.NewValue(tftypes.Number, nil),
	}
	for name, value := range overrides {
		values[name] = value
	}
	return tftypes.NewValue(schema.Type().TerraformType(context.Background()), values)
}

// singleAttrResourceSchema returns a schema that does not match
// projectResourceModel, so Plan.Get and State.Get report a decode error.
func singleAttrResourceSchema() resourceschema.Schema {
	return resourceschema.Schema{Attributes: map[string]resourceschema.Attribute{
		"name": resourceschema.StringAttribute{Required: true},
	}}
}

func singleAttrRaw(schema resourceschema.Schema) tftypes.Value {
	return tftypes.NewValue(schema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "demo"),
	})
}

func requireNoError(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}
}

func requireError(t *testing.T, diags diag.Diagnostics, context string) {
	t.Helper()
	if !diags.HasError() {
		t.Fatalf("expected diagnostics for %s", context)
	}
}
