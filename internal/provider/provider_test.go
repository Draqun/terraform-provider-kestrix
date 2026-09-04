package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderMetadataAndSchema(t *testing.T) {
	p := &Provider{version: "test"}
	var metadata provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &metadata)
	if metadata.TypeName != "kestrix" || metadata.Version != "test" {
		t.Fatalf("metadata = %+v", metadata)
	}
	var schemaResponse provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResponse)
	if len(schemaResponse.Schema.Attributes) != 5 {
		t.Fatalf("provider attributes = %d", len(schemaResponse.Schema.Attributes))
	}
}

func TestProjectValidationAndRedaction(t *testing.T) {
	if err := validatePlan(projectResourceModel{}); err == nil {
		t.Fatal("expected required field validation")
	}
	if got := redact("token=secret", "secret"); got == "token=secret" {
		t.Fatal("secret was not redacted")
	}
}
