package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/t-eckert/homelab/spark/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

var shellCmd = &cobra.Command{
	Use:   "shell [spark-name]",
	Short: "SSH into an existing spark",
	Long:  `Open an SSH connection to an existing spark dev environment`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sparkName := args[0]
		ctx := context.Background()

		// Create Kubernetes client
		k8sClient, err := k8s.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create k8s client: %w", err)
		}

		// Check if spark exists and is ready
		pod, err := k8sClient.GetSparkPod(ctx, sparkName)
		if err != nil {
			return fmt.Errorf("spark not found or not ready: %w", err)
		}

		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("spark %s is not running (status: %s)", sparkName, pod.Status.Phase)
		}

		fmt.Printf("Connecting to spark: %s\n", sparkName)

		// SSH into the spark
		sshCmd := exec.Command("ssh", fmt.Sprintf("user@spark-%s", sparkName))
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		err = sshCmd.Run()
		if err != nil {
			return fmt.Errorf("failed to SSH into spark: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
