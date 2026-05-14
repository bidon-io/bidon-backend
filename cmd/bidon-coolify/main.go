package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
)

func main() {
	baseURL := strings.TrimSpace(getFirstNonEmpty(os.Getenv("COOLIFY_BASE_URL")))
	apiKey := strings.TrimSpace(getFirstNonEmpty(os.Getenv("COOLIFY_API_KEY")))

	root := flag.NewFlagSet("bidon-coolify", flag.ExitOnError)
	root.StringVar(&baseURL, "base-url", baseURL, "Coolify base URL, e.g. https://coolify.example.com")
	root.StringVar(&apiKey, "api-key", apiKey, "Coolify API bearer token")
	root.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, `Usage: bidon-coolify [global options] command [command options]

Global options:
  --base-url   Coolify base URL (or COOLIFY_BASE_URL)
  --api-key    Coolify API token (or COOLIFY_API_KEY)

Commands:
  configure-github-repo   Register a GitHub App integration in Coolify
  create-app              Create a Coolify application from a configured repository
  configure-app-env       Set/update application environment variables (inline and/or .env file)
`)
	}

	if err := root.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	args := root.Args()
	if len(args) == 0 {
		root.Usage()
		os.Exit(2)
	}

	client, err := newCoolifyClient(baseURL, apiKey)
	if err != nil {
		log.Fatal(err)
	}

	var runErr error
	switch args[0] {
	case "configure-github-repo":
		runErr = runConfigureGitHubRepo(client, args[1:])
	case "create-app":
		runErr = runCreateApp(client, args[1:])
	case "configure-app-env":
		runErr = runConfigureAppEnv(client, args[1:])
	default:
		root.Usage()
		runErr = fmt.Errorf("unknown command: %s", args[0])
	}

	if runErr != nil {
		log.Fatal(runErr)
	}
}

type coolifyClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newCoolifyClient(baseURL, apiKey string) (*coolifyClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("missing Coolify base URL (set --base-url or COOLIFY_BASE_URL)")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid --base-url: %w", err)
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("missing Coolify API key (set --api-key or COOLIFY_API_KEY)")
	}

	return &coolifyClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil
}

func (c *coolifyClient) postJSON(path string, reqBody any, respBody any) error {
	return c.requestJSON(http.MethodPost, path, reqBody, respBody)
}

func (c *coolifyClient) patchJSON(path string, reqBody any, respBody any) error {
	return c.requestJSON(http.MethodPatch, path, reqBody, respBody)
}

func (c *coolifyClient) requestJSON(method, path string, reqBody any, respBody any) error {
	fullURL := c.baseURL + path
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request payload: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return fmt.Errorf("build HTTP request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s %s failed: status=%d body=%s", method, path, resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}
	if respBody == nil || len(rawBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(rawBody, respBody); err != nil {
		return fmt.Errorf("decode success response: %w", err)
	}

	return nil
}

type configureGitHubRepoRequest struct {
	Name           string `json:"name"`
	Organization   string `json:"organization,omitempty"`
	APIURL         string `json:"api_url,omitempty"`
	HTMLURL        string `json:"html_url,omitempty"`
	CustomUser     string `json:"custom_user,omitempty"`
	CustomPort     int    `json:"custom_port,omitempty"`
	AppID          int    `json:"app_id"`
	InstallationID int    `json:"installation_id"`
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	WebhookSecret  string `json:"webhook_secret"`
	PrivateKeyUUID string `json:"private_key_uuid"`
	IsSystemWide   bool   `json:"is_system_wide"`
}

type configureGitHubRepoResponse struct {
	UUID string `json:"uuid"`
}

func runConfigureGitHubRepo(client *coolifyClient, args []string) error {
	fs := flag.NewFlagSet("configure-github-repo", flag.ContinueOnError)
	var req configureGitHubRepoRequest

	fs.StringVar(&req.Name, "name", "", "Integration name (required)")
	fs.StringVar(&req.Organization, "organization", "", "GitHub organization name")
	fs.StringVar(&req.APIURL, "api-url", "https://api.github.com", "GitHub API URL")
	fs.StringVar(&req.HTMLURL, "html-url", "https://github.com", "GitHub HTML URL")
	fs.StringVar(&req.CustomUser, "custom-user", "", "Optional custom SSH user")
	fs.IntVar(&req.CustomPort, "custom-port", 0, "Optional custom SSH port")
	fs.IntVar(&req.AppID, "app-id", 0, "GitHub App ID (required)")
	fs.IntVar(&req.InstallationID, "installation-id", 0, "GitHub App installation ID (required)")
	fs.StringVar(&req.ClientID, "client-id", "", "GitHub App client ID (required)")
	fs.StringVar(&req.ClientSecret, "client-secret", "", "GitHub App client secret (required)")
	fs.StringVar(&req.WebhookSecret, "webhook-secret", "", "GitHub App webhook secret (required)")
	fs.StringVar(&req.PrivateKeyUUID, "private-key-uuid", "", "Coolify private key UUID for the GitHub App (required)")
	fs.BoolVar(&req.IsSystemWide, "system-wide", true, "Whether integration is system-wide")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if req.Name == "" || req.AppID == 0 || req.InstallationID == 0 || req.ClientID == "" || req.ClientSecret == "" || req.WebhookSecret == "" || req.PrivateKeyUUID == "" {
		return errors.New("missing required flags; required: --name --app-id --installation-id --client-id --client-secret --webhook-secret --private-key-uuid")
	}

	var resp configureGitHubRepoResponse
	if err := client.postJSON("/api/v1/github-apps", req, &resp); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "github_app_uuid=%s\n", resp.UUID)
	return nil
}

type createAppRequest struct {
	ProjectUUID             string `json:"project_uuid"`
	ServerUUID              string `json:"server_uuid"`
	EnvironmentName         string `json:"environment_name,omitempty"`
	EnvironmentUUID         string `json:"environment_uuid,omitempty"`
	GitHubAppUUID           string `json:"github_app_uuid,omitempty"`
	GitRepository           string `json:"git_repository"`
	GitBranch               string `json:"git_branch,omitempty"`
	Name                    string `json:"name"`
	Description             string `json:"description,omitempty"`
	DestinationUUID         string `json:"destination_uuid,omitempty"`
	PortsExposes            string `json:"ports_exposes,omitempty"`
	Domains                 string `json:"domains,omitempty"`
	BaseDirectory           string `json:"base_directory,omitempty"`
	BuildPack               string `json:"build_pack,omitempty"`
	DockerRegistryImageName string `json:"docker_registry_image_name,omitempty"`
	DockerRegistryImageTag  string `json:"docker_registry_image_tag,omitempty"`
	PortsMappings           string `json:"ports_mappings,omitempty"`
	HealthCheckEnabled      bool   `json:"health_check_enabled,omitempty"`
	HealthCheckPath         string `json:"health_check_path,omitempty"`
	HealthCheckPort         string `json:"health_check_port,omitempty"`
	IsAutoDeployEnabled     bool   `json:"is_auto_deploy_enabled,omitempty"`
	IsForceHTTPSEnabled     bool   `json:"is_force_https_enabled,omitempty"`
	UseBuildServer          bool   `json:"use_build_server,omitempty"`
	CustomDockerRunOptions  string `json:"custom_docker_run_options,omitempty"`
}

type createAppResponse struct {
	UUID string `json:"uuid"`
}

func runCreateApp(client *coolifyClient, args []string) error {
	fs := flag.NewFlagSet("create-app", flag.ContinueOnError)
	var req createAppRequest
	var healthCheck bool
	var autoDeploy bool
	var forceHTTPS bool

	fs.StringVar(&req.ProjectUUID, "project-uuid", "", "Coolify project UUID (required)")
	fs.StringVar(&req.ServerUUID, "server-uuid", "", "Coolify server UUID (required)")
	fs.StringVar(&req.EnvironmentName, "environment-name", "production", "Environment name")
	fs.StringVar(&req.EnvironmentUUID, "environment-uuid", "", "Environment UUID (optional)")
	fs.StringVar(&req.GitHubAppUUID, "github-app-uuid", "", "Configured GitHub App UUID; when set, private-repo endpoint is used")
	fs.StringVar(&req.GitRepository, "git-repository", "", "Git repository URL (required)")
	fs.StringVar(&req.GitBranch, "git-branch", "main", "Git branch")
	fs.StringVar(&req.Name, "name", "", "Coolify application name (required)")
	fs.StringVar(&req.Description, "description", "", "Optional description")
	fs.StringVar(&req.DestinationUUID, "destination-uuid", "", "Destination UUID if required by your Coolify setup")
	fs.StringVar(&req.PortsExposes, "ports-exposes", "", "Ports to expose, e.g. 1323")
	fs.StringVar(&req.Domains, "domains", "", "Comma-separated domains")
	fs.StringVar(&req.BaseDirectory, "base-directory", "", "Monorepo subdirectory")
	fs.StringVar(&req.BuildPack, "build-pack", "dockerimage", "Build pack to use (dockerimage recommended for prebuilt images)")
	fs.StringVar(&req.DockerRegistryImageName, "image-name", "", "Container image name, e.g. ghcr.io/bidon-io/bidon-admin")
	fs.StringVar(&req.DockerRegistryImageTag, "image-tag", "latest", "Container image tag")
	fs.StringVar(&req.PortsMappings, "ports-mappings", "", "Port mappings if needed")
	fs.StringVar(&req.HealthCheckPath, "health-check-path", "", "Health check path, e.g. /health")
	fs.StringVar(&req.HealthCheckPort, "health-check-port", "", "Health check port")
	fs.BoolVar(&healthCheck, "health-check-enabled", true, "Enable health checks")
	fs.BoolVar(&autoDeploy, "auto-deploy", true, "Enable auto deploy on push")
	fs.BoolVar(&forceHTTPS, "force-https", true, "Enable force HTTPS")
	fs.BoolVar(&req.UseBuildServer, "use-build-server", false, "Use Coolify build server (default false for bidon)")
	fs.StringVar(&req.CustomDockerRunOptions, "docker-run-options", "--read-only --security-opt no-new-privileges:true", "Custom docker run options for container hardening")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if req.ProjectUUID == "" || req.ServerUUID == "" || req.GitRepository == "" || req.Name == "" {
		return errors.New("missing required flags; required: --project-uuid --server-uuid --git-repository --name")
	}
	if req.BuildPack == "dockerimage" && req.DockerRegistryImageName == "" {
		return errors.New("--image-name is required when --build-pack=dockerimage")
	}

	req.HealthCheckEnabled = healthCheck
	req.IsAutoDeployEnabled = autoDeploy
	req.IsForceHTTPSEnabled = forceHTTPS

	endpoint := "/api/v1/applications/public"
	if strings.TrimSpace(req.GitHubAppUUID) != "" {
		endpoint = "/api/v1/applications/private-github-app"
	}

	var resp createAppResponse
	if err := client.postJSON(endpoint, req, &resp); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "application_uuid=%s\n", resp.UUID)
	return nil
}

type envKVFlag []string

func (f *envKVFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *envKVFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("empty --env entry")
	}
	*f = append(*f, value)
	return nil
}

type envItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsPreview   bool   `json:"is_preview"`
	IsLiteral   bool   `json:"is_literal"`
	IsMultiline bool   `json:"is_multiline"`
	IsShownOnce bool   `json:"is_shown_once"`
}

type updateEnvsRequest struct {
	Data []envItem `json:"data"`
}

func runConfigureAppEnv(client *coolifyClient, args []string) error {
	fs := flag.NewFlagSet("configure-app-env", flag.ContinueOnError)
	var appUUID string
	var envFile string
	var preview bool
	var literal bool
	var multiline bool
	var shownOnce bool
	var kvs envKVFlag

	fs.StringVar(&appUUID, "app-uuid", "", "Coolify application UUID (required)")
	fs.StringVar(&envFile, "env-file", "", "Optional .env file path")
	fs.BoolVar(&preview, "preview", false, "Set preview env vars")
	fs.BoolVar(&literal, "literal", true, "Treat values as literals")
	fs.BoolVar(&multiline, "multiline", false, "Mark values as multiline")
	fs.BoolVar(&shownOnce, "shown-once", false, "Mark values as shown once")
	fs.Var(&kvs, "env", "Inline env in KEY=VALUE form (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if appUUID == "" {
		return errors.New("missing required flag: --app-uuid")
	}
	if len(kvs) == 0 && strings.TrimSpace(envFile) == "" {
		return errors.New("provide at least one --env KEY=VALUE or --env-file")
	}

	values := map[string]string{}
	for _, kv := range kvs {
		k, v, err := splitEnvKV(kv)
		if err != nil {
			return err
		}
		values[k] = v
	}

	if envFile != "" {
		fileValues, err := loadEnvFile(envFile)
		if err != nil {
			return err
		}
		for k, v := range fileValues {
			values[k] = v
		}
	}

	items := make([]envItem, 0, len(values))
	for k, v := range values {
		items = append(items, envItem{
			Key:         k,
			Value:       v,
			IsPreview:   preview,
			IsLiteral:   literal,
			IsMultiline: multiline,
			IsShownOnce: shownOnce,
		})
	}

	req := updateEnvsRequest{Data: items}
	path := fmt.Sprintf("/api/v1/applications/%s/envs/bulk", url.PathEscape(appUUID))
	if err := client.patchJSON(path, req, nil); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "updated_env_count=%d\n", len(items))
	return nil
}

func splitEnvKV(kv string) (string, string, error) {
	parts := strings.SplitN(kv, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid env pair %q; expected KEY=VALUE", kv)
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", fmt.Errorf("invalid env pair %q; empty key", kv)
	}
	return key, parts[1], nil
}

func loadEnvFile(path string) (map[string]string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve env file path: %w", err)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}

	values := map[string]string{}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, err := splitEnvKV(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", resolved, i+1, err)
		}
		values[key] = value
	}
	return values, nil
}

func getFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
