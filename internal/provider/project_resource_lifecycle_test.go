package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// projectAPI serves the subset of the Kestrix project API the resource calls,
// so the resource lifecycle is exercised without a running Kestrix server.
func projectAPI(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/projects":
			var project projectModel
			if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			project.PipelineFile = "pipeline.tf"
			_ = json.NewEncoder(w).Encode(project)
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/demo":
			_ = json.NewEncoder(w).Encode(projectModel{Name: "demo", RepoURL: "https://example.test/repo.git", RepoRef: "drifted"})
		case r.Method == http.MethodPut && r.URL.Path == "/api/projects/demo":
			_ = json.NewEncoder(w).Encode(projectModel{Name: "demo", RepoURL: "https://example.test/repo.git", RepoRef: "release"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/projects/demo":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func testProjectResource(t *testing.T, endpoint string) *projectResource {
	t.Helper()
	client, err := newAPIClient(endpoint, "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return &projectResource{client: client}
}

func demoPlan(t *testing.T) tfsdk.Plan {
	t.Helper()
	schema := testResourceSchema(t)
	return tfsdk.Plan{Schema: schema, Raw: projectRaw(t, schema, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "demo"),
		"repo_url": tftypes.NewValue(tftypes.String, "https://example.test/repo.git"),
		"repo_ref": tftypes.NewValue(tftypes.String, "main"),
	})}
}

func demoState(t *testing.T) tfsdk.State {
	t.Helper()
	return tfsdk.State(demoPlan(t))
}

func emptyState(t *testing.T) tfsdk.State {
	t.Helper()
	schema := testResourceSchema(t)
	return tfsdk.State{Schema: schema, Raw: projectRaw(t, schema, nil)}
}

func TestProjectResourceMetadata(t *testing.T) {
	var resp resource.MetadataResponse
	newProjectResource().Metadata(context.Background(), resource.MetadataRequest{}, &resp)
	if resp.TypeName != "kestrix_project" {
		t.Fatalf("type name = %q", resp.TypeName)
	}
}

func TestProjectResourceConfigure(t *testing.T) {
	ctx := context.Background()

	// Terraform configures resources before the provider on the first pass.
	res := &projectResource{}
	var unconfigured resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{}, &unconfigured)
	requireNoError(t, unconfigured.Diagnostics)
	if res.client != nil {
		t.Fatal("client set without provider data")
	}

	var wrongType resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: "unexpected"}, &wrongType)
	requireError(t, wrongType.Diagnostics, "unexpected provider data type")

	var missingClient resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: &providerData{}}, &missingClient)
	requireError(t, missingClient.Diagnostics, "provider data without a client")

	client, err := newAPIClient("http://example.test", "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var configured resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: &providerData{client: client}}, &configured)
	requireNoError(t, configured.Diagnostics)
	if res.client != client {
		t.Fatal("client was not stored")
	}
}

func TestProjectResourceCreate(t *testing.T) {
	ctx := context.Background()
	res := testProjectResource(t, projectAPI(t).URL)

	resp := resource.CreateResponse{State: emptyState(t)}
	res.Create(ctx, resource.CreateRequest{Plan: demoPlan(t)}, &resp)
	requireNoError(t, resp.Diagnostics)
	var created projectResourceModel
	requireNoError(t, resp.State.Get(ctx, &created))
	if created.PipelineFile.ValueString() != "pipeline.tf" {
		t.Fatalf("server normalization lost: %+v", created)
	}
}

func TestProjectResourceCreateFailures(t *testing.T) {
	ctx := context.Background()
	res := testProjectResource(t, projectAPI(t).URL)

	badSchema := singleAttrResourceSchema()
	var decodeFailure resource.CreateResponse
	decodeFailure.State = emptyState(t)
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: badSchema, Raw: singleAttrRaw(badSchema)}}, &decodeFailure)
	requireError(t, decodeFailure.Diagnostics, "undecodable plan")

	schema := testResourceSchema(t)
	invalid := tfsdk.Plan{Schema: schema, Raw: projectRaw(t, schema, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "demo"),
		"repo_ref": tftypes.NewValue(tftypes.String, "main"),
	})}
	var invalidPlan resource.CreateResponse
	invalidPlan.State = emptyState(t)
	res.Create(ctx, resource.CreateRequest{Plan: invalid}, &invalidPlan)
	requireError(t, invalidPlan.Diagnostics, "plan without a repo_url")

	unreachable := testProjectResource(t, "http://127.0.0.1:1")
	var apiFailure resource.CreateResponse
	apiFailure.State = emptyState(t)
	unreachable.Create(ctx, resource.CreateRequest{Plan: demoPlan(t)}, &apiFailure)
	requireError(t, apiFailure.Diagnostics, "unreachable API")
}

func TestProjectResourceReadRefreshesDrift(t *testing.T) {
	ctx := context.Background()
	res := testProjectResource(t, projectAPI(t).URL)

	resp := resource.ReadResponse{State: demoState(t)}
	res.Read(ctx, resource.ReadRequest{State: demoState(t)}, &resp)
	requireNoError(t, resp.Diagnostics)
	var refreshed projectResourceModel
	requireNoError(t, resp.State.Get(ctx, &refreshed))
	if refreshed.RepoRef.ValueString() != "drifted" {
		t.Fatalf("remote drift not adopted: %+v", refreshed)
	}
}

func TestProjectResourceReadFailures(t *testing.T) {
	ctx := context.Background()

	badSchema := singleAttrResourceSchema()
	res := testProjectResource(t, projectAPI(t).URL)
	var decodeFailure resource.ReadResponse
	decodeFailure.State = emptyState(t)
	res.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: badSchema, Raw: singleAttrRaw(badSchema)}}, &decodeFailure)
	requireError(t, decodeFailure.Diagnostics, "undecodable state")

	missing := testProjectResource(t, httptest.NewServer(http.NotFoundHandler()).URL)
	removed := resource.ReadResponse{State: demoState(t)}
	missing.Read(ctx, resource.ReadRequest{State: demoState(t)}, &removed)
	requireNoError(t, removed.Diagnostics)
	if !removed.State.Raw.IsNull() {
		t.Fatal("missing remote project was not removed from state")
	}

	unreachable := testProjectResource(t, "http://127.0.0.1:1")
	apiFailure := resource.ReadResponse{State: demoState(t)}
	unreachable.Read(ctx, resource.ReadRequest{State: demoState(t)}, &apiFailure)
	requireError(t, apiFailure.Diagnostics, "unreachable API")
}

func TestProjectResourceUpdate(t *testing.T) {
	ctx := context.Background()
	res := testProjectResource(t, projectAPI(t).URL)

	resp := resource.UpdateResponse{State: demoState(t)}
	res.Update(ctx, resource.UpdateRequest{Plan: demoPlan(t)}, &resp)
	requireNoError(t, resp.Diagnostics)
	var updated projectResourceModel
	requireNoError(t, resp.State.Get(ctx, &updated))
	if updated.RepoRef.ValueString() != "release" {
		t.Fatalf("update did not apply server state: %+v", updated)
	}
}

func TestProjectResourceUpdateFailures(t *testing.T) {
	ctx := context.Background()
	res := testProjectResource(t, projectAPI(t).URL)

	badSchema := singleAttrResourceSchema()
	decodeFailure := resource.UpdateResponse{State: emptyState(t)}
	res.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: badSchema, Raw: singleAttrRaw(badSchema)}}, &decodeFailure)
	requireError(t, decodeFailure.Diagnostics, "undecodable plan")

	schema := testResourceSchema(t)
	invalid := tfsdk.Plan{Schema: schema, Raw: projectRaw(t, schema, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "demo"),
		"repo_url": tftypes.NewValue(tftypes.String, "https://example.test/repo.git"),
	})}
	invalidPlan := resource.UpdateResponse{State: emptyState(t)}
	res.Update(ctx, resource.UpdateRequest{Plan: invalid}, &invalidPlan)
	requireError(t, invalidPlan.Diagnostics, "plan without a repo_ref")

	unreachable := testProjectResource(t, "http://127.0.0.1:1")
	apiFailure := resource.UpdateResponse{State: emptyState(t)}
	unreachable.Update(ctx, resource.UpdateRequest{Plan: demoPlan(t)}, &apiFailure)
	requireError(t, apiFailure.Diagnostics, "unreachable API")
}

func TestProjectResourceDelete(t *testing.T) {
	ctx := context.Background()
	res := testProjectResource(t, projectAPI(t).URL)

	var resp resource.DeleteResponse
	resp.State = demoState(t)
	res.Delete(ctx, resource.DeleteRequest{State: demoState(t)}, &resp)
	requireNoError(t, resp.Diagnostics)

	// A project already gone remotely is a completed delete, not an error.
	missing := testProjectResource(t, httptest.NewServer(http.NotFoundHandler()).URL)
	var alreadyGone resource.DeleteResponse
	alreadyGone.State = demoState(t)
	missing.Delete(ctx, resource.DeleteRequest{State: demoState(t)}, &alreadyGone)
	requireNoError(t, alreadyGone.Diagnostics)

	badSchema := singleAttrResourceSchema()
	var decodeFailure resource.DeleteResponse
	decodeFailure.State = emptyState(t)
	res.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: badSchema, Raw: singleAttrRaw(badSchema)}}, &decodeFailure)
	requireError(t, decodeFailure.Diagnostics, "undecodable state")

	unreachable := testProjectResource(t, "http://127.0.0.1:1")
	var apiFailure resource.DeleteResponse
	apiFailure.State = demoState(t)
	unreachable.Delete(ctx, resource.DeleteRequest{State: demoState(t)}, &apiFailure)
	requireError(t, apiFailure.Diagnostics, "unreachable API")
}

func TestProjectResourceImportStateSetsName(t *testing.T) {
	ctx := context.Background()
	resp := resource.ImportStateResponse{State: emptyState(t)}
	newProjectResource().(*projectResource).ImportState(ctx, resource.ImportStateRequest{ID: "demo"}, &resp)
	requireNoError(t, resp.Diagnostics)
	var imported projectResourceModel
	requireNoError(t, resp.State.Get(ctx, &imported))
	if imported.Name.ValueString() != "demo" {
		t.Fatalf("imported name = %q", imported.Name.ValueString())
	}
}

func TestProjectResourceDiagnosticsRedactSecrets(t *testing.T) {
	ctx := context.Background()
	leaking := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(leaking.Close)
	res := testProjectResource(t, leaking.URL+"/token")

	var resp resource.CreateResponse
	resp.State = emptyState(t)
	res.Create(ctx, resource.CreateRequest{Plan: demoPlan(t)}, &resp)
	requireError(t, resp.Diagnostics, "server error")
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Detail(), "token") {
			t.Fatalf("diagnostic leaked a secret marker: %s", d.Detail())
		}
	}
}

func TestToResourceFallsBackToPlanName(t *testing.T) {
	state := toResource(projectModel{RepoURL: "https://example.test/repo.git", RepoRef: "main"}, projectResourceModel{Name: types.StringValue("demo")})
	if state.Name.ValueString() != "demo" {
		t.Fatalf("name fallback = %q", state.Name.ValueString())
	}
}
