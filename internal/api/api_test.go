package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/oleggorj/tfc-agent-oss/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://app.terraform.io"
	token := os.Getenv("TFC_API_TOKEN")

	client := NewClient(baseURL, token)

	assert.Equal(t, baseURL, client.baseURL)
	assert.Equal(t, token, client.token)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestRegisterAgent(t *testing.T) {
	tests := []struct {
		name       string
		config     *models.AgentConfig
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful registration",
			config: &models.AgentConfig{
				Name: "test-agent",
				Tags: []string{"test"},
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name: "registration failure",
			config: &models.AgentConfig{
				Name: "test-agent",
				Tags: []string{"test"},
			},
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/agents", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			err := client.RegisterAgent(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPollForRuns(t *testing.T) {
	tests := []struct {
		name       string
		response   *models.RunEvent
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful poll with run",
			response: &models.RunEvent{
				ID:     "run-123",
				Status: "pending",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "no runs available",
			response:   nil,
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "server error",
			response:   nil,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/agent-pools/runs/next", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			run, err := client.PollForRuns()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.response == nil {
					assert.Nil(t, run)
				} else {
					assert.Equal(t, tt.response.ID, run.ID)
					assert.Equal(t, tt.response.Status, run.Status)
				}
			}
		})
	}
}

func TestDownloadRunArtifact(t *testing.T) {
	tests := []struct {
		name         string
		runID        string
		artifactType string
		response     *models.Artifact
		statusCode   int
		wantErr      bool
	}{
		{
			name:         "successful download",
			runID:        "run-123",
			artifactType: "plan",
			response: &models.Artifact{
				ID:          "art-123",
				DownloadURL: "https://example.com/artifact",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:         "artifact not found",
			runID:        "run-123",
			artifactType: "plan",
			response:     nil,
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/v2/runs/" + tt.runID + "/artifacts/" + tt.artifactType
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			downloadURL, err := client.DownloadRunArtifact(tt.runID, tt.artifactType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.response.DownloadURL, downloadURL)
			}
		})
	}
}

func TestUpdateRunStatus(t *testing.T) {
	tests := []struct {
		name       string
		runID      string
		status     string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful update",
			runID:      "run-123",
			status:     "completed",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "update failure",
			runID:      "run-123",
			status:     "error",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/runs/"+tt.runID, r.URL.Path)
				assert.Equal(t, "PATCH", r.Method)

				var payload map[string]interface{}
				json.NewDecoder(r.Body).Decode(&payload)

				data := payload["data"].(map[string]interface{})
				assert.Equal(t, "run", data["type"])
				assert.Equal(t, tt.status, data["attributes"].(map[string]interface{})["status"])

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			err := client.UpdateRunStatus(tt.runID, tt.status)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSaveFile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful save",
			content:    "test content",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "download failure",
			content:    "",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.content != "" {
					w.Write([]byte(tt.content))
				}
			}))
			defer server.Close()

			client := NewClient("", "test-token")
			tempFile := t.TempDir() + "/test-file"
			err := client.SaveFile(server.URL, tempFile)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify file contents
				content, err := os.ReadFile(tempFile)
				assert.NoError(t, err)
				assert.Equal(t, tt.content, string(content))
			}
		})
	}
}
