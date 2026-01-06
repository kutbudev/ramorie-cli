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
	baseURL = "https://jbraincli-go-backend-production.up.railway.app/v1"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
}

func (c *Client) Request(method, endpoint string, body interface{}) ([]byte, error) {
	return c.makeRequest(method, endpoint, body)
}

// NewClient creates a new API client
func NewClient() *Client {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://jbraincli-go-backend-production.up.railway.app/v1"
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

func (c *Client) ListProjects() ([]models.Project, error) {
	respBody, err := c.makeRequest("GET", "/projects", nil)
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

func (c *Client) SetProjectActive(id string) error {
	_, err := c.makeRequest("POST", "/projects/"+id+"/use", nil)
	return err
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
	reqBody := map[string]interface{}{
		"project_id": projectID,
		"content":    content,
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

// Context Pack API methods

// ContextPack represents a context pack
type ContextPack struct {
	ID             string        `json:"id"`
	UserID         string        `json:"user_id"`
	OrgID          *string       `json:"org_id,omitempty"`
	Type           string        `json:"type"`
	Name           string        `json:"name"`
	Description    *string       `json:"description,omitempty"`
	Status         string        `json:"status"`
	Version        int           `json:"version"`
	Tags           []string      `json:"tags"`
	Memories       []interface{} `json:"memories,omitempty"`
	Tasks          []interface{} `json:"tasks,omitempty"`
	Contexts       []interface{} `json:"contexts,omitempty"`
	MemoryIDs      interface{}   `json:"memory_ids,omitempty"`
	TaskIDs        interface{}   `json:"task_ids,omitempty"`
	ContextsCount  int           `json:"contexts_count,omitempty"`
	MemoriesCount  int           `json:"memories_count,omitempty"`
	TasksCount     int           `json:"tasks_count,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
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
	ActiveContextPackID *string         `json:"active_context_pack_id"`
	ActivePack          *FocusPackDetail `json:"active_pack"`
}

// FocusPackDetail represents a context pack in focus response
type FocusPackDetail struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Description   *string              `json:"description,omitempty"`
	Type          string               `json:"type"`
	Status        string               `json:"status"`
	ContextsCount int                  `json:"contexts_count"`
	MemoriesCount int                  `json:"memories_count"`
	TasksCount    int                  `json:"tasks_count"`
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
	endpoint := fmt.Sprintf("/organizations/%s/switch", orgID)
	respBody, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var org Organization
	if err := json.Unmarshal(respBody, &org); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization: %w", err)
	}
	return &org, nil
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
