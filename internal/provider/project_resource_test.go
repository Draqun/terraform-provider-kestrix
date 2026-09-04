package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProjectResourceSchemaAndMapping(t *testing.T) {
	res := newProjectResource()
	var schemaResponse resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	for _, name := range []string{"name", "repo_url", "repo_ref", "repo_credential_ref", "pipeline_dir", "pipeline_file", "gitlab_project_id", "default_branch", "webhook_secret"} {
		if _, ok := schemaResponse.Schema.Attributes[name]; !ok {
			t.Fatalf("missing resource attribute %q", name)
		}
	}

	plan := projectResourceModel{
		Name: types.StringValue("demo"), RepoURL: types.StringValue("https://example.test/repo.git"), RepoRef: types.StringValue("main"),
		RepoCredentialRef: types.StringValue("cred_demo"), PipelineFile: types.StringValue("pipeline.tf"),
	}
	if err := validatePlan(plan); err != nil {
		t.Fatal(err)
	}
	api := fromResource(plan)
	if api.Name != "demo" || api.RepoCredentialRef != "cred_demo" || api.RepoRef != "main" {
		t.Fatalf("API model = %+v", api)
	}
	state := toResource(projectModel{Name: "demo", RepoURL: api.RepoURL, RepoRef: api.RepoRef}, plan)
	if state.PipelineFile.ValueString() != "pipeline.tf" || state.RepoCredentialRef.ValueString() != "cred_demo" {
		t.Fatalf("state fallback = %+v", state)
	}
}

func TestProjectResourceRejectsPlaintextCredentialFieldsBySchema(t *testing.T) {
	res := newProjectResource()
	var schemaResponse resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	if _, ok := schemaResponse.Schema.Attributes["token"]; ok {
		t.Fatal("resource exposes plaintext token attribute")
	}
	if _, ok := schemaResponse.Schema.Attributes["private_key"]; ok {
		t.Fatal("resource exposes plaintext private key attribute")
	}
}

// Kestrix fills in these attributes on create. A plain Optional attribute would
// make Terraform reject the applied state as inconsistent with a null plan, so
// every server-normalized attribute has to be Computed as well.
func TestProjectResourceMarksServerNormalizedAttributesComputed(t *testing.T) {
	res := newProjectResource()
	var schemaResponse resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	for _, name := range []string{"pipeline_dir", "pipeline_file", "default_branch", "gitlab_project_id"} {
		attribute, ok := schemaResponse.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing resource attribute %q", name)
		}
		if !attribute.IsOptional() || !attribute.IsComputed() {
			t.Fatalf("attribute %q must be optional and computed, got optional=%v computed=%v", name, attribute.IsOptional(), attribute.IsComputed())
		}
	}
}

func TestOptionalStringNeverReturnsUnknown(t *testing.T) {
	if got := optionalString("", types.StringUnknown()); !got.IsNull() {
		t.Fatalf("unknown fallback = %v, want null", got)
	}
	if got := optionalString("", types.StringValue("fallback")); got.ValueString() != "fallback" {
		t.Fatalf("known fallback = %v", got)
	}
	if got := optionalString("server", types.StringNull()); got.ValueString() != "server" {
		t.Fatalf("server value = %v", got)
	}
}
