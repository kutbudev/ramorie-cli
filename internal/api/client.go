package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/terzigolu/josepshbrain-go/internal/models"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &Client{
		BaseURL: baseURL,
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

	respBody, err := c.makeRequest("POST", "/v1/projects", reqBody)
	if err != nil {
		return nil, err
	}

	var response models.APIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Error)
	}

	projectData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal(projectData, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

func (c *Client) ListProjects() ([]models.Project, error) {
	respBody, err := c.makeRequest("GET", "/v1/projects", nil)
	if err != nil {
		return nil, err
	}

	var response models.ProjectListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error")
	}

	return response.Data, nil
}

func (c *Client) GetProject(id string) (*models.Project, error) {
	respBody, err := c.makeRequest("GET", "/v1/projects/"+id, nil)
	if err != nil {
		return nil, err
	}

	var response models.APIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error: %s", response.Error)
	}

	projectData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal(projectData, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// Task API methods
func (c *Client) CreateTask(projectID, title, description, priority string, tags []string) (*models.Task, error) {
	reqBody := map[string]interface{}{
		"project_id":  projectID,
		"title":       title,
		"description": description,
		"priority":    priority,
		"tags":        tags,
	}

	respBody, err := c.makeRequest("POST", "/v1/tasks", reqBody)
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
	endpoint := "/v1/tasks"
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
	respBody, err := c.makeRequest("GET", "/v1/tasks/"+id, nil)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &task, nil
}

func (c *Client) UpdateTaskStatus(id, status string) (*models.Task, error) {
	reqBody := map[string]string{
		"status": status,
	}

	respBody, err := c.makeRequest("PUT", "/v1/tasks/"+id, reqBody)
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
	_, err := c.makeRequest("DELETE", "/v1/tasks/"+id, nil)
	return err
}

// Memory API methods
func (c *Client) CreateMemory(projectID, content string, tags []string) (*models.Memory, error) {
	reqBody := map[string]interface{}{
		"project_id": projectID,
		"content":    content,
		"tags":       tags,
	}

	respBody, err := c.makeRequest("POST", "/v1/memories", reqBody)
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
	endpoint := "/v1/memories"
	if projectID != "" {
		endpoint += "?project_id=" + projectID
		if search != "" {
			endpoint += "&search=" + search
		}
	} else if search != "" {
		endpoint += "?search=" + search
	}

	respBody, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response models.MemoryListResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error")
	}

	return response.Data, nil
}

// Annotation API methods
func (c *Client) CreateAnnotation(taskID, content string) (*models.Annotation, error) {
	reqBody := map[string]string{
		"content": content,
	}

	url := fmt.Sprintf("/v1/tasks/%s/annotations", taskID)
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
	endpoint := "/v1/annotations"
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