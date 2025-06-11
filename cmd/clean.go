package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/davidroman0O/steam-workshop-downloader/pkg/steamcmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean workshop cache to fix SteamCMD errors",
	Long: `Clean workshop cache directories to fix CWorkThreadPool errors.

This command removes temporary files and cache directories that can cause
SteamCMD to hang with errors like:
  CWorkThreadPool::~CWorkThreadPool: work complete queue not empty, X items discarded

The command will clean:
- Workshop downloads folder
- Workshop temp folder
- Workshop content folder (if --all flag is used)

Use --force to skip confirmation prompt.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanWorkshop()
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().BoolP("force", "f", false, "Force clean without confirmation prompt")
	cleanCmd.Flags().BoolP("all", "a", false, "Also remove downloaded workshop content (not just cache)")
	viper.BindPFlag("force_clean", cleanCmd.Flags().Lookup("force"))
	viper.BindPFlag("clean_all", cleanCmd.Flags().Lookup("all"))
}

func cleanWorkshop() error {
	// Create SteamCMD client to get paths
	steamcmdDir := viper.GetString("steamcmd_dir")
	client, err := steamcmd.NewClient(steamcmdDir)
	if err != nil {
		return fmt.Errorf("failed to create SteamCMD client: %w", err)
	}

	// Get all workshop cache paths
	cachePaths := client.GetWorkshopCachePaths()

	if len(cachePaths) == 0 {
		fmt.Println("No workshop cache directories found to clean.")
		return nil
	}

	// Check what actually exists
	var existingPaths []string
	for _, path := range cachePaths {
		if _, err := os.Stat(path); err == nil {
			existingPaths = append(existingPaths, path)
		}
	}

	if len(existingPaths) == 0 {
		fmt.Println("No workshop cache directories found to clean.")
		return nil
	}

	// Show what will be cleaned
	fmt.Println("The following workshop cache directories will be removed:")
	for _, path := range existingPaths {
		fmt.Printf("  - %s\n", path)
	}
	fmt.Println()

	// If --all flag is used, also show content directories
	cleanAll := viper.GetBool("clean_all")
	if cleanAll {
		fmt.Println("⚠️  --all flag used: Downloaded workshop content will also be removed!")
		fmt.Println("   You will need to re-download any workshop items.")
		fmt.Println()
	}

	// Ask for confirmation unless --force is used
	force := viper.GetBool("force_clean")
	if !force {
		fmt.Print("Are you sure you want to continue? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Clean operation cancelled.")
			return nil
		}
	}

	// Clean the directories
	var removedCount int
	var errors []string

	for _, path := range existingPaths {
		// Skip content directories unless --all is used
		if !cleanAll && strings.Contains(path, "content") {
			continue
		}

		fmt.Printf("Removing %s...\n", path)
		if err := os.RemoveAll(path); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to remove %s: %v", path, err))
		} else {
			removedCount++
		}
	}

	// Report results
	if len(errors) > 0 {
		fmt.Printf("\n❌ Completed with %d errors:\n", len(errors))
		for _, errMsg := range errors {
			fmt.Printf("  %s\n", errMsg)
		}
	}

	if removedCount > 0 {
		fmt.Printf("\n✅ Successfully cleaned %d workshop cache directories.\n", removedCount)
		fmt.Println("This should fix CWorkThreadPool errors in SteamCMD.")
	}

	return nil
}
