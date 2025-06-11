package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/davidroman0O/steam-workshop-downloader/pkg/steamcmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Steam interactively (handles Steam Guard)",
	Long: `Launch SteamCMD interactively to login to Steam.
This allows you to handle Steam Guard authentication naturally.
Once logged in, your credentials are stored for future downloads.

After running this command:
1. SteamCMD will start with a Steam> prompt
2. Type: login yourusername
3. Enter your password when prompted
4. Enter Steam Guard code if requested
5. Type: quit

Your authentication will be stored for future downloads.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return launchInteractiveSteamCMD()
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func launchInteractiveSteamCMD() error {
	// Create SteamCMD client to get the path
	steamcmdDir := viper.GetString("steamcmd_dir")
	client, err := steamcmd.NewClient(steamcmdDir)
	if err != nil {
		return fmt.Errorf("failed to create SteamCMD client: %w", err)
	}

	fmt.Println("🚀 Launching SteamCMD for interactive login...")
	fmt.Println()
	fmt.Println("Instructions:")
	fmt.Println("1. At the Steam> prompt, type: login yourusername")
	fmt.Println("2. Enter your password when prompted")
	fmt.Println("3. If Steam Guard is enabled, check your email and enter the code")
	fmt.Println("4. Once logged in successfully, type: quit")
	fmt.Println("5. Your authentication will be stored for future downloads")
	fmt.Println()
	fmt.Printf("Starting SteamCMD at: %s\n", client.SteamCMDPath)
	fmt.Println()

	// Launch SteamCMD interactively
	cmd := exec.Command(client.SteamCMDPath)
	cmd.Dir = client.WorkingDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("SteamCMD execution failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ SteamCMD session completed!")
	fmt.Println("If you logged in successfully, you can now download workshop items without authentication.")
	return nil
}
