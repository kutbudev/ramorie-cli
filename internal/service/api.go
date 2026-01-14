package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

const BaseURL = "https://api.ramorie.com/v1"

type APIService struct {
	client *http.Client
	apiKey string
}

func NewAPIService() (*APIService, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key not set. Please run 'ramorie setup' to configure it")
	}

	return &APIService{
		client: &http.Client{},
		apiKey: cfg.APIKey,
	}, nil
}

func (s *APIService) makeRequest(method, url string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	return respBody, nil
}

func (s *APIService) ListTasks(projectID string) ([]models.Task, error) {
	url := fmt.Sprintf("%s/tasks?project_id=%s", BaseURL, projectID)
	body, err := s.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	var tasks []models.Task
	err = json.Unmarshal(body, &tasks)
	return tasks, err
}

func (s *APIService) CreateTask(projectID, title, description, priority string, tags []string) (*models.Task, error) {
	url := fmt.Sprintf("%s/tasks", BaseURL)
	payload := map[string]interface{}{
		"project_id":  projectID,
		"title":       title,
		"description": description,
		"priority":    priority,
		"tags":        tags,
	}
	body, err := s.makeRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	var task models.Task
	err = json.Unmarshal(body, &task)
	return &task, err
}

func (s *APIService) GetTask(id string) (*models.Task, error) {
	url := fmt.Sprintf("%s/tasks/%s", BaseURL, id)
	body, err := s.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	var task models.Task
	err = json.Unmarshal(body, &task)
	return &task, err
}

func (s *APIService) ListProjects() ([]models.Project, error) {
	url := fmt.Sprintf("%s/projects", BaseURL)
	body, err := s.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	var projects []models.Project
	err = json.Unmarshal(body, &projects)
	return projects, err
}

func (s *APIService) CreateProject(name string) (*models.Project, error) {
	url := fmt.Sprintf("%s/projects", BaseURL)
	payload := map[string]string{"name": name}
	body, err := s.makeRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	var project models.Project
	err = json.Unmarshal(body, &project)
	return &project, err
}

func (s *APIService) ListMemories() ([]models.Memory, error) {
	url := fmt.Sprintf("%s/memories", BaseURL)
	body, err := s.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	var memories []models.Memory
	err = json.Unmarshal(body, &memories)
	return memories, err
}

func (s *APIService) CreateMemory(content, projectID string) (*models.Memory, error) {
	url := fmt.Sprintf("%s/memories", BaseURL)
	payload := map[string]string{
		"content":    content,
		"project_id": projectID,
	}
	body, err := s.makeRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	var memory models.Memory
	err = json.Unmarshal(body, &memory)
	return &memory, err
}

func (s *APIService) UpdateTaskStatus(taskID, status string) (*models.Task, error) {
	// This is a placeholder
	return &models.Task{}, nil
}

func (s *APIService) DeleteTask(taskID string) error {
	// This is a placeholder
	return nil
}

func (s *APIService) CreateAnnotation(taskID, content string) (*models.Annotation, error) {
	// This is a placeholder
	return &models.Annotation{}, nil
}
