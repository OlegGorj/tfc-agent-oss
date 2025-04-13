package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oleggorj/tfc-agent-oss/internal/api"
	"github.com/oleggorj/tfc-agent-oss/internal/executor"
	"github.com/oleggorj/tfc-agent-oss/internal/models"
)

func main() {
	randomSuffix := fmt.Sprintf("%08x", time.Now().UnixNano())[0:8]
	cfg := &models.AgentConfig{
		Name:         fmt.Sprintf("%s-%s", getEnvOrDefault("TF_AGENT_NAME", "tfc-agent-oss"), randomSuffix),
		Token:        os.Getenv("TF_API_TOKEN"),
		Address:      getEnvOrDefault("TF_ADDRESS", "https://app.terraform.io"),
		WorkspaceDir: getEnvOrDefault("TF_WORKSPACE_DIR", "/tmp/terraform-runs"),
		Tags:         []string{"self-hosted", "custom"},
		LogLevel:     getEnvOrDefault("TF_LOG_LEVEL", "info"),
		AgentPoolID:  os.Getenv("TF_AGENT_POOL_ID"), // Optional pool assignment
	}
	if cfg.Token == "" {
		log.Fatal("TF_API_TOKEN environment variable is required")
	}
	// Initialize API client
	client := api.NewClient(cfg.Address, cfg.Token)
	// Initialize workspace directory
	if err := os.MkdirAll(cfg.WorkspaceDir, 0755); err != nil {
		log.Fatalf("Failed to create workspace directory: %v", err)
	}

	// Add debug logging
	log.Printf("Starting agent with config:")
	log.Printf("Name: %s", cfg.Name)
	log.Printf("Address: %s", cfg.Address)
	log.Printf("Workspace Dir: %s", cfg.WorkspaceDir)
	log.Printf("Agent Pool ID: %s", cfg.AgentPoolID)
	log.Printf("Organization: %s", os.Getenv("TF_ORGANIZATION"))

	// Register agent with retries
	log.Printf("Registering agent %s...", cfg.Name)
	if err := client.RegisterAgent(cfg); err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}
	log.Printf("Agent registered successfully with ID: %s", cfg.AgentID)
	log.Printf("Agent token received: %s", cfg.Token[:8]+"...")

	executor := executor.NewTerraformExecutor(cfg.WorkspaceDir, client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Register agent
	// if err := client.RegisterAgent(cfg); err != nil {
	// 	log.Fatalf("Failed to register agent: %v", err)
	// }

	// Main polling loop
	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Printf("Polling for runs...")
			run, err := client.PollForRuns(cfg)
			if err != nil {
				log.Printf("Error polling for runs: %v", err)
				if os.Getenv("TF_LOG") == "debug" {
					log.Printf("Debug details: %+v", err)
				}
				time.Sleep(10 * time.Second)
				continue
			}

			if run == nil {
				if os.Getenv("TF_LOG") == "debug" {
					log.Printf("No runs available")
				}
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("Executing run %s...", run.ID)
			if err := executor.ExecuteRun(ctx, run); err != nil {
				log.Printf("Error executing run %s: %v", run.ID, err)
			}
		}
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
