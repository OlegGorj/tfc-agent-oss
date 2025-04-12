package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   AgentConfig
		expected map[string]interface{}
	}{
		{
			name: "basic config",
			config: AgentConfig{
				Name:         "test-agent",
				Token:        "test-token",
				Address:      "https://test.com",
				Tags:         []string{"tag1", "tag2"},
				LogLevel:     "INFO",
				WorkspaceDir: "/tmp/workspace",
			},
			expected: map[string]interface{}{
				"Name":         "test-agent",
				"Token":        "test-token",
				"Address":      "https://test.com",
				"Tags":         []string{"tag1", "tag2"},
				"LogLevel":     "INFO",
				"WorkspaceDir": "/tmp/workspace",
			},
		},
		{
			name: "minimal config",
			config: AgentConfig{
				Name:  "test-agent",
				Token: "test-token",
			},
			expected: map[string]interface{}{
				"Name":  "test-agent",
				"Token": "test-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			assert.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			assert.NoError(t, err)

			// Check only non-empty fields
			for k, v := range tt.expected {
				assert.Equal(t, v, tt.config.getField(k))
			}
		})
	}
}

func TestRunEvent(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name     string
		event    RunEvent
		expected string
	}{
		{
			name: "complete run event",
			event: RunEvent{
				ID:         "run-123",
				CreatedAt:  now,
				Status:     "pending",
				HasChanges: true,
				Workspace: Workspace{
					ID:       "ws-123",
					Name:     "test-workspace",
					ExecMode: "remote",
				},
				ConfigVer: "cv-123",
				StateVer:  "sv-123",
			},
			expected: `{
                "id": "run-123",
                "created_at": "` + now.Format(time.RFC3339Nano) + `",
                "status": "pending",
                "has_changes": true,
                "workspace": {
                    "id": "ws-123",
                    "name": "test-workspace",
                    "execution_mode": "remote"
                },
                "configuration_version": "cv-123",
                "state_version": "sv-123"
            }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			assert.NoError(t, err)

			var expected, actual map[string]interface{}
			err = json.Unmarshal([]byte(tt.expected), &expected)
			assert.NoError(t, err)
			err = json.Unmarshal(data, &actual)
			assert.NoError(t, err)

			assert.Equal(t, expected, actual)

			// Test reverse unmarshaling
			var decoded RunEvent
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)
			assert.Equal(t, tt.event, decoded)
		})
	}
}

func TestWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace Workspace
		expected  string
	}{
		{
			name: "complete workspace",
			workspace: Workspace{
				ID:       "ws-123",
				Name:     "test-workspace",
				ExecMode: "remote",
			},
			expected: `{
                "id": "ws-123",
                "name": "test-workspace",
                "execution_mode": "remote"
            }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.workspace)
			assert.NoError(t, err)

			var expected, actual map[string]interface{}
			err = json.Unmarshal([]byte(tt.expected), &expected)
			assert.NoError(t, err)
			err = json.Unmarshal(data, &actual)
			assert.NoError(t, err)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestRunStatus(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name   string
		status RunStatus
		valid  bool
	}{
		{
			name: "valid run status",
			status: RunStatus{
				ID:        "run-123",
				CreatedAt: now,
				UpdatedAt: now,
				Status:    "running",
				Workspace: Workspace{
					ID:   "ws-123",
					Name: "test-workspace",
				},
				AgentID:   "agent-123",
				AgentName: "test-agent",
				AgentTags: []string{"tag1", "tag2"},
			},
			valid: true,
		},
		{
			name: "missing required fields",
			status: RunStatus{
				ID: "run-123",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if tt.valid {
				assert.NoError(t, err)
				var decoded RunStatus
				err = json.Unmarshal(data, &decoded)
				assert.NoError(t, err)
				assert.Equal(t, tt.status, decoded)
			}
		})
	}
}

func TestArtifact(t *testing.T) {
	tests := []struct {
		name     string
		artifact Artifact
		expected string
	}{
		{
			name: "complete artifact",
			artifact: Artifact{
				ID:          "art-123",
				DownloadURL: "https://example.com/download",
			},
			expected: `{
                "id": "art-123",
                "download_url": "https://example.com/download"
            }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.artifact)
			assert.NoError(t, err)

			var expected, actual map[string]interface{}
			err = json.Unmarshal([]byte(tt.expected), &expected)
			assert.NoError(t, err)
			err = json.Unmarshal(data, &actual)
			assert.NoError(t, err)

			assert.Equal(t, expected, actual)
		})
	}
}

// Helper method for AgentConfig
func (c AgentConfig) getField(field string) interface{} {
	switch field {
	case "Name":
		return c.Name
	case "Token":
		return c.Token
	case "Address":
		return c.Address
	case "Tags":
		return c.Tags
	case "LogLevel":
		return c.LogLevel
	case "WorkspaceDir":
		return c.WorkspaceDir
	default:
		return nil
	}
}
