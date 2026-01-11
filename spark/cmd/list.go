package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/t-eckert/homelab/spark/internal/k8s"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active sparks",
	Long:  `List all currently running spark dev environments`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Create Kubernetes client
		k8sClient, err := k8s.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create k8s client: %w", err)
		}

		// List all sparks
		sparks, err := k8sClient.ListSparks(ctx)
		if err != nil {
			return fmt.Errorf("failed to list sparks: %w", err)
		}

		if len(sparks) == 0 {
			fmt.Println("No sparks found")
			return nil
		}

		fmt.Printf("Active sparks (%d):\n\n", len(sparks))
		for _, sparkName := range sparks {
			// Get deployment to check status
			deployment, err := k8sClient.GetDeployment(ctx, sparkName)
			if err != nil {
				fmt.Printf("  - %s (error getting status)\n", sparkName)
				continue
			}

			status := "Not Ready"
			if deployment.Status.ReadyReplicas > 0 {
				status = "Ready"
			}

			fmt.Printf("  - %s (%s)\n", sparkName, status)
			fmt.Printf("    SSH: ssh user@spark-%s\n", sparkName)
			fmt.Printf("    Database: %s\n", sparkName)
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
