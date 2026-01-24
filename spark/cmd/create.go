package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/t-eckert/homelab/spark/internal/config"
	"github.com/t-eckert/homelab/spark/internal/db"
	"github.com/t-eckert/homelab/spark/internal/k8s"
	"github.com/t-eckert/homelab/spark/internal/names"
)

var gitRepo string

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new spark dev environment",
	Long: `Create a new spark dev environment with:
- Random adjective-noun name
- Debian container with SSH, Claude Code, and dotfiles
- Tailscale connectivity
- Dedicated PostgreSQL database
- Pre-configured environment variables`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Generate random name
		sparkName := names.Generate()
		fmt.Printf("Creating spark: %s\n", sparkName)

		// Create PostgreSQL database
		fmt.Println("Creating PostgreSQL database...")
		dbConnString := db.BuildConnectionString(
			cfg.PostgresHost,
			cfg.PostgresPort,
			cfg.PostgresUser,
			cfg.PostgresDB,
		)
		dbClient, err := db.NewClient(dbConnString, cfg.PostgresPassword)
		if err != nil {
			return fmt.Errorf("failed to connect to postgres: %w", err)
		}
		defer dbClient.Close()

		err = dbClient.CreateDatabase(sparkName)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		fmt.Printf("Database created: %s\n", sparkName)

		// Build database URL for the spark (URI format with password for container use)
		sparkDBURL := db.BuildConnectionURI(
			cfg.PostgresHost,
			cfg.PostgresPort,
			cfg.PostgresUser,
			cfg.PostgresPassword,
			sparkName,
		)

		// Create Kubernetes resources
		fmt.Println("Creating Kubernetes resources...")
		k8sClient, err := k8s.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create k8s client: %w", err)
		}

		resources := &k8s.SparkResources{
			Name:            sparkName,
			GitRepo:         gitRepo,
			DatabaseURL:     sparkDBURL,
			AnthropicAPIKey: cfg.AnthropicAPIKey,
			SSHPublicKey:    cfg.SSHPublicKey,
			GitHubToken:     cfg.GitHubToken,
		}

		err = k8sClient.CreateSpark(ctx, resources)
		if err != nil {
			// Clean up database if k8s creation fails
			_ = dbClient.DeleteDatabase(sparkName)
			return fmt.Errorf("failed to create spark: %w", err)
		}

		fmt.Printf("Spark created successfully!\n")
		fmt.Printf("\nWaiting for pod to be ready...\n")

		// Wait for pod to be ready
		var podReady bool
		for i := 0; i < 60; i++ {
			pod, err := k8sClient.GetSparkPod(ctx, sparkName)
			if err == nil && pod.Status.Phase == "Running" {
				podReady = true
				break
			}
			time.Sleep(2 * time.Second)
			fmt.Print(".")
		}
		fmt.Println()

		if !podReady {
			fmt.Printf("\nSpark created but pod is not ready yet.\n")
			fmt.Printf("You can connect later with: spark shell %s\n", sparkName)
			return nil
		}

		fmt.Printf("✓ Pod is ready!\n")
		fmt.Printf("\nSpark Details:\n")
		fmt.Printf("  Name:     %s\n", sparkName)
		fmt.Printf("  Database: %s\n", sparkName)
		fmt.Printf("  SSH:      ssh user@spark-%s\n", sparkName)
		if gitRepo != "" {
			fmt.Printf("  Git Repo: %s\n", gitRepo)
		}

		fmt.Printf("\nConnecting to spark...\n")

		// SSH into the spark
		sshCmd := exec.Command("ssh", fmt.Sprintf("user@spark-%s", sparkName))
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		err = sshCmd.Run()
		if err != nil {
			fmt.Printf("\nFailed to SSH into spark: %v\n", err)
			fmt.Printf("You can try connecting manually with: ssh user@spark-%s\n", sparkName)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&gitRepo, "repo", "r", "", "Git repository to clone into the spark")
}
