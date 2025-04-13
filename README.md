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
export TF_ORGANIZATION="your-org-name"      # Required for agent pool lookup
export TF_AGENT_POOL_ID="apool-xxx"        # Optional: specify pool directly

./bin/tfc-agent-oss
```

Note: You must either:
1. Set `TF_ORGANIZATION` to let the agent find the default pool
2. Set `TF_AGENT_POOL_ID` to specify the pool directly

# TFE Agent registration flow

1. Create new auth token

Create new authentication token for a given agents pool using endpoint `POST /api/v2/agent-pools/<pool_id>/authentication-tokens`
```
POST https://app.terraform.io/api/v2/agent-pools/apool-xxx/authentication-tokens HTTP/2.0
authorization: Bearer <token>
content-type: application/json
accept-encoding: gzip
{
    "description": "myagent"
}
```


2. The agent starts and registers with the TFC API using the provided token.
```
POST https://app.terraform.io/api/agent/register HTTP/2.0
tfc-agent-version: 1.14.5
content-type: application/json
authorization: Bearer <token>
{
    "name": "myagent"
}
```

Response:
```
HTTP/2.0 200
content-type: application/json; charset=utf-8
{
    "id": "agent-aFGb8twYmYuv1GvN",
    "pool_id": "apool-ybQhCPkx8A59eJ3J"
}
```


3. Sets Agent's status
```
PUT https://app.terraform.io/api/agent/status HTTP/2.0
user-agent: tfc-agent/1.14.5
authorization: Bearer <token>
tfc-agent-version: 1.14.5
tfc-agent-id: agent-aFGb8twYmYuv1GvN
content-type: application/json
accept-encoding: gzip
{
    "status": "idle"
}
```

Response:
```
HTTP/2.0 204
```

4. The agent starts listening for jobs and receives a job from the TFC API.
```
GET https://app.terraform.io/api/agent/jobs HTTP/2.0
user-agent: tfc-agent/1.14.5
```

