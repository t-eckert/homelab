package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/t-eckert/homelab/spark/internal/config"
	"github.com/t-eckert/homelab/spark/internal/db"
	"github.com/t-eckert/homelab/spark/internal/k8s"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [spark-name]",
	Short: "Delete a spark dev environment",
	Long:  `Delete a spark dev environment and its associated database`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sparkName := args[0]
		ctx := context.Background()

		fmt.Printf("Deleting spark: %s\n", sparkName)

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Delete Kubernetes resources
		fmt.Println("Deleting Kubernetes resources...")
		k8sClient, err := k8s.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create k8s client: %w", err)
		}

		err = k8sClient.DeleteSpark(ctx, sparkName)
		if err != nil {
			return fmt.Errorf("failed to delete spark: %w", err)
		}
		fmt.Printf("✓ Kubernetes resources deleted\n")

		// Delete PostgreSQL database
		fmt.Println("Deleting PostgreSQL database...")
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

		err = dbClient.DeleteDatabase(sparkName)
		if err != nil {
			return fmt.Errorf("failed to delete database: %w", err)
		}
		fmt.Printf("✓ Database deleted\n")

		fmt.Printf("\n✓ Spark %s deleted successfully!\n", sparkName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
