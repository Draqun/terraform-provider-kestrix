package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIClientProjectLifecycle(t *testing.T) {
	const token = "api-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/projects":
			var project projectModel
			if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
				t.Fatal(err)
			}
			project.PipelineFile = "pipeline.tf"
			_ = json.NewEncoder(w).Encode(project)
		case r.Method == http.MethodGet && r.URL.Path == "/api/projects/demo":
			_ = json.NewEncoder(w).Encode(projectModel{Name: "demo", RepoURL: "https://example.test/repo.git", RepoRef: "main"})
		case r.Method == http.MethodPut && r.URL.Path == "/api/projects/demo":
			_ = json.NewEncoder(w).Encode(projectModel{Name: "demo", RepoURL: "https://example.test/repo.git", RepoRef: "release"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/projects/demo":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, token, "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.createProject(context.Background(), projectModel{Name: "demo", RepoURL: "https://example.test/repo.git", RepoRef: "main"})
	if err != nil || created.PipelineFile != "pipeline.tf" {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	if got, err := client.getProject(context.Background(), "demo"); err != nil || got.Name != "demo" {
		t.Fatalf("get=%+v err=%v", got, err)
	}
	if got, err := client.updateProject(context.Background(), "demo", projectModel{RepoRef: "release"}); err != nil || got.RepoRef != "release" {
		t.Fatalf("update=%+v err=%v", got, err)
	}
	if err := client.deleteProject(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
}

func TestAPIClientErrorsAndValidation(t *testing.T) {
	if _, err := newAPIClient("", "", "", false, time.Second); err == nil {
		t.Fatal("expected empty endpoint error")
	}
	if _, err := newAPIClient("ftp://example.test", "", "", false, time.Second); err == nil {
		t.Fatal("expected endpoint scheme error")
	}
	if _, err := newAPIClient("http://example.test", "", "", false, 0); err == nil {
		t.Fatal("expected timeout error")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"token=api-secret password=hidden"}`))
	}))
	defer server.Close()
	client, err := newAPIClient(server.URL, "api-secret", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getProject(context.Background(), "demo")
	if err == nil || strings.Contains(err.Error(), "api-secret") || strings.Contains(err.Error(), "hidden") {
		t.Fatalf("error leaked secret: %v", err)
	}

	notFound := httptest.NewServer(http.NotFoundHandler())
	defer notFound.Close()
	client, err = newAPIClient(notFound.URL, "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getProject(context.Background(), "missing")
	if err != errNotFound {
		t.Fatalf("not found error = %v", err)
	}
	server5xx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream failure"}`))
	}))
	defer server5xx.Close()
	client, err = newAPIClient(server5xx.URL, "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getProject(context.Background(), "demo")
	if err == nil || !strings.Contains(err.Error(), "502") || strings.Contains(err.Error(), "upstream failure") {
		t.Fatalf("unexpected 5xx error: %v", err)
	}

	malformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer malformed.Close()
	client, err = newAPIClient(malformed.URL, "", "", false, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getProject(context.Background(), "demo")
	if err == nil {
		t.Fatal("expected malformed JSON error")
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer slow.Close()
	client, err = newAPIClient(slow.URL, "", "", false, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getProject(context.Background(), "demo")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
