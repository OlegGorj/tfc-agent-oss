# tfc-agent-oss

# Terraform Cloud Agent OSS
Terraform Cloud Agent OSS is an open-source agent for Terraform Cloud that allows you to run Terraform jobs on your own infrastructure. It is designed to be lightweight and easy to use, making it a great choice for teams that want to run Terraform jobs in their own environment.

## This implementation

Follows TFC's API specifications
Implements proper agent registration
Handles workspace isolation
Supports configuration downloads
Manages state versions
Provides real-time logging
Handles graceful shutdown
The agent is compatible with Terraform Cloud's agent pools and can be used as a drop-in replacement for the official agent while maintaining full control over the execution environment.

##  Key features

TFC API compatibility
Secure authentication
Configuration version handling
State management
Run lifecycle management
Workspace isolation
Error handling and reporting


# Installation

```bash
cd tfc-agent-oss
go mod tidy
go build -o bin/tfc-agent-oss cmd/agent/main.go
```

# Run the agent

```bash
export TF_API_TOKEN="your-token"
export TF_ADDRESS="https://app.terraform.io"
export TF_AGENT_NAME="custom-agent-1"
export TF_WORKSPACE_DIR="/tmp/terraform-runs"

./bin/tfc-agent-oss
```

