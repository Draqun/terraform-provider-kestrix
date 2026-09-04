package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var errNotFound = errors.New("kestrix project not found")

// systemCertPool is a seam so tests can exercise the fallback taken when the
// host provides no usable system trust store.
var systemCertPool = x509.SystemCertPool

type projectModel struct {
	Name              string `json:"name"`
	RepoURL           string `json:"repo_url"`
	RepoRef           string `json:"repo_ref"`
	RepoCredentialRef string `json:"repo_credential_ref,omitempty"`
	PipelineDir       string `json:"pipeline_dir,omitempty"`
	PipelineFile      string `json:"pipeline_file,omitempty"`
	GitLabProjectID   int64  `json:"gitlab_project_id,omitempty"`
	DefaultBranch     string `json:"default_branch,omitempty"`
	WebhookSecret     string `json:"webhook_secret,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
}

type apiClient struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

func newAPIClient(endpoint, token, caFile string, insecure bool, timeout time.Duration) (*apiClient, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("endpoint must not be empty")
	}
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(endpoint), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("endpoint must be an absolute HTTP(S) URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("endpoint scheme must be http or https")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecure} //nolint:gosec // explicit provider option
	if caFile != "" {
		data, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		pool, err := systemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("CA file contains no certificates")
		}
		tlsConfig.RootCAs = pool
	}
	return &apiClient{
		baseURL: parsed,
		token:   token,
		httpClient: &http.Client{Timeout: timeout, Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		}},
	}, nil
}

func (c *apiClient) projectURL(name string) string {
	return strings.TrimRight(c.baseURL.String(), "/") + "/api/projects/" + url.PathEscape(name)
}

func (c *apiClient) do(ctx context.Context, method, endpoint string, input any, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = strings.NewReader(string(encoded))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to Kestrix API failed: %w", err)
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return fmt.Errorf("read Kestrix API response: %w", readErr)
	}
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Do not copy an arbitrary server response into Terraform diagnostics:
		// proxies and upstream APIs can echo request secrets. The HTTP status is
		// sufficient for a safe, actionable provider error.
		return fmt.Errorf("unexpected Kestrix API status %s", resp.Status)
	}
	if output == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode Kestrix API response: %w", err)
	}
	return nil
}

func (c *apiClient) createProject(ctx context.Context, project projectModel) (projectModel, error) {
	var result projectModel
	err := c.do(ctx, http.MethodPost, strings.TrimRight(c.baseURL.String(), "/")+"/api/projects", project, &result)
	return result, err
}

func (c *apiClient) getProject(ctx context.Context, name string) (projectModel, error) {
	var result projectModel
	err := c.do(ctx, http.MethodGet, c.projectURL(name), nil, &result)
	return result, err
}

func (c *apiClient) updateProject(ctx context.Context, name string, project projectModel) (projectModel, error) {
	var result projectModel
	err := c.do(ctx, http.MethodPut, c.projectURL(name), project, &result)
	return result, err
}

func (c *apiClient) deleteProject(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, c.projectURL(name), nil, nil)
}

func redact(value string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	for _, marker := range []string{"token", "password", "private_key", "webhook_secret"} {
		value = strings.ReplaceAll(value, marker, "[REDACTED]")
	}
	return value
}
