package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/oleggorj/tfc-agent-oss/internal/models"
)

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
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "agent",
			"attributes": map[string]interface{}{
				"name": config.Name,
				"tags": config.Tags,
			},
		},
	}

	req, err := c.newRequest("POST", "/api/v2/agents", payload)
	if err != nil {
		return fmt.Errorf("create register request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("register agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) PollForRuns() (*models.RunEvent, error) {
	req, err := c.newRequest("GET", "/api/v2/agent-pools/runs/next", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var runEvent models.RunEvent
	if err := json.NewDecoder(resp.Body).Decode(&runEvent); err != nil {
		return nil, err
	}

	return &runEvent, nil
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
