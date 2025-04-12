package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/oleggorj/tfc-agent-oss/internal/api"
	"github.com/oleggorj/tfc-agent-oss/internal/models"
)

var execCommand = exec.CommandContext

type TerraformExecutor struct {
	workDir string
	client  api.APIClient
}

func NewTerraformExecutor(workDir string, client api.APIClient) *TerraformExecutor {
	return &TerraformExecutor{
		workDir: workDir,
		client:  client,
	}
}

func (e *TerraformExecutor) ExecuteRun(ctx context.Context, run *models.RunEvent) error {
	// Create workspace directory
	runDir := filepath.Join(e.workDir, run.ID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	defer os.RemoveAll(runDir)

	// Download configuration version
	if err := e.downloadConfig(run.ConfigVer, runDir); err != nil {
		return fmt.Errorf("download config: %w", err)
	}

	// Download state if exists
	if run.StateVer != "" {
		if err := e.downloadState(run.StateVer, runDir); err != nil {
			return fmt.Errorf("download state: %w", err)
		}
	}

	// Initialize Terraform
	if err := e.runTerraformCommand(ctx, runDir, "init", "-input=false"); err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	// Run plan or apply based on run type
	switch run.Status {
	case "planning":
		return e.runPlan(ctx, runDir, run)
	case "applying":
		return e.runApply(ctx, runDir, run)
	default:
		return fmt.Errorf("unknown run status: %s", run.Status)
	}
}

func (e *TerraformExecutor) runTerraformCommand(ctx context.Context, dir string, args ...string) error {
	cmd := execCommand(ctx, "terraform", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform command failed: %s: %w", string(output), err)
	}

	return nil
}

func (e *TerraformExecutor) runPlan(ctx context.Context, dir string, run *models.RunEvent) error {
	// Run terraform plan
	if err := e.runTerraformCommand(ctx, dir, "plan", "-out=plan.tfplan"); err != nil {
		return fmt.Errorf("terraform plan: %w", err)
	}

	// Upload the plan file
	if err := e.uploadPlan(run.ID, dir); err != nil {
		return fmt.Errorf("upload plan: %w", err)
	}

	return nil
}
func (e *TerraformExecutor) runApply(ctx context.Context, dir string, run *models.RunEvent) error {
	// Run terraform apply
	if err := e.runTerraformCommand(ctx, dir, "apply", "-auto-approve", "plan.tfplan"); err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}

	// Upload the state file
	if err := e.uploadState(run.ID, dir); err != nil {
		return fmt.Errorf("upload state: %w", err)
	}

	return nil
}
func (e *TerraformExecutor) downloadConfig(configVer string, dir string) error {
	// Download the configuration version file
	url, err := e.client.DownloadRunConfigurationVersion(configVer)
	if err != nil {
		return fmt.Errorf("download config version: %w", err)
	}

	// Save the file to the workspace directory
	filePath := filepath.Join(dir, "config.tf")
	if err := e.client.SaveFile(url, filePath); err != nil {
		return fmt.Errorf("save config file: %w", err)
	}

	return nil
}
func (e *TerraformExecutor) downloadState(stateVer string, dir string) error {
	// Download the state version file
	url, err := e.client.DownloadRunStateVersion(stateVer)
	if err != nil {
		return fmt.Errorf("download state version: %w", err)
	}

	// Save the file to the workspace directory
	filePath := filepath.Join(dir, "state.tfstate")
	if err := e.client.SaveFile(url, filePath); err != nil {
		return fmt.Errorf("save state file: %w", err)
	}

	return nil
}
func (e *TerraformExecutor) uploadPlan(runID string, dir string) error {
	// Upload the plan file
	filePath := filepath.Join(dir, "plan.tfplan")
	url, err := e.client.UploadRunPlan(runID, filePath)
	if err != nil {
		return fmt.Errorf("upload plan: %w", err)
	}

	// Save the URL to the workspace directory
	if err := e.client.SaveFile(url, filePath); err != nil {
		return fmt.Errorf("save plan URL: %w", err)
	}

	return nil
}
func (e *TerraformExecutor) uploadState(runID string, dir string) error {
	// Upload the state file
	filePath := filepath.Join(dir, "state.tfstate")
	url, err := e.client.UploadRunState(runID, filePath)
	if err != nil {
		return fmt.Errorf("upload state: %w", err)
	}

	// Save the URL to the workspace directory
	if err := e.client.SaveFile(url, filePath); err != nil {
		return fmt.Errorf("save state URL: %w", err)
	}

	return nil
}
func (e *TerraformExecutor) SetWorkDir(workDir string) {
	e.workDir = workDir
}
func (e *TerraformExecutor) GetWorkDir() string {
	return e.workDir
}
func (e *TerraformExecutor) SetClient(client api.APIClient) {
	e.client = client
}
func (e *TerraformExecutor) GetClient() api.APIClient {
	return e.client
}
func (e *TerraformExecutor) Execute(ctx context.Context, run *models.RunEvent) error {
	if err := e.ExecuteRun(ctx, run); err != nil {
		return fmt.Errorf("execute run: %w", err)
	}

	return nil
}
func (e *TerraformExecutor) Stop() error {
	// Implement stop logic if needed
	return nil
}
func (e *TerraformExecutor) Start() error {
	// Implement start logic if needed
	return nil
}
func (e *TerraformExecutor) Restart() error {
	// Implement restart logic if needed
	return nil
}
func (e *TerraformExecutor) Status() string {
	// Implement status logic if needed
	return "running"
}
func (e *TerraformExecutor) HealthCheck() error {
	// Implement health check logic if needed
	return nil
}
func (e *TerraformExecutor) Cleanup() error {
	// Implement cleanup logic if needed
	return nil
}
func (e *TerraformExecutor) GetAgentConfig() *models.AgentConfig {
	return &models.AgentConfig{
		Name:         "Terraform Agent",
		Token:        os.Getenv("AGENT_TOKEN"),
		Address:      os.Getenv("AGENT_ADDRESS"),
		Tags:         []string{"terraform", "agent"},
		LogLevel:     "info",
		WorkspaceDir: e.workDir,
	}
}
func (e *TerraformExecutor) SetAgentConfig(config *models.AgentConfig) {
	e.workDir = config.WorkspaceDir
}
func (e *TerraformExecutor) GetAgentStatus() *models.RunStatus {
	return &models.RunStatus{
		ID:         "agent-123",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     "running",
		HasChanges: false,
		Workspace: models.Workspace{
			ID:       "workspace-123",
			Name:     "example-workspace",
			ExecMode: "local",
		},
		ConfigVer: "config-123",
		StateVer:  "state-123",
	}
}
func (e *TerraformExecutor) GetAgentID() string {
	return "agent-123"
}
func (e *TerraformExecutor) GetAgentName() string {
	return "Terraform Agent"
}
func (e *TerraformExecutor) GetAgentTags() []string {
	return []string{"terraform", "agent"}
}
func (e *TerraformExecutor) GetAgentToken() string {
	return os.Getenv("AGENT_TOKEN")
}
func (e *TerraformExecutor) GetAgentAddress() string {
	return os.Getenv("AGENT_ADDRESS")
}
func (e *TerraformExecutor) GetAgentPort() int {
	return 8080
}
func (e *TerraformExecutor) GetAgentVersion() string {
	return "1.0.0"
}
func (e *TerraformExecutor) GetAgentOS() string {
	return "linux"
}
func (e *TerraformExecutor) GetAgentArch() string {
	return "amd64"
}
func (e *TerraformExecutor) GetAgentPlatform() string {
	return "linux/amd64"
}
func (e *TerraformExecutor) GetAgentEnvironment() string {
	return "production"
}
func (e *TerraformExecutor) GetAgentRegion() string {
	return "us-east-1"
}
func (e *TerraformExecutor) GetAgentZone() string {
	return "us-east-1a"
}
func (e *TerraformExecutor) GetAgentInstanceType() string {
	return "t2.micro"
}
func (e *TerraformExecutor) GetAgentInstanceID() string {
	return "i-1234567890abcdef0"
}
func (e *TerraformExecutor) GetAgentInstanceName() string {
	return "example-instance"
}
func (e *TerraformExecutor) GetAgentInstanceTags() map[string]string {
	return map[string]string{
		"Name": "example-instance",
	}
}
func (e *TerraformExecutor) GetAgentInstanceState() string {
	return "running"
}
func (e *TerraformExecutor) GetAgentInstancePublicIP() string {
	return ""
}
func (e *TerraformExecutor) GetAgentInstancePrivateIP() string {
	return ""
}
func (e *TerraformExecutor) GetAgentInstancePublicDNS() string {
	return ""
}
func (e *TerraformExecutor) GetAgentInstancePrivateDNS() string {
	return ""
}
func (e *TerraformExecutor) GetAgentInstanceLaunchTime() time.Time {
	return time.Now()
}
func (e *TerraformExecutor) GetAgentInstanceLaunchTimeString() string {
	return time.Now().Format(time.RFC3339)
}
func (e *TerraformExecutor) GetAgentInstanceLaunchTimeUnix() int64 {
	return time.Now().Unix()
}
func (e *TerraformExecutor) GetAgentInstanceLaunchTimeUnixNano() int64 {
	return time.Now().UnixNano()
}
func (e *TerraformExecutor) GetAgentInstanceLaunchTimeUnixMilli() int64 {
	return time.Now().UnixMilli()
}
func (e *TerraformExecutor) GetAgentInstanceLaunchTimeUnixMicro() int64 {
	return time.Now().UnixMicro()
}
func (e *TerraformExecutor) GetAgentInstanceLaunchTimeUnixNanoString() string {
	return time.Now().Format(time.RFC3339Nano)
}
