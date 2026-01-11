package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spark",
	Short: "Quick-deploy dev environments for one-off projects",
	Long: `Spark creates ephemeral development environments in your homelab cluster.

Each spark gets:
  - Random adjective-noun name (e.g., brave-dolphin)
  - Debian container with SSH, Claude Code, and your dotfiles
  - Tailscale connectivity for external access
  - Dedicated PostgreSQL database
  - Pre-configured environment variables (DATABASE_URL, ANTHROPIC_API_KEY)
  - Optional git repository cloning

Commands:
  create - Create a new spark
  list   - List all active sparks
  shell  - SSH into an existing spark
  delete - Destroy a spark and its database

Examples:
  spark create                     # Create a new spark
  spark create --repo https://...  # Create with git repo
  spark list                       # List all sparks
  spark shell brave-dolphin        # SSH into a spark
  spark delete brave-dolphin       # Delete a spark`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}
