package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/terzigolu/josepshbrain-go/internal/config"
	"github.com/terzigolu/josepshbrain-go/internal/models"
)

var (
	baseURL = "https://jbraincli-go-backend-production.up.railway.app/v1"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
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
func (c *Client) CreateTask(projectID, title, description, priority string) (*models.Task, error) {
	reqBody := map[string]interface{}{
		"project_id":  projectID,
		"title":       title,
		"description": description,
		"priority":    priority,
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

// Memory API methods
func (c *Client) CreateMemory(projectID, content string) (*models.Memory, error) {
	reqBody := map[string]interface{}{
		"project_id": projectID,
		"content":    content,
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

	var memories []models.Memory
	if err := json.Unmarshal(respBody, &memories); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return memories, nil
}

func (c *Client) DeleteMemory(id string) error {
	_, err := c.makeRequest("DELETE", "/memories/"+id, nil)
	return err
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
	reqBody := map[string]interface{}{"name": name}
	respBody, err := c.makeRequest("POST", "/contexts/use", reqBody)
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
	endpoint := "/annotations"
	if taskID != "" {
		endpoint += "?task_id=" + taskID
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response models.AnnotationListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error")
	}

	return response.Data, nil
}

// Auth API methods
func (c *Client) RegisterUser(firstName, lastName, email, password string) (string, error) {
	reqBody := map[string]string{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"password":   password,
	}

	respBody, err := c.makeRequest("POST", "/auth/register", reqBody)
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

	respBody, err := c.makeRequest("POST", "/auth/login", reqBody)
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
