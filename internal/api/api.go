package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/oleggorj/tfc-agent-oss/internal/models"
)

type APIClient interface {
	RegisterAgent(config *models.AgentConfig) error
	PollForRuns(config *models.AgentConfig) (*models.RunEvent, error)
	DownloadRunConfigurationVersion(configVer string) (string, error)
	DownloadRunStateVersion(stateVer string) (string, error)
	DownloadRunArtifact(runID, artifactType string) (string, error)
	SaveFile(url, filePath string) error
	UploadRunPlan(runID, filePath string) (string, error)
	UploadRunState(runID, filePath string) (string, error)
	UpdateRunStatus(runID, status string) error
	UpdateRunState(runID string, hasChanges bool) error
	DownloadAndSaveConfig(configVer, destPath string) error
}

type Client struct {
	baseURL    string
	token      string
	agentToken string
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
	maxRetries := 3
	var lastErr error

	if err := c.createAgentToken(config); err != nil {
		return fmt.Errorf("create agent token: %w", err)
	}

	for i := 0; i < maxRetries; i++ {
		if err := c.registerAgent(config); err == nil {
			time.Sleep(3 * time.Second) // Wait for 3 seconds before setting status
			if err := c.setAgentStatus(config); err != nil {
				return fmt.Errorf("set agent status: %w", err)
			}

			return nil // Success - exit immediately
		} else {
			lastErr = err
			log.Printf("Registration attempt %d failed: %v", i+1, err)
			if i < maxRetries-1 {
				sleepDuration := time.Second * time.Duration(i+1)
				log.Printf("Retrying in %v...", sleepDuration)
				time.Sleep(sleepDuration)
			}
		}
	}

	return fmt.Errorf("failed to register agent after %d attempts: %w", maxRetries, lastErr)
}

func (c *Client) createAgentToken(config *models.AgentConfig) error {
	log.Printf("Creating authentication token for agent %s", config.Name)

	tokenPayload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "authentication-tokens",
			"attributes": map[string]interface{}{
				"description": fmt.Sprintf("Token for agent %s", config.Name),
			},
		},
	}

	jsonData, err := json.Marshal(tokenPayload)
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
	req.Header.Set("Accept-Encoding", "gzip")

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
		return fmt.Errorf("create token failed: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		Data struct {
			Attributes struct {
				Token string `json:"token"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	log.Printf("Token response: %s", string(body))

	// config.Token = tokenResponse.Data.Attributes.Token
	// config.AgentToken = tokenResponse.Data.Attributes.Token
	c.agentToken = tokenResponse.Data.Attributes.Token
	log.Println("Token created successfully")
	log.Printf("Agent token: %s", tokenResponse.Data.Attributes.Token)
	return nil
}

// Register the agent with the server
// This function sends a POST request to the server with the agent's name and other details.
// It expects the server to respond with a JSON object containing the agent ID and pool ID.
// If the request is successful, it updates the agent's configuration with the received agent ID.
// If the request fails, it returns an error.
func (c *Client) registerAgent(config *models.AgentConfig) error {

	log.Printf("Registering agent %s", config.Name)
	agentPayload := map[string]interface{}{
		"name": config.Name,
	}
	jsonData, err := json.Marshal(agentPayload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/agent/register", c.baseURL),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+c.agentToken)
	req.Header.Set("tfc-agent-version", "1.14.5")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register agent failed: %d, body: %s", resp.StatusCode, string(body))
	} else {
		log.Printf("Agent registration response: %s", string(body))
	}

	// Parse the response to get the agent ID and pool ID
	var agentResponse struct {
		ID     string `json:"id"`
		PoolID string `json:"pool_id"`
	}
	if err := json.Unmarshal(body, &agentResponse); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	log.Printf("Setting Agent ID %s and Pool ID %s", agentResponse.ID, agentResponse.PoolID)
	// Update the config with the agent ID and pool ID
	config.AgentID = agentResponse.ID
	config.AgentPoolID = agentResponse.PoolID
	return nil
}

func (c *Client) setAgentStatus(config *models.AgentConfig) error {
	statusPayload := map[string]interface{}{
		"status": "idle",
	}

	jsonData, err := json.Marshal(statusPayload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("PUT",
		fmt.Sprintf("%s/api/agent/status", c.baseURL),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	// Set headers
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+c.agentToken)
	req.Header.Set("User-Agent", "tfc-agent/1.14.5")
	req.Header.Set("tfc-agent-version", "1.14.5")
	req.Header.Set("tfc-agent-id", config.AgentID)
	req.Header.Set("Accept-Encoding", "gzip")

	if os.Getenv("TF_LOG") == "debug" {
		log.Printf("Setting agent status with token: %s...", c.agentToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("set status failed: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// func (c *Client) getDefaultAgentPoolID() (string, error) {
// 	req, err := http.NewRequest("GET",
// 		fmt.Sprintf("%s/api/v2/organizations/%s/agent-pools",
// 			c.baseURL, os.Getenv("TF_ORGANIZATION")),
// 		nil)
// 	if err != nil {
// 		return "", fmt.Errorf("create request: %w", err)
// 	}
// 	req.Header.Set("Authorization", "Bearer "+c.token)
// 	req.Header.Set("Accept", "application/vnd.api+json")
// 	resp, err := c.httpClient.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("do request: %w", err)
// 	}
// 	defer resp.Body.Close()
// 	var response struct {
// 		Data []struct {
// 			ID string `json:"id"`
// 		} `json:"data"`
// 	}
// 	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
// 		return "", fmt.Errorf("decode response: %w", err)
// 	}
// 	if len(response.Data) == 0 {
// 		return "", fmt.Errorf("no agent pools found")
// 	}
// 	return response.Data[0].ID, nil
// }

func (c *Client) PollForRuns(config *models.AgentConfig) (*models.RunEvent, error) {
	// Create request
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/agent/jobs", c.baseURL),
		nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.agentToken))
	req.Header.Set("User-Agent", "tfc-agent/1.14.5")
	req.Header.Set("tfc-agent-version", "1.14.5")
	req.Header.Set("tfc-agent-id", config.AgentID)
	req.Header.Set("tfc-agent-accept", "plan,apply,policy,assessment,test")

	log.Printf("Setting headers for polling request: %+v ", req.Header)

	if os.Getenv("TF_LOG") == "debug" {
		fmt.Printf("Polling URL: %s\n", req.URL.String())
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Handle 204 No Content as normal polling case
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	// Read body for error reporting and debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if os.Getenv("TF_LOG") == "debug" {
		fmt.Printf("Poll response status: %d\nBody: %s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s",
			resp.StatusCode, string(body))
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
				StateVer   string    `json:"state_version"`
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
				StateVersion struct {
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

func (c *Client) DownloadRunConfigurationVersion(configVer string) (string, error) {
	if configVer == "" {
		return "", fmt.Errorf("configuration version ID is required")
	}

	// Log attempt
	log.Printf("Downloading configuration version %s", configVer)

	// Use correct endpoint for configuration version download
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v2/configuration-versions/%s/download", c.baseURL, configVer),
		nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Authorization", "Bearer "+c.agentToken)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("User-Agent", "tfc-agent/1.14.5")
	req.Header.Set("tfc-agent-version", "1.14.5")

	if os.Getenv("TF_LOG") == "debug" {
		log.Printf("Download URL: %s", req.URL.String())
		log.Printf("Headers: %+v", req.Header)
	}

	// Add retries with exponential backoff
	maxRetries := 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("do request: %w", err)
			continue
		}
		defer resp.Body.Close()

		// Read body for error reporting
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if os.Getenv("TF_LOG") == "debug" {
			log.Printf("Response status: %d", resp.StatusCode)
			log.Printf("Response body: %s", string(body))
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return string(body), nil
		case http.StatusNotFound:
			return "", fmt.Errorf("configuration version %s not found", configVer)
		case http.StatusUnauthorized:
			return "", fmt.Errorf("unauthorized: invalid or expired token")
		default:
			lastErr = fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
			if resp.StatusCode < 500 {
				return "", lastErr
			}
		}

		// Exponential backoff
		if i < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(1<<uint(i)))
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// Helper function to download and save configuration
func (c *Client) DownloadAndSaveConfig(configVer, destPath string) error {
	url, err := c.DownloadRunConfigurationVersion(configVer)
	if err != nil {
		return fmt.Errorf("get download URL: %w", err)
	}

	// Create a request for the actual download
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}

	// Add headers for download
	req.Header.Set("Authorization", "Bearer "+c.agentToken)
	req.Header.Set("User-Agent", "tfc-agent/1.14.5")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Copy downloaded content to file
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	return nil
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
