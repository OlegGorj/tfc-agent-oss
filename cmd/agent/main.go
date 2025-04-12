package main

import (
	"context"
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
	cfg := &models.AgentConfig{
		Name:         os.Getenv("TF_AGENT_NAME"),
		Token:        os.Getenv("TF_API_TOKEN"),
		Address:      os.Getenv("TF_ADDRESS"),
		WorkspaceDir: os.Getenv("TF_WORKSPACE_DIR"),
		Tags:         []string{"self-hosted", "custom"},
	}

	if cfg.Token == "" {
		log.Fatal("TF_API_TOKEN environment variable is required")
	}

	client := api.NewClient(cfg.Address, cfg.Token)
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
	if err := client.RegisterAgent(cfg); err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}

	// Main polling loop
	for {
		select {
		case <-ctx.Done():
			return
		default:
			run, err := client.PollForRuns()
			if err != nil {
				log.Printf("Error polling for runs: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			if run == nil {
				time.Sleep(5 * time.Second)
				continue
			}

			if err := executor.ExecuteRun(ctx, run); err != nil {
				log.Printf("Error executing run %s: %v", run.ID, err)
			}
		}
	}
}
