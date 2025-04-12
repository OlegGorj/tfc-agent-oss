package models

import "time"

type AgentConfig struct {
	Name         string   `json:"name"`
	Token        string   `json:"token"`
	Address      string   `json:"address"`
	Tags         []string `json:"tags"`
	LogLevel     string   `json:"log_level"`
	WorkspaceDir string   `json:"workspace_dir"`
	AgentID      string   `json:"agent_id,omitempty"`      // Added for tracking
	AgentPoolID  string   `json:"agent_pool_id,omitempty"` // Optional
}

type RunEvent struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Attributes    RunEventAttributes    `json:"attributes"`
	Relationships RunEventRelationships `json:"relationships"`
}

type RunEventAttributes struct {
	Status     string    `json:"status"`
	HasChanges bool      `json:"has_changes"`
	CreatedAt  time.Time `json:"created_at"`
	StateVer   string    `json:"state_version"`
}

type RunEventRelationships struct {
	Workspace            Relationship `json:"workspace"`
	ConfigurationVersion Relationship `json:"configuration_version"`
	StateVersion         Relationship `json:"state_version"`
}

type Relationship struct {
	Data RelationshipData `json:"data"`
}

type RelationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type Workspace struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ExecMode string `json:"execution_mode"`
}

type ConfigurationVersion struct {
	ID          string `json:"id"`
	DownloadURL string `json:"download_url"`
}

type StateVersion struct {
	ID          string `json:"id"`
	DownloadURL string `json:"download_url"`
}

type Artifact struct {
	ID          string `json:"id"`
	DownloadURL string `json:"download_url"`
}

type RunStatus struct {
	ID                              string    `json:"id"`
	CreatedAt                       time.Time `json:"created_at"`
	UpdatedAt                       time.Time `json:"updated_at"`
	Status                          string    `json:"status"`
	HasChanges                      bool      `json:"has_changes"`
	Workspace                       Workspace `json:"workspace"`
	ConfigVer                       string    `json:"configuration_version"`
	StateVer                        string    `json:"state_version"`
	Agent                           string    `json:"agent"`
	AgentID                         string    `json:"agent_id"`
	AgentName                       string    `json:"agent_name"`
	AgentTags                       []string  `json:"agent_tags"`
	AgentToken                      string    `json:"agent_token"`
	AgentAddr                       string    `json:"agent_address"`
	AgentPort                       int       `json:"agent_port"`
	AgentState                      string    `json:"agent_state"`
	AgentVersion                    string    `json:"agent_version"`
	AgentOS                         string    `json:"agent_os"`
	AgentArch                       string    `json:"agent_arch"`
	AgentOSVersion                  string    `json:"agent_os_version"`
	AgentKernelVersion              string    `json:"agent_kernel_version"`
	AgentKernelArch                 string    `json:"agent_kernel_arch"`
	AgentKernelName                 string    `json:"agent_kernel_name"`
	AgentKernelRelease              string    `json:"agent_kernel_release"`
	AgentKernelVersionID            string    `json:"agent_kernel_version_id"`
	AgentKernelVersionName          string    `json:"agent_kernel_version_name"`
	AgentKernelVersionRelease       string    `json:"agent_kernel_version_release"`
	AgentKernelVersionArch          string    `json:"agent_kernel_version_arch"`
	AgentKernelVersionOS            string    `json:"agent_kernel_version_os"`
	AgentKernelVersionOSVersion     string    `json:"agent_kernel_version_os_version"`
	AgentKernelVersionOSRelease     string    `json:"agent_kernel_version_os_release"`
	AgentKernelVersionOSArch        string    `json:"agent_kernel_version_os_arch"`
	AgentKernelVersionOSName        string    `json:"agent_kernel_version_os_name"`
	AgentKernelVersionOSNameID      string    `json:"agent_kernel_version_os_name_id"`
	AgentKernelVersionOSNameVersion string    `json:"agent_kernel_version_os_name_version"`
	AgentKernelVersionOSNameRelease string    `json:"agent_kernel_version_os_name_release"`
	AgentKernelVersionOSNameArch    string    `json:"agent_kernel_version_os_name_arch"`
}

type RunEventResponse struct {
	ID                              string    `json:"id"`
	CreatedAt                       time.Time `json:"created_at"`
	UpdatedAt                       time.Time `json:"updated_at"`
	Status                          string    `json:"status"`
	HasChanges                      bool      `json:"has_changes"`
	Workspace                       Workspace `json:"workspace"`
	ConfigVer                       string    `json:"configuration_version"`
	StateVer                        string    `json:"state_version"`
	Agent                           string    `json:"agent"`
	AgentID                         string    `json:"agent_id"`
	AgentName                       string    `json:"agent_name"`
	AgentTags                       []string  `json:"agent_tags"`
	AgentToken                      string    `json:"agent_token"`
	AgentAddr                       string    `json:"agent_address"`
	AgentPort                       int       `json:"agent_port"`
	AgentState                      string    `json:"agent_state"`
	AgentVersion                    string    `json:"agent_version"`
	AgentOS                         string    `json:"agent_os"`
	AgentArch                       string    `json:"agent_arch"`
	AgentOSVersion                  string    `json:"agent_os_version"`
	AgentKernelVersion              string    `json:"agent_kernel_version"`
	AgentKernelArch                 string    `json:"agent_kernel_arch"`
	AgentKernelName                 string    `json:"agent_kernel_name"`
	AgentKernelRelease              string    `json:"agent_kernel_release"`
	AgentKernelVersionID            string    `json:"agent_kernel_version_id"`
	AgentKernelVersionName          string    `json:"agent_kernel_version_name"`
	AgentKernelVersionRelease       string    `json:"agent_kernel_version_release"`
	AgentKernelVersionArch          string    `json:"agent_kernel_version_arch"`
	AgentKernelVersionOS            string    `json:"agent_kernel_version_os"`
	AgentKernelVersionOSVersion     string    `json:"agent_kernel_version_os_version"`
	AgentKernelVersionOSRelease     string    `json:"agent_kernel_version_os_release"`
	AgentKernelVersionOSArch        string    `json:"agent_kernel_version_os_arch"`
	AgentKernelVersionOSName        string    `json:"agent_kernel_version_os_name"`
	AgentKernelVersionOSNameID      string    `json:"agent_kernel_version_os_name_id"`
	AgentKernelVersionOSNameVersion string    `json:"agent_kernel_version_os_name_version"`
	AgentKernelVersionOSNameRelease string    `json:"agent_kernel_version_os_name_release"`
	AgentKernelVersionOSNameArch    string    `json:"agent_kernel_version_os_name_arch"`
}
