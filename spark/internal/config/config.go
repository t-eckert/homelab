package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	AnthropicAPIKey  string
	GitHubToken      string
	SSHPublicKey     string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
}

// Load reads configuration from environment variables and returns a Config struct.
func Load() (*Config, error) {
	cfg := &Config{
		AnthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		GitHubToken:      os.Getenv("GITHUB_TOKEN"),
		PostgresHost:     getEnvOrDefault("POSTGRES_HOST", "postgres.postgres.svc.cluster.local"),
		PostgresPort:     getEnvOrDefault("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnvOrDefault("POSTGRES_USER", "spark"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       getEnvOrDefault("POSTGRES_DB", "homelab"),
	}

	// Load SSH public key
	sshKeyPath := getEnvOrDefault("SSH_PUBLIC_KEY_PATH", filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519.pub"))
	sshKey, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH public key from %s: %w", sshKeyPath, err)
	}
	cfg.SSHPublicKey = string(sshKey)

	// Validate required fields
	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	if cfg.PostgresPassword == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD environment variable is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
