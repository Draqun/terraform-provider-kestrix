package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	stringplanmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type projectResource struct{ client *apiClient }

type projectResourceModel struct {
	Name              types.String `tfsdk:"name"`
	RepoURL           types.String `tfsdk:"repo_url"`
	RepoRef           types.String `tfsdk:"repo_ref"`
	RepoCredentialRef types.String `tfsdk:"repo_credential_ref"`
	PipelineDir       types.String `tfsdk:"pipeline_dir"`
	PipelineFile      types.String `tfsdk:"pipeline_file"`
	GitLabProjectID   types.Int64  `tfsdk:"gitlab_project_id"`
	DefaultBranch     types.String `tfsdk:"default_branch"`
	WebhookSecret     types.String `tfsdk:"webhook_secret"`
}

func newProjectResource() resource.Resource { return &projectResource{} }

func (r *projectResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kestrix_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{Attributes: map[string]resourceschema.Attribute{
		"name":                resourceschema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Stable Kestrix project name."},
		"repo_url":            resourceschema.StringAttribute{Required: true, Description: "Remote Git repository URL."},
		"repo_ref":            resourceschema.StringAttribute{Required: true, Description: "Git branch, tag, or commit to use."},
		"repo_credential_ref": resourceschema.StringAttribute{Optional: true, Description: "Reference to an encrypted Kestrix repository credential."},
		// Kestrix normalizes these on create (pipeline_dir, pipeline_file and
		// default_branch get defaults, gitlab_project_id defaults to 0), so they
		// must be Computed as well or Terraform rejects the applied state as
		// inconsistent with a null plan.
		"pipeline_dir":      resourceschema.StringAttribute{Optional: true, Computed: true, Description: "Directory containing the pipeline file."},
		"pipeline_file":     resourceschema.StringAttribute{Optional: true, Computed: true, Description: "Pipeline file name."},
		"gitlab_project_id": resourceschema.Int64Attribute{Optional: true, Computed: true, Description: "GitLab project ID for dispatch."},
		"default_branch":    resourceschema.StringAttribute{Optional: true, Computed: true, Description: "Default branch for webhooks."},
		"webhook_secret":    resourceschema.StringAttribute{Optional: true, Sensitive: true, Description: "Webhook verification secret."},
	}}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok || data.client == nil {
		resp.Diagnostics.AddError("Invalid provider data", "Kestrix project resource received invalid provider configuration.")
		return
	}
	r.client = data.client
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validatePlan(plan); err != nil {
		resp.Diagnostics.AddError("Invalid Kestrix project", err.Error())
		return
	}
	created, err := r.client.createProject(ctx, fromResource(plan))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Create project failed", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, toResource(created, plan))...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project, err := r.client.getProject(ctx, state.Name.ValueString())
	if errors.Is(err, errNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Read project failed", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, toResource(project, state))...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validatePlan(plan); err != nil {
		resp.Diagnostics.AddError("Invalid Kestrix project", err.Error())
		return
	}
	updated, err := r.client.updateProject(ctx, plan.Name.ValueString(), fromResource(plan))
	if err != nil {
		addAPIError(&resp.Diagnostics, "Update project failed", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, toResource(updated, plan))...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.deleteProject(ctx, state.Name.ValueString())
	if errors.Is(err, errNotFound) {
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "Delete project failed", err)
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func validatePlan(plan projectResourceModel) error {
	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		return fmt.Errorf("name must not be empty")
	}
	if plan.RepoURL.IsNull() || plan.RepoURL.ValueString() == "" {
		return fmt.Errorf("repo_url is required")
	}
	if plan.RepoRef.IsNull() || plan.RepoRef.ValueString() == "" {
		return fmt.Errorf("repo_ref is required")
	}
	return nil
}

func fromResource(p projectResourceModel) projectModel {
	return projectModel{Name: p.Name.ValueString(), RepoURL: p.RepoURL.ValueString(), RepoRef: p.RepoRef.ValueString(), RepoCredentialRef: p.RepoCredentialRef.ValueString(), PipelineDir: p.PipelineDir.ValueString(), PipelineFile: p.PipelineFile.ValueString(), GitLabProjectID: p.GitLabProjectID.ValueInt64(), DefaultBranch: p.DefaultBranch.ValueString(), WebhookSecret: p.WebhookSecret.ValueString()}
}

func toResource(p projectModel, fallback projectResourceModel) projectResourceModel {
	if p.Name == "" {
		p.Name = fallback.Name.ValueString()
	}
	return projectResourceModel{Name: types.StringValue(p.Name), RepoURL: types.StringValue(p.RepoURL), RepoRef: types.StringValue(p.RepoRef), RepoCredentialRef: optionalString(p.RepoCredentialRef, fallback.RepoCredentialRef), PipelineDir: optionalString(p.PipelineDir, fallback.PipelineDir), PipelineFile: optionalString(p.PipelineFile, fallback.PipelineFile), GitLabProjectID: types.Int64Value(p.GitLabProjectID), DefaultBranch: optionalString(p.DefaultBranch, fallback.DefaultBranch), WebhookSecret: optionalString(p.WebhookSecret, fallback.WebhookSecret)}
}

func optionalString(value string, fallback types.String) types.String {
	if value != "" {
		return types.StringValue(value)
	}
	// A computed attribute the caller left unset is unknown in the plan, and
	// state may never hold an unknown value.
	if fallback.IsUnknown() {
		return types.StringNull()
	}
	return fallback
}

func addAPIError(diags *diag.Diagnostics, summary string, err error) {
	diags.AddError(summary, redact(err.Error()))
}

var _ resource.Resource = (*projectResource)(nil)
