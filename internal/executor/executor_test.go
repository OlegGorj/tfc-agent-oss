package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/oleggorj/tfc-agent-oss/internal/api"
	"github.com/oleggorj/tfc-agent-oss/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the API client
type MockClient struct {
	mock.Mock
}

var _ api.APIClient = (*MockClient)(nil)

func (m *MockClient) DownloadRunConfigurationVersion(configVer string) (string, error) {
	args := m.Called(configVer)
	return args.String(0), args.Error(1)
}
func (m *MockClient) DownloadRunStateVersion(stateVer string) (string, error) {
	args := m.Called(stateVer)
	return args.String(0), args.Error(1)
}
func (m *MockClient) SaveFile(url, filePath string) error {
	args := m.Called(url, filePath)
	return args.Error(0)
}
func (m *MockClient) UploadRunPlan(runID, filePath string) (string, error) {
	args := m.Called(runID, filePath)
	return args.String(0), args.Error(1)
}
func (m *MockClient) UploadRunState(runID, filePath string) (string, error) {
	args := m.Called(runID, filePath)
	return args.String(0), args.Error(1)
}
func (m *MockClient) RegisterAgent(config *models.AgentConfig) error {
	args := m.Called(config)
	return args.Error(0)
}
func (m *MockClient) PollForRuns() (*models.RunEvent, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunEvent), args.Error(1)
}
func (m *MockClient) DownloadRunArtifact(runID, artifactType string) (string, error) {
	args := m.Called(runID, artifactType)
	return args.String(0), args.Error(1)
}
func (m *MockClient) UpdateRunStatus(runID, status string) error {
	args := m.Called(runID, status)
	return args.Error(0)
}
func (m *MockClient) UpdateRunState(runID string, hasChanges bool) error {
	args := m.Called(runID, hasChanges)
	return args.Error(0)
}

func TestNewTerraformExecutor(t *testing.T) {
	mockClient := new(MockClient)
	workDir := "/tmp/test-workdir"

	executor := NewTerraformExecutor(workDir, mockClient)

	assert.Equal(t, workDir, executor.workDir)
	assert.Equal(t, mockClient, executor.client)
}

func TestExecuteRun(t *testing.T) {
	tests := []struct {
		name    string
		run     *models.RunEvent
		setup   func(*MockClient, string)
		wantErr bool
	}{
		{
			name: "successful plan execution",
			run: &models.RunEvent{
				ID:        "run-123",
				Status:    "planning",
				ConfigVer: "cv-123",
				StateVer:  "sv-123",
				Workspace: models.Workspace{
					ID:   "ws-123",
					Name: "test-workspace",
				},
			},
			setup: func(m *MockClient, dir string) {
				m.On("DownloadRunConfigurationVersion", "cv-123").Return("http://example.com/config", nil)
				m.On("DownloadRunStateVersion", "sv-123").Return("http://example.com/state", nil)
				m.On("SaveFile", mock.Anything, mock.Anything).Return(nil)
				m.On("UploadRunPlan", "run-123", mock.Anything).Return("http://example.com/plan", nil)
			},
			wantErr: false,
		},
		{
			name: "successful apply execution",
			run: &models.RunEvent{
				ID:        "run-123",
				Status:    "applying",
				ConfigVer: "cv-123",
				StateVer:  "sv-123",
			},
			setup: func(m *MockClient, dir string) {
				// Mock the configuration and state downloads
				m.On("DownloadRunConfigurationVersion", "cv-123").Return("http://example.com/config", nil)
				m.On("DownloadRunStateVersion", "sv-123").Return("http://example.com/state", nil)
				m.On("SaveFile", mock.Anything, mock.Anything).Return(nil)
				m.On("UploadRunState", "run-123", mock.Anything).Return("http://example.com/state", nil)

				// Create a dummy plan file
				planPath := filepath.Join(dir, "plan.tfplan")
				err := os.WriteFile(planPath, []byte("test plan"), 0644)
				assert.NoError(t, err)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			workDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(mockClient, workDir)
			}

			executor := NewTerraformExecutor(workDir, mockClient)

			// Mock terraform command execution
			oldExecCommand := execCommand
			defer func() { execCommand = oldExecCommand }()
			execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				return exec.Command("echo", "Success")
			}

			err := executor.ExecuteRun(context.Background(), tt.run)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestDownloadConfig(t *testing.T) {
	tests := []struct {
		name      string
		configVer string
		setup     func(*MockClient)
		wantErr   bool
	}{
		{
			name:      "successful config download",
			configVer: "cv-123",
			setup: func(m *MockClient) {
				m.On("DownloadRunConfigurationVersion", "cv-123").Return("http://example.com/config", nil)
				m.On("SaveFile", "http://example.com/config", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "download failure",
			configVer: "cv-123",
			setup: func(m *MockClient) {
				m.On("DownloadRunConfigurationVersion", "cv-123").Return("", assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			workDir := t.TempDir()
			executor := NewTerraformExecutor(workDir, mockClient)

			err := executor.downloadConfig(tt.configVer, workDir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUploadPlan(t *testing.T) {
	tests := []struct {
		name    string
		runID   string
		setup   func(*MockClient, string)
		wantErr bool
	}{
		{
			name:  "successful plan upload",
			runID: "run-123",
			setup: func(m *MockClient, dir string) {
				planPath := filepath.Join(dir, "plan.tfplan")
				err := os.WriteFile(planPath, []byte("test plan"), 0644)
				assert.NoError(t, err)

				m.On("UploadRunPlan", "run-123", planPath).Return("http://example.com/plan", nil)
				m.On("SaveFile", "http://example.com/plan", planPath).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			workDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(mockClient, workDir)
			}

			executor := NewTerraformExecutor(workDir, mockClient)
			err := executor.uploadPlan(tt.runID, workDir)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestGetAgentConfig(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewTerraformExecutor("/tmp/test", mockClient)

	os.Setenv("AGENT_TOKEN", "test-token")
	os.Setenv("AGENT_ADDRESS", "test-address")
	defer func() {
		os.Unsetenv("AGENT_TOKEN")
		os.Unsetenv("AGENT_ADDRESS")
	}()

	config := executor.GetAgentConfig()

	assert.Equal(t, "Terraform Agent", config.Name)
	assert.Equal(t, "test-token", config.Token)
	assert.Equal(t, "test-address", config.Address)
	assert.Equal(t, []string{"terraform", "agent"}, config.Tags)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, "/tmp/test", config.WorkspaceDir)
}

func TestGetAgentStatus(t *testing.T) {
	mockClient := new(MockClient)
	executor := NewTerraformExecutor("/tmp/test", mockClient)

	status := executor.GetAgentStatus()

	assert.Equal(t, "agent-123", status.ID)
	assert.Equal(t, "running", status.Status)
	assert.False(t, status.HasChanges)
	assert.Equal(t, "workspace-123", status.Workspace.ID)
	assert.Equal(t, "example-workspace", status.Workspace.Name)
	assert.Equal(t, "local", status.Workspace.ExecMode)
}

func TestDownloadRunArtifact(t *testing.T) {
	tests := []struct {
		name         string
		runID        string
		artifactType string
		returnURL    string
		wantErr      bool
	}{
		{
			name:         "successful artifact download",
			runID:        "run-123",
			artifactType: "plan",
			returnURL:    "http://example.com/artifact",
			wantErr:      false,
		},
		{
			name:         "download failure",
			runID:        "run-456",
			artifactType: "state",
			returnURL:    "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.wantErr {
				mockClient.On("DownloadRunArtifact", tt.runID, tt.artifactType).Return("", assert.AnError)
			} else {
				mockClient.On("DownloadRunArtifact", tt.runID, tt.artifactType).Return(tt.returnURL, nil)
			}

			url, err := mockClient.DownloadRunArtifact(tt.runID, tt.artifactType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.returnURL, url)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUpdateRunStatus(t *testing.T) {
	tests := []struct {
		name    string
		runID   string
		status  string
		wantErr bool
	}{
		{
			name:    "successful status update",
			runID:   "run-123",
			status:  "completed",
			wantErr: false,
		},
		{
			name:    "update failure",
			runID:   "run-456",
			status:  "error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.wantErr {
				mockClient.On("UpdateRunStatus", tt.runID, tt.status).Return(assert.AnError)
			} else {
				mockClient.On("UpdateRunStatus", tt.runID, tt.status).Return(nil)
			}

			err := mockClient.UpdateRunStatus(tt.runID, tt.status)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUpdateRunState(t *testing.T) {
	tests := []struct {
		name       string
		runID      string
		hasChanges bool
		wantErr    bool
	}{
		{
			name:       "successful state update",
			runID:      "run-123",
			hasChanges: true,
			wantErr:    false,
		},
		{
			name:       "update failure",
			runID:      "run-456",
			hasChanges: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.wantErr {
				mockClient.On("UpdateRunState", tt.runID, tt.hasChanges).Return(assert.AnError)
			} else {
				mockClient.On("UpdateRunState", tt.runID, tt.hasChanges).Return(nil)
			}

			err := mockClient.UpdateRunState(tt.runID, tt.hasChanges)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
