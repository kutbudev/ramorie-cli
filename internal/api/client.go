package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

var (
	baseURL = "https://api.ramorie.com/v1"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string

	// Agent metadata for all requests (set via SetAgentInfo)
	AgentName      string
	AgentModel     string
	AgentSessionID string
}

// SetAgentInfo sets agent metadata that will be included in all subsequent requests.
// This allows all API calls to be attributed to the correct agent session.
func (c *Client) SetAgentInfo(name, model, sessionID string) {
	c.AgentName = name
	c.AgentModel = model
	c.AgentSessionID = sessionID
}

func (c *Client) Request(method, endpoint string, body interface{}) ([]byte, error) {
	return c.makeRequest(method, endpoint, body)
}

// NewClient creates a new API client
func NewClient() *Client {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.ramorie.com/v1"
	}

	// Load API key from config
	cfg, err := config.LoadConfig()
	apiKey := ""
	if err == nil && cfg.APIKey != "" {
		apiKey = cfg.APIKey
	}

	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getAuthBaseURL returns the base URL without /v1 for auth endpoints
func (c *Client) getAuthBaseURL() string {
	// Remove /v1 suffix if present
	baseURL := c.BaseURL
	if strings.HasSuffix(baseURL, "/v1") {
		baseURL = strings.TrimSuffix(baseURL, "/v1")
	}
	return baseURL
}

// makeAuthRequest makes an HTTP request to auth endpoints (at root level, not /v1)
func (c *Client) makeAuthRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := c.getAuthBaseURL() + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// makeRequest makes an HTTP request and returns the response body
func (c *Client) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := c.BaseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add Authorization header if API key is available
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	// Add agent headers if agent info is set (enables event tracking in backend)
	if c.AgentName != "" {
		req.Header.Set("X-Created-Via", "mcp")
		req.Header.Set("X-Agent-Name", c.AgentName)
	}
	if c.AgentModel != "" {
		req.Header.Set("X-Agent-Model", c.AgentModel)
	}
	if c.AgentSessionID != "" {
		req.Header.Set("X-Agent-Session-ID", c.AgentSessionID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Project API methods
func (c *Client) CreateProject(name, description string) (*models.Project, error) {
	reqBody := map[string]string{
		"name":        name,
		"description": description,
	}

	respBody, err := c.makeRequest("POST", "/projects", reqBody)
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal(respBody, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project: %w", err)
	}

	return &project, nil
}

func (c *Client) ListProjects(orgID ...string) ([]models.Project, error) {
	endpoint := "/projects"
	if len(orgID) > 0 && orgID[0] != "" {
		endpoint += "?organization_id=" + orgID[0]
	}
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var projects []models.Project
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return nil, fmt.Errorf("failed to unmarshal projects: %w", err)
	}

	return projects, nil
}

func (c *Client) GetProject(id string) (*models.Project, error) {
	respBody, err := c.makeRequest("GET", "/projects/"+id, nil)
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal(respBody, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project: %w", err)
	}

	return &project, nil
}

func (c *Client) DeleteProject(id string) error {
	_, err := c.makeRequest("DELETE", "/projects/"+id, nil)
	return err
}

// ProjectSuggestion represents a similar project suggestion
type ProjectSuggestion struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Similarity float64 `json:"similarity"`
	OrgName    string  `json:"org_name,omitempty"`
}

// SuggestProjectsResponse represents the response from the project suggest endpoint
type SuggestProjectsResponse struct {
	ExactMatch *models.Project     `json:"exact_match"`
	Similar    []ProjectSuggestion `json:"similar"`
}

// SuggestProjects finds similar existing projects by name
func (c *Client) SuggestProjects(name string, orgID string) (*SuggestProjectsResponse, error) {
	params := url.Values{}
	params.Add("name", name)
	if orgID != "" {
		params.Add("org_id", orgID)
	}

	endpoint := "/projects/suggest?" + params.Encode()
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response SuggestProjectsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal suggest response: %w", err)
	}

	return &response, nil
}

func (c *Client) UpdateProject(id string, data map[string]interface{}) (*models.Project, error) {
	respBody, err := c.makeRequest("PUT", "/projects/"+id, data)
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal(respBody, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project from update response: %w", err)
	}

	return &project, nil
}

// AgentMetadata holds agent information for tracking task creation source
type AgentMetadata struct {
	AgentName  string
	AgentModel string
	SessionID  string
	CreatedVia string
}

// Task API methods
func (c *Client) CreateTask(projectID, title, description, priority string, tags ...string) (*models.Task, error) {
	reqBody := map[string]interface{}{
		"project_id":  projectID,
		"title":       title,
		"description": description,
		"priority":    priority,
	}

	// Add tags if provided
	if len(tags) > 0 {
		reqBody["tags"] = tags
	}

	respBody, err := c.makeRequest("POST", "/tasks", reqBody)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &task, nil
}

// CreateEncryptedTask creates a task with encrypted title and description (zero-knowledge encryption)
func (c *Client) CreateEncryptedTask(projectID, encryptedTitle, titleNonce, encryptedDesc, descNonce, priority string, tags ...string) (*models.Task, error) {
	reqBody := map[string]interface{}{
		"project_id":            projectID,
		"encrypted_title":       encryptedTitle, // base64 ciphertext
		"title_nonce":           titleNonce,     // base64 nonce
		"encrypted_description": encryptedDesc,  // base64 ciphertext
		"description_nonce":     descNonce,      // base64 nonce
		"is_encrypted":          true,
		"priority":              priority,
	}

	// Add tags if provided
	if len(tags) > 0 {
		reqBody["tags"] = tags
	}

	respBody, err := c.makeRequest("POST", "/tasks", reqBody)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &task, nil
}

// CreateEncryptedTaskWithMeta creates an encrypted task with agent metadata for MCP/CLI tracking
func (c *Client) CreateEncryptedTaskWithMeta(projectID, encryptedTitle, titleNonce, priority string, meta *AgentMetadata) (*models.Task, error) {
	reqBody := map[string]interface{}{
		"project_id":      projectID,
		"encrypted_title": encryptedTitle,
		"title_nonce":     titleNonce,
		"is_encrypted":    true,
		"priority":        priority,
	}

	// Add agent metadata if provided
	if meta != nil {
		if meta.AgentName != "" {
			reqBody["created_by_agent"] = meta.AgentName
		}
		if meta.AgentModel != "" {
			reqBody["agent_model"] = meta.AgentModel
		}
		if meta.SessionID != "" {
			reqBody["agent_session_id"] = meta.SessionID
		}
		if meta.CreatedVia != "" {
			reqBody["created_via"] = meta.CreatedVia
		} else {
			reqBody["created_via"] = "mcp"
		}
	}

	respBody, err := c.makeRequest("POST", "/tasks", reqBody)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &task, nil
}

// CreateTaskWithMeta creates a task with agent metadata for MCP/CLI tracking
func (c *Client) CreateTaskWithMeta(projectID, title, description, priority string, meta *AgentMetadata, tags ...string) (*models.Task, error) {
	reqBody := map[string]interface{}{
		"project_id":  projectID,
		"title":       title,
		"description": description,
		"priority":    priority,
	}

	// Add tags if provided
	if len(tags) > 0 {
		reqBody["tags"] = tags
	}

	// Add agent metadata if provided
	if meta != nil {
		if meta.AgentName != "" {
			reqBody["created_by_agent"] = meta.AgentName
		}
		if meta.AgentModel != "" {
			reqBody["agent_model"] = meta.AgentModel
		}
		if meta.SessionID != "" {
			reqBody["agent_session_id"] = meta.SessionID
		}
		if meta.CreatedVia != "" {
			reqBody["created_via"] = meta.CreatedVia
		} else {
			reqBody["created_via"] = "mcp" // Default to 'mcp' for this method
		}
	}

	respBody, err := c.makeRequest("POST", "/tasks", reqBody)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &task, nil
}

func (c *Client) ListTasks(projectID, status string) ([]models.Task, error) {
	endpoint := "/tasks"
	if projectID != "" {
		endpoint += "?project_id=" + projectID
		if status != "" {
			endpoint += "&status=" + status
		}
	} else if status != "" {
		endpoint += "?status=" + status
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Try wrapped response first (backend returns {tasks: [], total: N})
	var wrappedResp struct {
		Tasks []models.Task `json:"tasks"`
		Total int           `json:"total"`
	}
	if err := json.Unmarshal(respBody, &wrappedResp); err == nil && wrappedResp.Tasks != nil {
		return wrappedResp.Tasks, nil
	}

	// Fallback to direct array
	var tasks []models.Task
	if err := json.Unmarshal(respBody, &tasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return tasks, nil
}

func (c *Client) ListTasksQuery(projectID string, status string, q string, priorities []string, tags []string) ([]models.Task, error) {
	endpoint := "/tasks"
	params := url.Values{}
	if strings.TrimSpace(projectID) != "" {
		params.Add("project_id", strings.TrimSpace(projectID))
	}
	if strings.TrimSpace(status) != "" {
		params.Add("status", strings.TrimSpace(status))
	}
	if strings.TrimSpace(q) != "" {
		params.Add("q", strings.TrimSpace(q))
	}
	for _, p := range priorities {
		p = strings.TrimSpace(p)
		if p != "" {
			params.Add("priorities", p)
		}
	}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			params.Add("tags", t)
		}
	}
	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Try wrapped response first (backend returns {tasks: [], total: N})
	var wrappedResp struct {
		Tasks []models.Task `json:"tasks"`
		Total int           `json:"total"`
	}
	if err := json.Unmarshal(respBody, &wrappedResp); err == nil && wrappedResp.Tasks != nil {
		return wrappedResp.Tasks, nil
	}

	// Fallback to direct array
	var tasks []models.Task
	if err := json.Unmarshal(respBody, &tasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return tasks, nil
}

func (c *Client) GetTask(id string) (*models.Task, error) {
	respBody, err := c.makeRequest("GET", "/tasks/"+id, nil)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &task, nil
}

func (c *Client) UpdateTask(id string, data map[string]interface{}) (*models.Task, error) {
	respBody, err := c.makeRequest("PUT", "/tasks/"+id, data)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &task, nil
}

func (c *Client) DeleteTask(id string) error {
	_, err := c.makeRequest("DELETE", "/tasks/"+id, nil)
	return err
}

func (c *Client) StartTask(taskID string) error {
	_, err := c.makeRequest("POST", "/tasks/"+taskID+"/start", nil)
	return err
}

func (c *Client) CompleteTask(taskID string) error {
	_, err := c.makeRequest("POST", "/tasks/"+taskID+"/done", nil)
	return err
}

func (c *Client) StopTask(taskID string) error {
	_, err := c.makeRequest("POST", "/tasks/"+taskID+"/stop", nil)
	return err
}

func (c *Client) GetActiveTask() (*models.Task, error) {
	respBody, err := c.makeRequest("GET", "/tasks/active", nil)
	if err != nil {
		return nil, err
	}

	// Check for empty response (no active task)
	if len(respBody) == 0 || string(respBody) == "{}" || string(respBody) == "null" {
		return nil, nil
	}

	// Backend returns {"active_task": task} or {"active_task": null}
	var response struct {
		ActiveTask *models.Task `json:"active_task"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal active task response: %w", err)
	}

	return response.ActiveTask, nil
}

func (c *Client) ElaborateTask(taskID string) (*models.Annotation, error) {
	endpoint := fmt.Sprintf("/tasks/%s/elaborate", taskID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var annotation models.Annotation
	if err := json.Unmarshal(respBody, &annotation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal annotation from elaborate response: %w", err)
	}

	return &annotation, nil
}

func (c *Client) AINextStep(taskID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/tasks/%s/ai/next-step", taskID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ai next-step response: %w", err)
	}
	if resp.Data == nil {
		resp.Data = map[string]interface{}{}
	}
	return resp.Data, nil
}

func (c *Client) AIEstimateTime(taskID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/tasks/%s/ai/estimate-time", taskID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ai estimate-time response: %w", err)
	}
	if resp.Data == nil {
		resp.Data = map[string]interface{}{}
	}
	return resp.Data, nil
}

func (c *Client) AIRisks(taskID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/tasks/%s/ai/risks", taskID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ai risks response: %w", err)
	}
	if resp.Data == nil {
		resp.Data = map[string]interface{}{}
	}
	return resp.Data, nil
}

func (c *Client) AIDependencies(taskID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/tasks/%s/ai/dependencies", taskID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ai dependencies response: %w", err)
	}
	if resp.Data == nil {
		resp.Data = map[string]interface{}{}
	}
	return resp.Data, nil
}

// Memory API methods
func (c *Client) CreateMemory(projectID, content string, tags ...string) (*models.Memory, error) {
	return c.CreateMemoryWithType(projectID, content, "", tags...)
}

// CreateMemoryWithType creates a memory with an optional type classification
func (c *Client) CreateMemoryWithType(projectID, content, memoryType string, tags ...string) (*models.Memory, error) {
	reqBody := map[string]interface{}{
		"project_id": projectID,
		"content":    content,
	}

	// Add type if provided
	if memoryType != "" {
		reqBody["type"] = memoryType
	}

	// Add tags if provided
	if len(tags) > 0 {
		reqBody["tags"] = tags
	}

	respBody, err := c.makeRequest("POST", "/memories", reqBody)
	if err != nil {
		return nil, err
	}

	var memory models.Memory
	if err := json.Unmarshal(respBody, &memory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &memory, nil
}

// CreateMemoryOptions contains options for creating a memory with temporal and decay support
type CreateMemoryOptions struct {
	ProjectID  string   // Required
	Content    string   // Required
	Type       string   // Optional: memory type
	Tags       []string // Optional: tags
	TTL        int      // Optional: time-to-live in seconds (0 = no expiration)
	ValidFrom  string   // Optional: RFC3339 timestamp when fact became valid
	ValidUntil string   // Optional: RFC3339 timestamp when fact was superseded
}

// CreateMemoryWithOptions creates a memory with full options including TTL and temporal fields
func (c *Client) CreateMemoryWithOptions(opts CreateMemoryOptions) (*models.Memory, error) {
	reqBody := map[string]interface{}{
		"project_id": opts.ProjectID,
		"content":    opts.Content,
	}

	// Add type if provided
	if opts.Type != "" {
		reqBody["type"] = opts.Type
	}

	// Add tags if provided
	if len(opts.Tags) > 0 {
		reqBody["tags"] = opts.Tags
	}

	// Add TTL if provided (for memory decay)
	if opts.TTL > 0 {
		reqBody["ttl"] = opts.TTL
	}

	// Add temporal validity if provided
	if opts.ValidFrom != "" {
		reqBody["valid_from"] = opts.ValidFrom
	}
	if opts.ValidUntil != "" {
		reqBody["valid_until"] = opts.ValidUntil
	}

	respBody, err := c.makeRequest("POST", "/memories", reqBody)
	if err != nil {
		return nil, err
	}

	var memory models.Memory
	if err := json.Unmarshal(respBody, &memory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &memory, nil
}

// CreateEncryptedMemoryOptions contains options for creating an encrypted memory
type CreateEncryptedMemoryOptions struct {
	ProjectID        string   // Required
	EncryptedContent string   // Required: base64 ciphertext
	ContentNonce     string   // Required: base64 nonce
	Tags             []string // Optional
	TTL              int      // Optional: time-to-live in seconds
	ValidFrom        string   // Optional: RFC3339 timestamp
	ValidUntil       string   // Optional: RFC3339 timestamp
}

// CreateEncryptedMemoryWithOptions creates an encrypted memory with TTL and temporal support
func (c *Client) CreateEncryptedMemoryWithOptions(opts CreateEncryptedMemoryOptions) (*models.Memory, error) {
	reqBody := map[string]interface{}{
		"project_id":        opts.ProjectID,
		"encrypted_content": opts.EncryptedContent,
		"content_nonce":     opts.ContentNonce,
		"is_encrypted":      true,
	}

	// Add tags if provided
	if len(opts.Tags) > 0 {
		reqBody["tags"] = opts.Tags
	}

	// Add TTL if provided
	if opts.TTL > 0 {
		reqBody["ttl"] = opts.TTL
	}

	// Add temporal validity if provided
	if opts.ValidFrom != "" {
		reqBody["valid_from"] = opts.ValidFrom
	}
	if opts.ValidUntil != "" {
		reqBody["valid_until"] = opts.ValidUntil
	}

	respBody, err := c.makeRequest("POST", "/memories", reqBody)
	if err != nil {
		return nil, err
	}

	var memory models.Memory
	if err := json.Unmarshal(respBody, &memory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &memory, nil
}

// CreateEncryptedMemory creates a memory with encrypted content (zero-knowledge encryption)
func (c *Client) CreateEncryptedMemory(projectID, encryptedContent, contentNonce string, tags ...string) (*models.Memory, error) {
	reqBody := map[string]interface{}{
		"project_id":        projectID,
		"encrypted_content": encryptedContent, // base64 ciphertext
		"content_nonce":     contentNonce,     // base64 nonce
		"is_encrypted":      true,
	}

	// Add tags if provided
	if len(tags) > 0 {
		reqBody["tags"] = tags
	}

	respBody, err := c.makeRequest("POST", "/memories", reqBody)
	if err != nil {
		return nil, err
	}

	var memory models.Memory
	if err := json.Unmarshal(respBody, &memory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &memory, nil
}

// MemoriesListResponse represents the paginated response from memories endpoint
type MemoriesListResponse struct {
	Memories []models.Memory `json:"memories"`
	Total    int             `json:"total"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
}

func (c *Client) ListMemories(projectID, search string) ([]models.Memory, error) {
	endpoint := "/memories"
	params := url.Values{}
	if projectID != "" {
		params.Add("project_id", projectID)
	}
	if search != "" {
		params.Add("search", search)
	}
	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response MemoriesListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Memories, nil
}

func (c *Client) DeleteMemory(id string) error {
	_, err := c.makeRequest("DELETE", "/memories/"+id, nil)
	return err
}

func (c *Client) UpdateMemory(id string, updates map[string]interface{}) (*models.Memory, error) {
	respBody, err := c.makeRequest("PUT", "/memories/"+id, updates)
	if err != nil {
		return nil, err
	}

	var memory models.Memory
	if err := json.Unmarshal(respBody, &memory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal memory: %w", err)
	}
	return &memory, nil
}

func (c *Client) GetMemory(id string) (*models.Memory, error) {
	respBody, err := c.makeRequest("GET", "/memories/"+id, nil)
	if err != nil {
		return nil, err
	}

	var memory models.Memory
	if err := json.Unmarshal(respBody, &memory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &memory, nil
}

// Job API methods

// EnqueueJob enqueues a background job
func (c *Client) EnqueueJob(jobType string, payload map[string]interface{}) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"type":    jobType,
		"payload": payload,
	}

	respBody, err := c.makeRequest("POST", "/jobs", reqBody)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// Context API methods
func (c *Client) CreateContext(name, description string) (*models.Context, error) {
	reqBody := map[string]interface{}{
		"name":        name,
		"description": description,
	}
	respBody, err := c.makeRequest("POST", "/contexts", reqBody)
	if err != nil {
		return nil, err
	}
	var context models.Context
	if err := json.Unmarshal(respBody, &context); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}
	return &context, nil
}

func (c *Client) ListContexts() ([]models.Context, error) {
	respBody, err := c.makeRequest("GET", "/contexts", nil)
	if err != nil {
		return nil, err
	}
	var contexts []models.Context
	if err := json.Unmarshal(respBody, &contexts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contexts: %w", err)
	}
	return contexts, nil
}

func (c *Client) DeleteContext(id string) error {
	_, err := c.makeRequest("DELETE", "/contexts/"+id, nil)
	return err
}

func (c *Client) UseContext(name string) (*models.Context, error) {
	endpoint := "/contexts/" + url.PathEscape(name) + "/use"
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var context models.Context
	if err := json.Unmarshal(respBody, &context); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}
	return &context, nil
}

// Annotation API methods
func (c *Client) CreateAnnotation(taskID, content string) (*models.Annotation, error) {
	reqBody := map[string]string{
		"content": content,
	}

	url := fmt.Sprintf("/tasks/%s/annotations", taskID)
	respBody, err := c.makeRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	var annotation models.Annotation
	if err := json.Unmarshal(respBody, &annotation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &annotation, nil
}

// CreateEncryptedAnnotation creates an annotation with encrypted content
func (c *Client) CreateEncryptedAnnotation(taskID, encryptedContent, contentNonce string) (*models.Annotation, error) {
	reqBody := map[string]interface{}{
		"encrypted_content": encryptedContent,
		"content_nonce":     contentNonce,
		"is_encrypted":      true,
	}

	url := fmt.Sprintf("/tasks/%s/annotations", taskID)
	respBody, err := c.makeRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	var annotation models.Annotation
	if err := json.Unmarshal(respBody, &annotation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &annotation, nil
}

func (c *Client) ListAnnotations(taskID string) ([]models.Annotation, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	// Backend exposes annotations embedded in task payload
	t, err := c.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	return t.Annotations, nil
}

func (c *Client) BulkUpdateTasks(taskIDs []string, status *string, projectID *string, priority *string) error {
	req := map[string]interface{}{
		"taskIds": taskIDs,
	}
	if status != nil {
		req["status"] = *status
	}
	if projectID != nil {
		req["projectId"] = *projectID
	}
	if priority != nil {
		req["priority"] = *priority
	}
	_, err := c.makeRequest("PUT", "/tasks/bulk-update", req)
	return err
}

func (c *Client) BulkDeleteTasks(taskIDs []string) error {
	req := map[string]interface{}{
		"taskIds": taskIDs,
	}
	_, err := c.makeRequest("POST", "/tasks/bulk-delete", req)
	return err
}

func (c *Client) CreateSubtask(taskID, description string) (*models.Subtask, error) {
	req := map[string]string{"description": description}
	endpoint := fmt.Sprintf("/tasks/%s/subtasks", taskID)
	respBody, err := c.makeRequest("POST", endpoint, req)
	if err != nil {
		return nil, err
	}
	var sub models.Subtask
	if err := json.Unmarshal(respBody, &sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subtask: %w", err)
	}
	return &sub, nil
}

func (c *Client) ListSubtasks(taskID string) ([]models.Subtask, error) {
	endpoint := fmt.Sprintf("/tasks/%s/subtasks", taskID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var subs []models.Subtask
	if err := json.Unmarshal(respBody, &subs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subtasks: %w", err)
	}
	return subs, nil
}

func (c *Client) CreateMemoryTaskLink(taskID, memoryID, relationType string) ([]byte, error) {
	req := map[string]interface{}{
		"task_id":   taskID,
		"memory_id": memoryID,
	}
	if strings.TrimSpace(relationType) != "" {
		req["relation_type"] = relationType
	}
	return c.makeRequest("POST", "/memory-task-links", req)
}

func (c *Client) ListTaskMemories(taskID string) ([]models.Memory, error) {
	endpoint := fmt.Sprintf("/tasks/%s/memories", taskID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var memories []models.Memory
	if err := json.Unmarshal(respBody, &memories); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task memories: %w", err)
	}
	return memories, nil
}

func (c *Client) ListMemoryTasks(memoryID string) ([]models.Task, error) {
	endpoint := fmt.Sprintf("/memories/%s/tasks", memoryID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var tasks []models.Task
	if err := json.Unmarshal(respBody, &tasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal memory tasks: %w", err)
	}
	return tasks, nil
}

// Auth API methods
func (c *Client) RegisterUser(firstName, lastName, email, password string) (string, error) {
	reqBody := map[string]string{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"password":   password,
	}

	// Auth endpoints are at root level, not under /v1
	respBody, err := c.makeAuthRequest("POST", "/auth/register", reqBody)
	if err != nil {
		return "", err
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			APIKey string `json:"api_key"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return "", fmt.Errorf("registration failed: %s", response.Error)
	}

	return response.Data.APIKey, nil
}

func (c *Client) LoginUser(email, password string) (string, error) {
	reqBody := map[string]string{
		"email":    email,
		"password": password,
	}

	// Auth endpoints are at root level, not under /v1
	respBody, err := c.makeAuthRequest("POST", "/auth/login", reqBody)
	if err != nil {
		return "", err
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			APIKey string `json:"api_key"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return "", fmt.Errorf("login failed: %s", response.Error)
	}

	return response.Data.APIKey, nil
}

// EncryptionConfig represents the user's encryption configuration from the server
type EncryptionConfig struct {
	EncryptionEnabled     bool   `json:"encryption_enabled"`
	EncryptedSymmetricKey string `json:"encrypted_symmetric_key"` // base64
	KeyNonce              string `json:"key_nonce"`               // base64 (or stored with encrypted key)
	Salt                  string `json:"salt"`                    // base64
	KDFIterations         int    `json:"kdf_iterations"`
	KDFAlgorithm          string `json:"kdf_algorithm"`
	KDFMemory             int    `json:"kdf_memory,omitempty"`
	KDFParallelism        int    `json:"kdf_parallelism,omitempty"`
	EncryptionVersion     int    `json:"encryption_version,omitempty"`
}

// GetEncryptionConfig fetches the user's encryption configuration
func (c *Client) GetEncryptionConfig() (*EncryptionConfig, error) {
	respBody, err := c.makeRequest("GET", "/auth/encryption-status", nil)
	if err != nil {
		return nil, err
	}

	// API returns direct JSON (not wrapped in success/data)
	var config EncryptionConfig
	if err := json.Unmarshal(respBody, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encryption config: %w", err)
	}

	return &config, nil
}

// Context Pack API methods

// ContextPack represents a context pack
type ContextPack struct {
	ID            string        `json:"id"`
	UserID        string        `json:"user_id"`
	OrgID         *string       `json:"org_id,omitempty"`
	Type          string        `json:"type"`
	Name          string        `json:"name"`
	Description   *string       `json:"description,omitempty"`
	Status        string        `json:"status"`
	Version       int           `json:"version"`
	Tags          []string      `json:"tags"`
	Memories      []interface{} `json:"memories,omitempty"`
	Tasks         []interface{} `json:"tasks,omitempty"`
	Contexts      []interface{} `json:"contexts,omitempty"`
	MemoryIDs     interface{}   `json:"memory_ids,omitempty"`
	TaskIDs       interface{}   `json:"task_ids,omitempty"`
	ContextsCount int           `json:"contexts_count,omitempty"`
	MemoriesCount int           `json:"memories_count,omitempty"`
	TasksCount    int           `json:"tasks_count,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// ContextPackListResponse represents the response from listing context packs
type ContextPackListResponse struct {
	ContextPacks []ContextPack `json:"context_packs"`
	Total        int64         `json:"total"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}

// ListContextPacks lists all context packs with optional filtering
func (c *Client) ListContextPacks(packType, status, query string, limit, offset int) (*ContextPackListResponse, error) {
	endpoint := "/context-packs"
	params := url.Values{}
	if packType != "" {
		params.Add("type", packType)
	}
	if status != "" {
		params.Add("status", status)
	}
	if query != "" {
		params.Add("q", query)
	}
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Add("offset", fmt.Sprintf("%d", offset))
	}
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response ContextPackListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context packs: %w", err)
	}
	return &response, nil
}

// GetContextPack gets a specific context pack by ID
func (c *Client) GetContextPack(id string) (*ContextPack, error) {
	endpoint := fmt.Sprintf("/context-packs/%s", id)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var pack ContextPack
	if err := json.Unmarshal(respBody, &pack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context pack: %w", err)
	}
	return &pack, nil
}

// CreateContextPack creates a new context pack
func (c *Client) CreateContextPack(name, packType, description, status string, tags []string) (*ContextPack, error) {
	reqBody := map[string]interface{}{
		"name": name,
		"type": packType,
	}
	if description != "" {
		reqBody["description"] = description
	}
	if status != "" {
		reqBody["status"] = status
	}
	if len(tags) > 0 {
		reqBody["tags"] = tags
	}

	respBody, err := c.makeRequest("POST", "/context-packs", reqBody)
	if err != nil {
		return nil, err
	}

	var pack ContextPack
	if err := json.Unmarshal(respBody, &pack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context pack: %w", err)
	}
	return &pack, nil
}

// UpdateContextPack updates an existing context pack
func (c *Client) UpdateContextPack(id string, updates map[string]interface{}) (*ContextPack, error) {
	endpoint := fmt.Sprintf("/context-packs/%s", id)
	respBody, err := c.makeRequest("PUT", endpoint, updates)
	if err != nil {
		return nil, err
	}

	var pack ContextPack
	if err := json.Unmarshal(respBody, &pack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context pack: %w", err)
	}
	return &pack, nil
}

// DeleteContextPack deletes a context pack
func (c *Client) DeleteContextPack(id string) error {
	endpoint := fmt.Sprintf("/context-packs/%s", id)
	_, err := c.makeRequest("DELETE", endpoint, nil)
	return err
}

// UseContextPack activates a context pack and all its contexts
func (c *Client) UseContextPack(id string) (*ContextPack, error) {
	endpoint := fmt.Sprintf("/context-packs/%s/use", id)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Response is { "message": "...", "pack": {...} }
	var response struct {
		Message string      `json:"message"`
		Pack    ContextPack `json:"pack"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context pack: %w", err)
	}
	return &response.Pack, nil
}

// GetActiveContextPack gets the currently active context pack
func (c *Client) GetActiveContextPack() (*ContextPack, error) {
	respBody, err := c.makeRequest("GET", "/context-packs/active", nil)
	if err != nil {
		return nil, err
	}

	// Response is { "pack": {...} or null, "message": "..." }
	var response struct {
		Pack    *ContextPack `json:"pack"`
		Message string       `json:"message"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal active context pack: %w", err)
	}
	return response.Pack, nil
}

// SetActiveContextPack sets the active context pack (alias for UseContextPack)
func (c *Client) SetActiveContextPack(id string) (*ContextPack, error) {
	return c.UseContextPack(id)
}

// AddMemoryToPack adds a memory to a context pack
func (c *Client) AddMemoryToPack(packID, memoryID string) (*ContextPack, error) {
	endpoint := fmt.Sprintf("/context-packs/%s/memories/%s", packID, memoryID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var pack ContextPack
	if err := json.Unmarshal(respBody, &pack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context pack: %w", err)
	}
	return &pack, nil
}

// AddTaskToPack adds a task to a context pack
func (c *Client) AddTaskToPack(packID, taskID string) (*ContextPack, error) {
	endpoint := fmt.Sprintf("/context-packs/%s/tasks/%s", packID, taskID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var pack ContextPack
	if err := json.Unmarshal(respBody, &pack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context pack: %w", err)
	}
	return &pack, nil
}

// RemoveMemoryFromPack removes a memory from a context pack
func (c *Client) RemoveMemoryFromPack(packID, memoryID string) error {
	endpoint := fmt.Sprintf("/context-packs/%s/memories/%s", packID, memoryID)
	_, err := c.makeRequest("DELETE", endpoint, nil)
	return err
}

// RemoveTaskFromPack removes a task from a context pack
func (c *Client) RemoveTaskFromPack(packID, taskID string) error {
	endpoint := fmt.Sprintf("/context-packs/%s/tasks/%s", packID, taskID)
	_, err := c.makeRequest("DELETE", endpoint, nil)
	return err
}

// Decision API methods

// Decision represents an architectural decision record (ADR)
type Decision struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ProjectID    *string   `json:"project_id,omitempty"`
	ADRNumber    string    `json:"adr_number"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	Area         string    `json:"area"`
	Content      *string   `json:"content,omitempty"`
	Context      *string   `json:"context,omitempty"`
	Consequences *string   `json:"consequences,omitempty"`
	Source       string    `json:"source"` // "user", "agent", "import"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DecisionListResponse represents the response from listing decisions
type DecisionListResponse struct {
	Decisions []Decision `json:"decisions"`
	Total     int64      `json:"total"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// ListDecisions lists all decisions with optional filtering
func (c *Client) ListDecisions(status, area string, limit int) ([]Decision, error) {
	endpoint := "/decisions"
	params := url.Values{}
	if status != "" {
		params.Add("status", status)
	}
	if area != "" {
		params.Add("area", area)
	}
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response DecisionListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decisions: %w", err)
	}
	return response.Decisions, nil
}

// GetDecision gets a specific decision by ID or ADR number
func (c *Client) GetDecision(identifier string) (*Decision, error) {
	endpoint := fmt.Sprintf("/decisions/%s", identifier)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var decision Decision
	if err := json.Unmarshal(respBody, &decision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decision: %w", err)
	}
	return &decision, nil
}

// CreateDecision creates a new decision (ADR)
func (c *Client) CreateDecision(title, description, status, area, context, consequences, source string) (*Decision, error) {
	reqBody := map[string]interface{}{
		"title": title,
	}
	if description != "" {
		reqBody["description"] = description
	}
	if status != "" {
		reqBody["status"] = status
	}
	if area != "" {
		reqBody["area"] = area
	}
	if context != "" {
		reqBody["context"] = context
	}
	if consequences != "" {
		reqBody["consequences"] = consequences
	}
	if source != "" {
		reqBody["source"] = source
	}

	respBody, err := c.makeRequest("POST", "/decisions", reqBody)
	if err != nil {
		return nil, err
	}

	var decision Decision
	if err := json.Unmarshal(respBody, &decision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decision: %w", err)
	}
	return &decision, nil
}

// UpdateDecision updates an existing decision
func (c *Client) UpdateDecision(id string, updates map[string]interface{}) (*Decision, error) {
	endpoint := fmt.Sprintf("/decisions/%s", id)
	respBody, err := c.makeRequest("PUT", endpoint, updates)
	if err != nil {
		return nil, err
	}

	var decision Decision
	if err := json.Unmarshal(respBody, &decision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decision: %w", err)
	}
	return &decision, nil
}

// DeleteDecision deletes a decision
func (c *Client) DeleteDecision(id string) error {
	endpoint := fmt.Sprintf("/decisions/%s", id)
	_, err := c.makeRequest("DELETE", endpoint, nil)
	return err
}

// User Focus API methods (SINGLE SOURCE OF TRUTH for active workspace)

// UserFocus represents the user's current focus state
type UserFocus struct {
	ActiveContextPackID *string          `json:"active_context_pack_id"`
	ActivePack          *FocusPackDetail `json:"active_pack"`
}

// FocusPackDetail represents a context pack in focus response
type FocusPackDetail struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   *string               `json:"description,omitempty"`
	Type          string                `json:"type"`
	Status        string                `json:"status"`
	ContextsCount int                   `json:"contexts_count"`
	MemoriesCount int                   `json:"memories_count"`
	TasksCount    int                   `json:"tasks_count"`
	Contexts      []FocusContextPreview `json:"contexts"`
}

// FocusContextPreview represents a context preview in focus response
type FocusContextPreview struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetFocus returns the user's current focus (active context pack)
func (c *Client) GetFocus() (*UserFocus, error) {
	respBody, err := c.makeRequest("GET", "/me/focus", nil)
	if err != nil {
		return nil, err
	}

	var focus UserFocus
	if err := json.Unmarshal(respBody, &focus); err != nil {
		return nil, fmt.Errorf("failed to unmarshal focus: %w", err)
	}
	return &focus, nil
}

// SetFocus sets the user's active context pack
func (c *Client) SetFocus(contextPackID string) (*UserFocus, error) {
	reqBody := map[string]interface{}{
		"context_pack_id": contextPackID,
	}

	respBody, err := c.makeRequest("POST", "/me/focus", reqBody)
	if err != nil {
		return nil, err
	}

	// Response is { "message": "...", "focus": {...} }
	var response struct {
		Message string    `json:"message"`
		Focus   UserFocus `json:"focus"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal focus: %w", err)
	}
	return &response.Focus, nil
}

// ClearFocus clears the user's active context pack
func (c *Client) ClearFocus() error {
	_, err := c.makeRequest("DELETE", "/me/focus", nil)
	return err
}

// Organization API methods

// Organization represents an organization
type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListOrganizations lists all organizations for the user
func (c *Client) ListOrganizations() ([]Organization, error) {
	respBody, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, err
	}

	var orgs []Organization
	if err := json.Unmarshal(respBody, &orgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organizations: %w", err)
	}
	return orgs, nil
}

// GetOrganization gets a specific organization by ID
func (c *Client) GetOrganization(id string) (*Organization, error) {
	endpoint := fmt.Sprintf("/organizations/%s", id)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var org Organization
	if err := json.Unmarshal(respBody, &org); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization: %w", err)
	}
	return &org, nil
}

// CreateOrganization creates a new organization
func (c *Client) CreateOrganization(name, description string) (*Organization, error) {
	reqBody := map[string]interface{}{
		"name": name,
	}
	if description != "" {
		reqBody["description"] = description
	}

	respBody, err := c.makeRequest("POST", "/organizations", reqBody)
	if err != nil {
		return nil, err
	}

	var org Organization
	if err := json.Unmarshal(respBody, &org); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization: %w", err)
	}
	return &org, nil
}

// UpdateOrganization updates an organization
func (c *Client) UpdateOrganization(id string, updates map[string]interface{}) (*Organization, error) {
	endpoint := fmt.Sprintf("/organizations/%s", id)
	respBody, err := c.makeRequest("PUT", endpoint, updates)
	if err != nil {
		return nil, err
	}

	var org Organization
	if err := json.Unmarshal(respBody, &org); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization: %w", err)
	}
	return &org, nil
}

// OrganizationMember represents a member of an organization
type OrganizationMember struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id"`
	Role      string    `json:"role"` // owner, admin, member
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// GetOrganizationMembers lists members of an organization
func (c *Client) GetOrganizationMembers(orgID string) ([]OrganizationMember, error) {
	endpoint := fmt.Sprintf("/organizations/%s/members", orgID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var members []OrganizationMember
	if err := json.Unmarshal(respBody, &members); err != nil {
		return nil, fmt.Errorf("failed to unmarshal members: %w", err)
	}
	return members, nil
}

// InviteToOrganization invites a user to an organization by email
func (c *Client) InviteToOrganization(orgID, email, role string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/organizations/%s/invite", orgID)
	reqBody := map[string]interface{}{
		"email": email,
	}
	if role != "" {
		reqBody["role"] = role
	}

	respBody, err := c.makeRequest("POST", endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal invite response: %w", err)
	}
	return result, nil
}

// SwitchOrganization switches the active organization context
func (c *Client) SwitchOrganization(orgID string) (*Organization, error) {
	// Backend endpoint: POST /auth/active-organization
	// Returns: UserResponse with active_organization_id set
	reqBody := map[string]interface{}{
		"organization_id": orgID,
	}
	respBody, err := c.makeRequest("POST", "/auth/active-organization", reqBody)
	if err != nil {
		return nil, err
	}

	// Backend returns UserResponse, not Organization directly
	// Parse to verify success, then get full organization details
	var userResp map[string]interface{}
	if err := json.Unmarshal(respBody, &userResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Verify active_organization_id was set correctly
	if activeOrgID, ok := userResp["active_organization_id"].(string); ok {
		if activeOrgID != orgID {
			return nil, fmt.Errorf("failed to switch organization: active_organization_id mismatch")
		}
	}

	// Get the full organization details
	return c.GetOrganization(orgID)
}

// GetActiveOrganization gets the currently active organization
func (c *Client) GetActiveOrganization() (*Organization, error) {
	respBody, err := c.makeRequest("GET", "/me/organization", nil)
	if err != nil {
		return nil, err
	}

	var org Organization
	if err := json.Unmarshal(respBody, &org); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization: %w", err)
	}
	return &org, nil
}

// LeaveOrganization leaves the specified organization
func (c *Client) LeaveOrganization(orgID string) error {
	endpoint := fmt.Sprintf("/organizations/%s/leave", orgID)
	_, err := c.makeRequest("POST", endpoint, nil)
	return err
}

// ============================================================================
// Agent Events API
// ============================================================================

// AgentEventItem represents a single agent event in the timeline
type AgentEventItem struct {
	ID             string  `json:"id"`
	EventType      string  `json:"event_type"`
	EntityType     string  `json:"entity_type"`
	EntityID       string  `json:"entity_id"`
	AgentName      string  `json:"agent_name"`
	AgentModel     *string `json:"agent_model,omitempty"`
	AgentSessionID *string `json:"agent_session_id,omitempty"`
	CreatedVia     string  `json:"created_via"`
	ProjectID      *string `json:"project_id,omitempty"`
	ProjectName    *string `json:"project_name,omitempty"`
	EntityTitle    *string `json:"entity_title,omitempty"`
	EntityPreview  *string `json:"entity_preview,omitempty"`
	CreatedAt      string  `json:"created_at"`
	IsEncrypted    bool    `json:"is_encrypted"`
}

// AgentEventListResponse represents the paginated response from agent events API
type AgentEventListResponse struct {
	Events      []AgentEventItem `json:"events"`
	NextCursor  string           `json:"next_cursor,omitempty"`
	HasMore     bool             `json:"has_more"`
	Total       int64            `json:"total_estimate,omitempty"`
	QueryTimeMs int64            `json:"query_time_ms,omitempty"`
}

// AgentEventFilter contains filter parameters for listing events
type AgentEventFilter struct {
	ProjectID  string
	AgentName  string
	EventType  string
	EntityType string
	Limit      int
}

// GetAgentEvents retrieves agent activity events with optional filtering
func (c *Client) GetAgentEvents(filter AgentEventFilter) (*AgentEventListResponse, error) {
	endpoint := "/agent-events"
	params := url.Values{}

	if filter.ProjectID != "" {
		params.Add("project_id", filter.ProjectID)
	}
	if filter.AgentName != "" {
		params.Add("agent_name", filter.AgentName)
	}
	if filter.EventType != "" {
		params.Add("event_type", filter.EventType)
	}
	if filter.EntityType != "" {
		params.Add("entity_type", filter.EntityType)
	}
	if filter.Limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", filter.Limit))
	} else {
		params.Add("limit", "20") // Default limit
	}

	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response AgentEventListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent events: %w", err)
	}
	return &response, nil
}

// ============================================================================
// Task Dependencies API
// ============================================================================

// TaskDependency represents a dependency between two tasks
type TaskDependency struct {
	ID          string    `json:"id"`
	TaskID      string    `json:"task_id"`
	DependsOnID string    `json:"depends_on_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskDependencyInfo contains dependency info with task details
type TaskDependencyInfo struct {
	ID              string    `json:"id"`
	TaskID          string    `json:"task_id"`
	TaskTitle       string    `json:"task_title,omitempty"`
	TaskStatus      string    `json:"task_status,omitempty"`
	DependsOnID     string    `json:"depends_on_id"`
	DependsOnTitle  string    `json:"depends_on_title,omitempty"`
	DependsOnStatus string    `json:"depends_on_status,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// AddTaskDependency creates a dependency where taskID depends on dependsOnID
func (c *Client) AddTaskDependency(taskID, dependsOnID string) (*TaskDependency, error) {
	endpoint := fmt.Sprintf("/tasks/%s/dependencies", taskID)
	body := map[string]string{
		"depends_on_id": dependsOnID,
	}

	respBody, err := c.makeRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	var dep TaskDependency
	if err := json.Unmarshal(respBody, &dep); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task dependency: %w", err)
	}
	return &dep, nil
}

// GetTaskDependencies returns all tasks that the specified task depends on (prerequisites)
func (c *Client) GetTaskDependencies(taskID string) ([]TaskDependencyInfo, error) {
	endpoint := fmt.Sprintf("/tasks/%s/dependencies", taskID)

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var deps []TaskDependencyInfo
	if err := json.Unmarshal(respBody, &deps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task dependencies: %w", err)
	}
	return deps, nil
}

// GetTaskDependents returns all tasks that depend on the specified task
func (c *Client) GetTaskDependents(taskID string) ([]TaskDependencyInfo, error) {
	endpoint := fmt.Sprintf("/tasks/%s/dependents", taskID)

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var deps []TaskDependencyInfo
	if err := json.Unmarshal(respBody, &deps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task dependents: %w", err)
	}
	return deps, nil
}

// RemoveTaskDependency removes a dependency between two tasks
func (c *Client) RemoveTaskDependency(taskID, dependsOnID string) error {
	endpoint := fmt.Sprintf("/tasks/%s/dependencies/%s", taskID, dependsOnID)

	_, err := c.makeRequest("DELETE", endpoint, nil)
	return err
}

// CheckTaskDependencyCycle checks if adding a dependency would create a circular dependency
func (c *Client) CheckTaskDependencyCycle(taskID, dependsOnID string) (bool, error) {
	endpoint := fmt.Sprintf("/tasks/%s/dependencies/%s/check-cycle", taskID, dependsOnID)

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return false, err
	}

	var result struct {
		WouldCreateCycle bool `json:"would_create_cycle"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("failed to unmarshal cycle check result: %w", err)
	}
	return result.WouldCreateCycle, nil
}

// ============================================================================
// Subtasks API (Enhanced)
// ============================================================================

// UpdateSubtaskRequest is the request body for updating a subtask
type UpdateSubtaskRequest struct {
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	Completed   *int    `json:"completed,omitempty"`
}

// UpdateSubtask updates an existing subtask
func (c *Client) UpdateSubtask(subtaskID string, req UpdateSubtaskRequest) (*models.Subtask, error) {
	endpoint := fmt.Sprintf("/subtasks/%s", subtaskID)

	respBody, err := c.makeRequest("PATCH", endpoint, req)
	if err != nil {
		return nil, err
	}

	var subtask models.Subtask
	if err := json.Unmarshal(respBody, &subtask); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subtask: %w", err)
	}
	return &subtask, nil
}

// DeleteSubtask deletes a subtask
func (c *Client) DeleteSubtask(subtaskID string) error {
	endpoint := fmt.Sprintf("/subtasks/%s", subtaskID)
	_, err := c.makeRequest("DELETE", endpoint, nil)
	return err
}

// CompleteSubtask marks a subtask as completed
func (c *Client) CompleteSubtask(subtaskID string) (*models.Subtask, error) {
	completed := 1
	return c.UpdateSubtask(subtaskID, UpdateSubtaskRequest{
		Completed: &completed,
	})
}

// ============================================================================
// Plan API
// ============================================================================

// Plan represents a plan run
type Plan struct {
	ID                  string       `json:"id"`
	UserID              string       `json:"user_id"`
	OrganizationID      *string      `json:"organization_id,omitempty"`
	ProjectID           *string      `json:"project_id,omitempty"`
	Title               string       `json:"title"`
	Requirements        string       `json:"requirements"`
	Type                string       `json:"type"`
	Status              string       `json:"status"`
	CurrentPhase        string       `json:"current_phase"`
	Progress            int          `json:"progress"`
	MaxBudgetUSD        *float64     `json:"max_budget_usd,omitempty"`
	SpentBudgetUSD      float64      `json:"spent_budget_usd"`
	TokensUsed          int64        `json:"tokens_used"`
	FinalConsensusScore *float64     `json:"final_consensus_score,omitempty"`
	TaskCount           int          `json:"task_count"`
	ADRCount            int          `json:"adr_count"`
	RiskCount           int          `json:"risk_count"`
	Error               string       `json:"error,omitempty"`
	CreatedAt           time.Time    `json:"created_at"`
	StartedAt           *time.Time   `json:"started_at,omitempty"`
	CompletedAt         *time.Time   `json:"completed_at,omitempty"`
	Phases              []PlanPhase  `json:"phases,omitempty"`
}

// PlanPhase represents a phase in a plan run
type PlanPhase struct {
	ID              string    `json:"id"`
	PlanRunID       string    `json:"plan_run_id"`
	Phase           string    `json:"phase"`
	Status          string    `json:"status"`
	Sequence        int       `json:"sequence"`
	ConsensusScore  *float64  `json:"consensus_score,omitempty"`
	WinningProposal *string   `json:"winning_proposal,omitempty"`
	TokensUsed      int64     `json:"tokens_used"`
	DurationMs      int64     `json:"duration_ms"`
	CostUSD         float64   `json:"cost_usd"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// PlanArtifact represents an artifact from a plan run
type PlanArtifact struct {
	ID        string    `json:"id"`
	PlanRunID string    `json:"plan_run_id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// PlanRisk represents a risk identified in a plan
type PlanRisk struct {
	ID          string    `json:"id"`
	PlanRunID   string    `json:"plan_run_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Severity    string    `json:"severity"`
	Likelihood  *float64  `json:"likelihood,omitempty"`
	Impact      *float64  `json:"impact,omitempty"`
	Mitigation  string    `json:"mitigation,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// PlanConfiguration contains plan configuration options
type PlanConfiguration struct {
	ConsensusThreshold float64 `json:"consensus_threshold,omitempty"`
	ProposalAgentCount int     `json:"proposal_agent_count,omitempty"`
	PreferredModel     string  `json:"preferred_model,omitempty"`
	BudgetTokens       int64   `json:"budget_tokens,omitempty"`
	BudgetUSD          float64 `json:"budget_usd,omitempty"`
}

// CreatePlanRequest is the request body for creating a plan
type CreatePlanRequest struct {
	Title         string             `json:"title"`
	Requirements  string             `json:"requirements"`
	ProjectID     string             `json:"project_id,omitempty"`
	Type          string             `json:"type,omitempty"`
	Configuration *PlanConfiguration `json:"configuration,omitempty"`
	ContextBundle map[string]interface{} `json:"context_bundle,omitempty"`
}

// ListPlansFilter contains filter parameters for listing plans
type ListPlansFilter struct {
	Status    string
	Type      string
	ProjectID string
	Limit     int
	Offset    int
}

// PlanListResponse represents the paginated response from plans API
type PlanListResponse struct {
	Plans  []Plan `json:"plans"`
	Total  int64  `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// ApplyPlanRequest is the request body for applying plan results
type ApplyPlanRequest struct {
	ApplyTasks bool   `json:"apply_tasks"`
	ApplyADRs  bool   `json:"apply_adrs"`
	TaskStatus string `json:"task_status,omitempty"`
	ADRStatus  string `json:"adr_status,omitempty"`
}

// ApplyPlanResponse is the response from applying plan results
type ApplyPlanResponse struct {
	TasksCreated int      `json:"tasks_created"`
	ADRsCreated  int      `json:"adrs_created"`
	TaskIDs      []string `json:"task_ids,omitempty"`
	ADRIDs       []string `json:"adr_ids,omitempty"`
}

// CreatePlan creates a new plan run
func (c *Client) CreatePlan(req CreatePlanRequest) (*Plan, error) {
	respBody, err := c.makeRequest("POST", "/plans", req)
	if err != nil {
		return nil, err
	}

	var plan Plan
	if err := json.Unmarshal(respBody, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &plan, nil
}

// ListPlans lists plan runs with optional filtering
func (c *Client) ListPlans(filter ListPlansFilter) ([]Plan, error) {
	endpoint := "/plans"
	params := url.Values{}

	if filter.Status != "" {
		params.Add("status", filter.Status)
	}
	if filter.Type != "" {
		params.Add("type", filter.Type)
	}
	if filter.ProjectID != "" {
		params.Add("project_id", filter.ProjectID)
	}
	if filter.Limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", filter.Limit))
	}
	if filter.Offset > 0 {
		params.Add("offset", fmt.Sprintf("%d", filter.Offset))
	}

	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response PlanListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plans: %w", err)
	}
	return response.Plans, nil
}

// GetPlan gets a specific plan by ID (supports partial ID matching)
func (c *Client) GetPlan(id string) (*Plan, error) {
	endpoint := fmt.Sprintf("/plans/%s", id)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var plan Plan
	if err := json.Unmarshal(respBody, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &plan, nil
}

// StartPlan starts execution of a pending plan
func (c *Client) StartPlan(id string) (*Plan, error) {
	endpoint := fmt.Sprintf("/plans/%s/start", id)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var plan Plan
	if err := json.Unmarshal(respBody, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &plan, nil
}

// CancelPlan cancels a running plan
func (c *Client) CancelPlan(id string) error {
	endpoint := fmt.Sprintf("/plans/%s/cancel", id)
	_, err := c.makeRequest("POST", endpoint, nil)
	return err
}

// ResumePlan resumes a failed or cancelled plan
func (c *Client) ResumePlan(id string) (*Plan, error) {
	endpoint := fmt.Sprintf("/plans/%s/resume", id)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var plan Plan
	if err := json.Unmarshal(respBody, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &plan, nil
}

// ApplyPlan applies plan results (creates tasks and/or ADRs)
func (c *Client) ApplyPlan(id string, req ApplyPlanRequest) (*ApplyPlanResponse, error) {
	endpoint := fmt.Sprintf("/plans/%s/apply", id)
	respBody, err := c.makeRequest("POST", endpoint, req)
	if err != nil {
		return nil, err
	}

	var response ApplyPlanResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal apply response: %w", err)
	}
	return &response, nil
}

// UpdatePlan updates a plan's properties
func (c *Client) UpdatePlan(id string, updates map[string]interface{}) (*Plan, error) {
	endpoint := fmt.Sprintf("/plans/%s", id)
	respBody, err := c.makeRequest("PUT", endpoint, updates)
	if err != nil {
		return nil, err
	}

	var plan Plan
	if err := json.Unmarshal(respBody, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &plan, nil
}

// DeletePlan deletes a plan
func (c *Client) DeletePlan(id string) error {
	endpoint := fmt.Sprintf("/plans/%s", id)
	_, err := c.makeRequest("DELETE", endpoint, nil)
	return err
}

// GetPlanPhases gets all phases for a plan
func (c *Client) GetPlanPhases(planID string) ([]PlanPhase, error) {
	endpoint := fmt.Sprintf("/plans/%s/phases", planID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var phases []PlanPhase
	if err := json.Unmarshal(respBody, &phases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phases: %w", err)
	}
	return phases, nil
}

// ListPlanArtifacts lists all artifacts for a plan
func (c *Client) ListPlanArtifacts(planID string) ([]PlanArtifact, error) {
	endpoint := fmt.Sprintf("/plans/%s/artifacts", planID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var artifacts []PlanArtifact
	if err := json.Unmarshal(respBody, &artifacts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artifacts: %w", err)
	}
	return artifacts, nil
}

// GetPlanArtifact gets a specific artifact by type
func (c *Client) GetPlanArtifact(planID, artifactType string) (*PlanArtifact, error) {
	endpoint := fmt.Sprintf("/plans/%s/artifacts/%s", planID, artifactType)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var artifact PlanArtifact
	if err := json.Unmarshal(respBody, &artifact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artifact: %w", err)
	}
	return &artifact, nil
}

// ListPlanRisks lists all risks for a plan
func (c *Client) ListPlanRisks(planID string) ([]PlanRisk, error) {
	endpoint := fmt.Sprintf("/plans/%s/risks", planID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var risks []PlanRisk
	if err := json.Unmarshal(respBody, &risks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal risks: %w", err)
	}
	return risks, nil
}

// --- Organization Encryption API ---

// OrgEncryptionConfig represents the org encryption configuration from server
type OrgEncryptionConfig struct {
	OrganizationID    string `json:"organization_id"`
	Salt              string `json:"salt"`
	KDFAlgorithm      string `json:"kdf_algorithm"`
	KDFIterations     int    `json:"kdf_iterations"`
	EncryptionVersion int    `json:"encryption_version"`
	IsEnabled         bool   `json:"is_enabled"`
}

// OrgEncryptionStatus represents the encryption status with member info
type OrgEncryptionStatus struct {
	IsEnabled         bool   `json:"is_enabled"`
	EncryptionVersion int    `json:"encryption_version"`
	SetupBy           string `json:"setup_by,omitempty"`
	SetupAt           string `json:"setup_at,omitempty"`
}

// OrgWrappedKey represents the wrapped org key for auto-unlock
type OrgWrappedKey struct {
	WrappedOrgKey string `json:"wrapped_org_key"`
	KeyNonce      string `json:"key_nonce"`
	KeyVersion    int    `json:"key_version"`
}

// GetOrgEncryptionConfig fetches the org encryption configuration (salt, KDF params)
func (c *Client) GetOrgEncryptionConfig(orgID string) (*OrgEncryptionConfig, error) {
	endpoint := fmt.Sprintf("/organizations/%s/encryption/config", orgID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var config OrgEncryptionConfig
	if err := json.Unmarshal(respBody, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal org encryption config: %w", err)
	}
	return &config, nil
}

// GetOrgEncryptionStatus gets the encryption status for an org
func (c *Client) GetOrgEncryptionStatus(orgID string) (*OrgEncryptionStatus, error) {
	endpoint := fmt.Sprintf("/organizations/%s/encryption/status", orgID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var status OrgEncryptionStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal org encryption status: %w", err)
	}
	return &status, nil
}

// SetupOrgEncryption initializes encryption for an organization
func (c *Client) SetupOrgEncryption(orgID, salt, passphraseHash string, kdfIterations int) error {
	endpoint := fmt.Sprintf("/organizations/%s/encryption/setup", orgID)
	body := map[string]interface{}{
		"salt":            salt,
		"passphrase_hash": passphraseHash,
		"kdf_iterations":  kdfIterations,
	}
	_, err := c.makeRequest("POST", endpoint, body)
	return err
}

// VerifyOrgPassphrase verifies the passphrase hash with the server
func (c *Client) VerifyOrgPassphrase(orgID, passphraseHash string) (bool, error) {
	endpoint := fmt.Sprintf("/organizations/%s/encryption/verify", orgID)
	body := map[string]interface{}{
		"passphrase_hash": passphraseHash,
	}
	respBody, err := c.makeRequest("POST", endpoint, body)
	if err != nil {
		return false, err
	}

	var result struct {
		Verified bool `json:"verified"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("failed to unmarshal verify response: %w", err)
	}
	return result.Verified, nil
}

// StoreOrgWrappedKey stores the wrapped org key for auto-unlock
func (c *Client) StoreOrgWrappedKey(orgID, wrappedKey, keyNonce string) error {
	endpoint := fmt.Sprintf("/organizations/%s/encryption/store-key", orgID)
	body := map[string]interface{}{
		"wrapped_org_key": wrappedKey,
		"key_nonce":       keyNonce,
	}
	_, err := c.makeRequest("POST", endpoint, body)
	return err
}

// GetOrgWrappedKey fetches the user's wrapped org key for auto-unlock
func (c *Client) GetOrgWrappedKey(orgID string) (*OrgWrappedKey, error) {
	endpoint := fmt.Sprintf("/organizations/%s/encryption/wrapped-key", orgID)
	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var wrapped OrgWrappedKey
	if err := json.Unmarshal(respBody, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wrapped key: %w", err)
	}
	return &wrapped, nil
}

// RotateOrgEncryption rotates the org encryption key (new salt + hash, bumps version)
func (c *Client) RotateOrgEncryption(orgID, newSalt, newPassphraseHash string, newKDFIterations int) error {
	endpoint := fmt.Sprintf("/organizations/%s/encryption/rotate", orgID)
	body := map[string]interface{}{
		"salt":            newSalt,
		"passphrase_hash": newPassphraseHash,
		"kdf_iterations":  newKDFIterations,
	}
	_, err := c.makeRequest("POST", endpoint, body)
	return err
}
