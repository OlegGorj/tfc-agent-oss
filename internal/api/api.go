package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/oleggorj/tfc-agent-oss/internal/models"
)

type APIClient interface {
	RegisterAgent(config *models.AgentConfig) error
	PollForRuns() (*models.RunEvent, error)
	DownloadRunConfigurationVersion(configVer string) (string, error)
	DownloadRunStateVersion(stateVer string) (string, error)
	DownloadRunArtifact(runID, artifactType string) (string, error)
	SaveFile(url, filePath string) error
	UploadRunPlan(runID, filePath string) (string, error)
	UploadRunState(runID, filePath string) (string, error)
	UpdateRunStatus(runID, status string) error
	UpdateRunState(runID string, hasChanges bool) error
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) RegisterAgent(config *models.AgentConfig) error {
	// Add retries for registration
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := c.doRegisterAgent(config)
		if err == nil {
			return nil
		}

		if i < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		return fmt.Errorf("failed to register agent after %d attempts: %w", maxRetries, err)
	}
	return nil
}

func (c *Client) doRegisterAgent(config *models.AgentConfig) error {
	// First, get agent pool ID if not provided
	if config.AgentPoolID == "" {
		poolID, err := c.getDefaultAgentPoolID()
		if err != nil {
			return fmt.Errorf("get default agent pool: %w", err)
		}
		config.AgentPoolID = poolID
	}

	// Create authentication token for the agent
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "authentication-tokens",
			"attributes": map[string]interface{}{
				"description": fmt.Sprintf("Token for agent %s", config.Name),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v2/agent-pools/%s/authentication-tokens",
			c.baseURL, config.AgentPoolID),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Token string `json:"token"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// Store agent ID and update token
	config.AgentID = response.Data.ID
	config.Token = response.Data.Attributes.Token

	return nil
}

func (c *Client) getDefaultAgentPoolID() (string, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v2/organizations/%s/agent-pools",
			c.baseURL, os.Getenv("TF_ORGANIZATION")),
		nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(response.Data) == 0 {
		return "", fmt.Errorf("no agent pools found")
	}

	return response.Data[0].ID, nil
}

func (c *Client) PollForRuns() (*models.RunEvent, error) {
	// Get agent pool ID if not already set
	agentPoolID := os.Getenv("TF_AGENT_POOL_ID")
	if agentPoolID == "" {
		var err error
		agentPoolID, err = c.getDefaultAgentPoolID()
		if err != nil {
			return nil, fmt.Errorf("get default agent pool: %w", err)
		}
	}

	// Create request with proper headers
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v2/agent-pools/%s/runs", // Changed from /tasks to /runs
			c.baseURL,
			agentPoolID),
		nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")

	// Add query parameters for filtering
	q := req.URL.Query()
	q.Add("filter[status]", "pending")
	q.Add("include", "configuration_version,workspace")
	req.URL.RawQuery = q.Encode()

	if os.Getenv("TF_LOG") == "debug" {
		fmt.Printf("Polling URL: %s\n", req.URL.String())
		fmt.Printf("Authorization: Bearer %s\n", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Read body for error reporting and debugging
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if os.Getenv("TF_LOG") == "debug" {
		fmt.Printf("Poll response status: %d\nBody: %s\n", resp.StatusCode, string(body))
	}

	// Handle response
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusNotFound:
		return nil, nil
	case http.StatusOK:
		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/vnd.api+json") {
			return nil, fmt.Errorf("unexpected content type: %s", contentType)
		}

		// Parse response
		var response struct {
			Data struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Attributes struct {
					Status     string    `json:"status"`
					HasChanges bool      `json:"has_changes"`
					CreatedAt  time.Time `json:"created_at"`
					StateVer   string    `json:"state_version"` // Added to match RunEvent model
				} `json:"attributes"`
				Relationships struct {
					Workspace struct {
						Data struct {
							ID   string `json:"id"`
							Type string `json:"type"`
						} `json:"data"`
					} `json:"workspace"`
					ConfigurationVersion struct {
						Data struct {
							ID   string `json:"id"`
							Type string `json:"type"`
						} `json:"data"`
					} `json:"configuration_version"`
					StateVersion struct { // Added to match RunEvent model
						Data struct {
							ID   string `json:"id"`
							Type string `json:"type"`
						} `json:"data"`
					} `json:"state_version"`
				} `json:"relationships"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("parse response: %w, body: %s", err, string(body))
		}

		// Create RunEvent with matching structure
		run := &models.RunEvent{
			ID:   response.Data.ID,
			Type: response.Data.Type,
			Attributes: models.RunEventAttributes{
				Status:     response.Data.Attributes.Status,
				HasChanges: response.Data.Attributes.HasChanges,
				CreatedAt:  response.Data.Attributes.CreatedAt,
				StateVer:   response.Data.Attributes.StateVer,
			},
			Relationships: models.RunEventRelationships{
				Workspace: models.Relationship{
					Data: models.RelationshipData{
						ID:   response.Data.Relationships.Workspace.Data.ID,
						Type: response.Data.Relationships.Workspace.Data.Type,
					},
				},
				ConfigurationVersion: models.Relationship{
					Data: models.RelationshipData{
						ID:   response.Data.Relationships.ConfigurationVersion.Data.ID,
						Type: response.Data.Relationships.ConfigurationVersion.Data.Type,
					},
				},
				StateVersion: models.Relationship{
					Data: models.RelationshipData{
						ID:   response.Data.Relationships.StateVersion.Data.ID,
						Type: response.Data.Relationships.StateVersion.Data.Type,
					},
				},
			},
		}
		return run, nil

	default:
		return nil, fmt.Errorf("unexpected status code: %d, body: %s",
			resp.StatusCode, string(body))
	}
}

func (c *Client) newRequest(method, path string, payload interface{}) (*http.Request, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	return req, nil
}
func (c *Client) DownloadRunArtifact(runID, artifactType string) (string, error) {
	req, err := c.newRequest("GET", fmt.Sprintf("/api/v2/runs/%s/artifacts/%s", runID, artifactType), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var artifact models.Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		return "", err
	}

	return artifact.DownloadURL, nil
}
func (c *Client) DownloadRunStateVersion(runID string) (string, error) {
	req, err := c.newRequest("GET", fmt.Sprintf("/api/v2/runs/%s/state-version", runID), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var stateVersion models.StateVersion
	if err := json.NewDecoder(resp.Body).Decode(&stateVersion); err != nil {
		return "", err
	}

	return stateVersion.DownloadURL, nil
}
func (c *Client) DownloadRunConfigurationVersion(runID string) (string, error) {
	req, err := c.newRequest("GET", fmt.Sprintf("/api/v2/runs/%s/configuration-version", runID), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var configVersion models.ConfigurationVersion
	if err := json.NewDecoder(resp.Body).Decode(&configVersion); err != nil {
		return "", err
	}

	return configVersion.DownloadURL, nil
}
func (c *Client) UpdateRunStatus(runID, status string) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "run",
			"attributes": map[string]interface{}{
				"status": status,
			},
		},
	}

	req, err := c.newRequest("PATCH", fmt.Sprintf("/api/v2/runs/%s", runID), payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
func (c *Client) UpdateRunState(runID string, hasChanges bool) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "run",
			"attributes": map[string]interface{}{
				"has_changes": hasChanges,
			},
		},
	}

	req, err := c.newRequest("PATCH", fmt.Sprintf("/api/v2/runs/%s", runID), payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
func (c *Client) UpdateRunWorkspace(runID, workspaceID string) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "run",
			"attributes": map[string]interface{}{
				"workspace_id": workspaceID,
			},
		},
	}

	req, err := c.newRequest("PATCH", fmt.Sprintf("/api/v2/runs/%s", runID), payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
func (c *Client) UpdateRunWorkspaceName(runID, workspaceName string) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "run",
			"attributes": map[string]interface{}{
				"workspace_name": workspaceName,
			},
		},
	}

	req, err := c.newRequest("PATCH", fmt.Sprintf("/api/v2/runs/%s", runID), payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
func (c *Client) UpdateRunWorkspaceExecMode(runID, execMode string) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "run",
			"attributes": map[string]interface{}{
				"exec_mode": execMode,
			},
		},
	}

	req, err := c.newRequest("PATCH", fmt.Sprintf("/api/v2/runs/%s", runID), payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
func (c *Client) UpdateRunWorkspaceTags(runID, tags string) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "run",
			"attributes": map[string]interface{}{
				"tags": tags,
			},
		},
	}

	req, err := c.newRequest("PATCH", fmt.Sprintf("/api/v2/runs/%s", runID), payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
func (c *Client) UpdateRunWorkspaceID(runID, workspaceID string) error {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "run",
			"attributes": map[string]interface{}{
				"workspace_id": workspaceID,
			},
		},
	}

	req, err := c.newRequest("PATCH", fmt.Sprintf("/api/v2/runs/%s", runID), payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) SaveFile(url, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	err = ioutil.WriteFile(filePath, body, 0644)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// UploadRunPlan uploads a run plan to the server and returns the download URL.
// The filePath parameter is the path to the plan file to be uploaded.
// The runID parameter is the ID of the run to which the plan belongs.
// The function returns the download URL of the uploaded plan.
// If an error occurs, it returns an error.
func (c *Client) UploadRunPlan(runID, filePath string) (string, error) {
	req, err := c.newRequest("POST", fmt.Sprintf("/api/v2/runs/%s/artifacts/plan", runID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filePath))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(filePath)))
	req.Body = io.NopCloser(bytes.NewReader([]byte(filePath)))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var artifact models.Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		return "", err
	}
	return artifact.DownloadURL, nil
}

// UploadRunState uploads a run state to the server and returns the download URL.
// The filePath parameter is the path to the state file to be uploaded.
// The runID parameter is the ID of the run to which the state belongs.
// The function returns the download URL of the uploaded state.
// If an error occurs, it returns an error.
func (c *Client) UploadRunState(runID, filePath string) (string, error) {
	req, err := c.newRequest("POST", fmt.Sprintf("/api/v2/runs/%s/artifacts/state", runID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filePath))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(filePath)))
	req.Body = io.NopCloser(bytes.NewReader([]byte(filePath)))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var artifact models.Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		return "", err
	}
	return artifact.DownloadURL, nil
}
